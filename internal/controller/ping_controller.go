package controller

import (
	"fmt"
	"sync"
	"time"

	"gossh/internal/config"
	"gossh/internal/executor"
	"gossh/internal/ssh"
	"gossh/internal/view"
)

// PingController 处理 ping 命令的业务逻辑
type PingController struct{}

// NewPingController 创建新的 PingController
func NewPingController() *PingController {
	return &PingController{}
}

// PingRequest ping 命令的请求参数
type PingRequest struct {
	ConfigFile  string // ansible.cfg 配置文件路径
	Inventory   string // 主机列表（文件路径、目录路径或逗号分隔的主机列表）
	Group       string // Ansible INI 格式的分组名称
	User        string
	KeyPath     string
	Password    string
	Port        string
	Concurrency int
	Timeout     time.Duration // 连接超时时间
}

// PingResponse ping 命令的响应
type PingResponse struct {
	Results       []*ssh.PingResult
	TotalDuration time.Duration  // 总执行时间（从开始到所有任务完成）
	Group         string         // 分组名称（用户指定的）
	Hosts         []executor.Host // 主机列表（包含分组信息）
}

// Execute 执行 ping 命令
func (c *PingController) Execute(req *PingRequest) (*PingResponse, error) {
	// 合并配置（优先级：命令行参数 > ansible.cfg > 默认值）
	mergedReq := c.mergeConfig(req)

	// 打印当前配置参数
	view.PrintPingConfig(
		mergedReq.Inventory,
		mergedReq.Group,
		mergedReq.User,
		mergedReq.KeyPath,
		mergedReq.Password,
		mergedReq.Port,
		mergedReq.Concurrency,
		mergedReq.Timeout,
	)

	// 验证参数
	if err := c.validateRequest(mergedReq); err != nil {
		return nil, err
	}

	// 加载主机列表（已经包含分组信息）
	hosts, err := c.loadHosts(mergedReq)
	if err != nil {
		return nil, err
	}

	// 设置默认端口
	port := mergedReq.Port
	if port == "" {
		port = "22"
	}

	// 创建进度跟踪器
	progressTracker := view.NewProgressTracker(len(hosts), "SSH 连接测试")

	// 记录开始时间
	startTime := time.Now()

	// 执行 ping 测试（超时时间已在 mergeConfig 中处理）
	results, err := c.executePing(hosts, mergedReq.User, mergedReq.KeyPath, mergedReq.Password, port, mergedReq.Concurrency, mergedReq.Timeout, progressTracker)
	if err != nil {
		progressTracker.Stop()
		return nil, fmt.Errorf("执行失败: %w", err)
	}

	// 记录结束时间并计算总耗时
	totalDuration := time.Since(startTime)

	// 停止进度跟踪器
	progressTracker.Stop()

	return &PingResponse{
		Results:       results,
		TotalDuration: totalDuration,
		Group:         mergedReq.Group,
		Hosts:         hosts,
	}, nil
}

// mergeConfig 合并配置（优先级：命令行参数 > ansible.cfg > 默认值）
func (c *PingController) mergeConfig(req *PingRequest) *PingRequest {
	commonCfg := MergeCommonConfig(&CommonConfig{
		ConfigFile:  req.ConfigFile,
		Inventory:   req.Inventory,
		Group:       req.Group,
		User:        req.User,
		KeyPath:     req.KeyPath,
		Password:    req.Password,
		Port:        req.Port,
		Concurrency: req.Concurrency,
	})

	// 合并超时配置（优先级：命令行参数 > ansible.cfg > 默认值）
	timeout := req.Timeout
	if timeout <= 0 {
		// 如果命令行未指定，尝试从 ansible.cfg 读取
		ansibleCfg, err := config.LoadAnsibleConfig(req.ConfigFile)
		if err == nil && ansibleCfg.Timeout > 0 {
			// ansible.cfg 中的 timeout 单位是秒
			timeout = time.Duration(ansibleCfg.Timeout) * time.Second
		} else {
			// 默认值：30 秒
			timeout = 30 * time.Second
		}
	}

	return &PingRequest{
		ConfigFile:  req.ConfigFile,
		Inventory:   commonCfg.Inventory,
		Group:       commonCfg.Group,
		User:        commonCfg.User,
		KeyPath:     commonCfg.KeyPath,
		Password:    commonCfg.Password,
		Port:        commonCfg.Port,
		Concurrency: commonCfg.Concurrency,
		Timeout:     timeout,
	}
}

// validateRequest 验证请求参数
func (c *PingController) validateRequest(req *PingRequest) error {
	// 主机列表可以从命令行参数或 ansible.cfg 加载，所以这里不强制要求

	if req.User == "" {
		return fmt.Errorf("必须指定用户名（-u 或 ansible.cfg 中的 remote_user）")
	}

	return nil
}

// loadHosts 加载主机列表
func (c *PingController) loadHosts(req *PingRequest) ([]executor.Host, error) {
	return LoadHosts(&CommonConfig{
		ConfigFile: req.ConfigFile,
		Inventory:  req.Inventory,
		Group:      req.Group,
	}, true)
}


// executePing 并发执行 ping 测试
func (c *PingController) executePing(hosts []executor.Host, user, keyPath, password, defaultPort string, concurrency int, timeout time.Duration, progressTracker *view.ProgressTracker) ([]*ssh.PingResult, error) {
	if concurrency <= 0 {
		concurrency = 5
	}

	results := make([]*ssh.PingResult, len(hosts))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
	var mu sync.Mutex

	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, h executor.Host) {
			// 确保 wg.Done() 总是被调用，即使发生 panic
			defer func() {
				if r := recover(); r != nil {
					// 捕获 panic，记录错误并确保资源释放
					mu.Lock()
					if results[idx] == nil {
						results[idx] = &ssh.PingResult{
							Host:     h.Address,
							Success:  false,
							Duration: 0,
							Error:    fmt.Errorf("panic: %v", r),
						}
					}
					mu.Unlock()
					progressTracker.MarkTrackerErrored(h.Address, fmt.Sprintf("panic: %v", r))
				}
				wg.Done()
			}()

			hostAddr := h.Address

			// 为主机创建 tracker
			progressTracker.AddTracker(hostAddr)
			progressTracker.UpdateTracker(hostAddr, 10, fmt.Sprintf("%s (连接中...)", hostAddr))

			// 限制并发数
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 使用主机特定的配置，如果没有则使用默认配置
			hostKeyPath := h.KeyPath
			if hostKeyPath == "" {
				hostKeyPath = keyPath
			}
			hostUser := h.User
			if hostUser == "" {
				hostUser = user
			}
			port := h.Port
			if port == "" {
				port = defaultPort
			}

			progressTracker.UpdateTracker(hostAddr, 30, fmt.Sprintf("%s (创建客户端...)", hostAddr))
			// 使用带超时的客户端创建方法
			client, err := ssh.NewClientWithTimeout(h.Address, port, hostUser, hostKeyPath, password, timeout)
			if err != nil {
				mu.Lock()
				results[idx] = &ssh.PingResult{
					Host:     h.Address,
					Success:  false,
					Duration: 0,
					Error:    fmt.Errorf("创建客户端失败: %v", err),
				}
				mu.Unlock()
				progressTracker.MarkTrackerErrored(hostAddr, fmt.Sprintf("连接失败: %v", err))
				return
			}

			progressTracker.UpdateTracker(hostAddr, 60, fmt.Sprintf("%s (测试连接...)", hostAddr))
			// 使用带超时的 Ping 方法
			result, err := client.PingWithTimeout(timeout)
			if err != nil {
				mu.Lock()
				results[idx] = &ssh.PingResult{
					Host:     h.Address,
					Success:  false,
					Duration: 0,
					Error:    err,
				}
				mu.Unlock()
				progressTracker.MarkTrackerErrored(hostAddr, fmt.Sprintf("测试失败: %v", err))
				return
			}

			mu.Lock()
			results[idx] = result
			mu.Unlock()

			if result.Success {
				progressTracker.UpdateTracker(hostAddr, 100, hostAddr)
				progressTracker.MarkTrackerDone(hostAddr)
			} else {
				progressTracker.MarkTrackerErrored(hostAddr, "连接失败")
			}
		}(i, host)
	}

	wg.Wait()
	return results, nil
}
