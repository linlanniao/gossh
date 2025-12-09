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
	HostsFile   string
	HostsDir    string // Ansible hosts 目录路径
	HostsString string
	Group       string // Ansible INI 格式的分组名称
	User        string
	KeyPath     string
	Password    string
	Port        string
	Concurrency int
}

// PingResponse ping 命令的响应
type PingResponse struct {
	Results       []*ssh.PingResult
	TotalDuration time.Duration // 总执行时间（从开始到所有任务完成）
}

// Execute 执行 ping 命令
func (c *PingController) Execute(req *PingRequest) (*PingResponse, error) {
	// 合并配置（优先级：命令行参数 > ansible.cfg > 默认值）
	mergedReq := c.mergeConfig(req)

	// 打印当前配置参数
	view.PrintPingConfig(
		mergedReq.HostsFile,
		mergedReq.HostsDir,
		mergedReq.HostsString,
		mergedReq.Group,
		mergedReq.User,
		mergedReq.KeyPath,
		mergedReq.Password,
		mergedReq.Port,
		mergedReq.Concurrency,
	)

	// 验证参数
	if err := c.validateRequest(mergedReq); err != nil {
		return nil, err
	}

	// 加载主机列表
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

	// 创建进度回调函数
	progressCallback := func(host string, stage string, value int64, isFailed bool) {
		if value == 100 {
			// 任务完成（成功或失败）
			if isFailed {
				progressTracker.AddFailedHost(host, stage)
			}
			progressTracker.Increment()
		}
	}

	// 记录开始时间
	startTime := time.Now()

	// 执行 ping 测试
	results, err := c.executePing(hosts, mergedReq.User, mergedReq.KeyPath, mergedReq.Password, port, mergedReq.Concurrency, progressCallback)
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
	}, nil
}

// mergeConfig 合并配置（优先级：命令行参数 > ansible.cfg > 默认值）
func (c *PingController) mergeConfig(req *PingRequest) *PingRequest {
	merged := &PingRequest{
		HostsFile:   req.HostsFile,
		HostsDir:    req.HostsDir,
		HostsString: req.HostsString,
		Group:       req.Group,
		User:        req.User,
		KeyPath:     req.KeyPath,
		Password:    req.Password,
		Port:        req.Port,
		Concurrency: req.Concurrency,
	}

	// 加载 ansible.cfg
	ansibleCfg, err := config.LoadAnsibleConfig()
	if err != nil {
		// 如果加载失败，使用默认值
		if merged.Concurrency == 0 {
			merged.Concurrency = 5
		}
		return merged
	}

	// 合并配置（命令行参数优先）
	if merged.User == "" {
		merged.User = ansibleCfg.RemoteUser
	}

	if merged.KeyPath == "" {
		merged.KeyPath = ansibleCfg.PrivateKeyFile
	}

	if merged.Port == "" || merged.Port == "22" {
		// Port 在 ansible.cfg 中没有对应项，保持默认值
		merged.Port = "22"
	}

	if merged.Concurrency == 0 {
		if ansibleCfg.Forks > 0 {
			merged.Concurrency = ansibleCfg.Forks
		} else {
			merged.Concurrency = 5
		}
	}

	return merged
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
	var hosts []executor.Host
	var err error

	// 优先级：命令行参数 > ansible.cfg > 默认值
	// 如果命令行没有指定主机来源，尝试从 ansible.cfg 加载
	if req.HostsDir == "" && req.HostsFile == "" && req.HostsString == "" {
		ansibleCfg, err := config.LoadAnsibleConfig()
		if err == nil && ansibleCfg.Inventory != "" {
			// 从 ansible.cfg 的 inventory 加载
			hosts, err = config.LoadHostsFromInventory(ansibleCfg.Inventory, req.Group)
			if err != nil {
				return nil, fmt.Errorf("从 ansible.cfg inventory 加载主机列表失败: %w", err)
			}
			return hosts, nil
		}
		return nil, fmt.Errorf("必须指定主机列表（-f、-d、-H 或 ansible.cfg 中的 inventory）")
	}

	// 使用命令行参数指定的方式加载
	if req.HostsDir != "" {
		// 从目录加载所有 INI 文件
		hosts, err = config.LoadHostsFromDirectory(req.HostsDir, req.Group)
		if err != nil {
			return nil, fmt.Errorf("从目录加载主机列表失败: %w", err)
		}
	} else if req.HostsFile != "" {
		// 从单个文件加载
		hosts, err = config.LoadHostsFromFileWithGroup(req.HostsFile, req.Group)
		if err != nil {
			return nil, fmt.Errorf("加载主机列表失败: %w", err)
		}
	} else if req.HostsString != "" {
		// 从字符串加载
		hosts, err = config.LoadHostsFromString(req.HostsString)
		if err != nil {
			return nil, fmt.Errorf("解析主机列表失败: %w", err)
		}
	}

	if len(hosts) == 0 {
		return nil, fmt.Errorf("主机列表为空")
	}

	return hosts, nil
}

// executePing 并发执行 ping 测试
func (c *PingController) executePing(hosts []executor.Host, user, keyPath, password, defaultPort string, concurrency int, progressCallback func(string, string, int64, bool)) ([]*ssh.PingResult, error) {
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
					if progressCallback != nil {
						progressCallback(h.Address, fmt.Sprintf("panic: %v", r), 100, true)
					}
				}
				wg.Done()
			}()

			hostAddr := h.Address

			// 限制并发数
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 连接阶段不需要回调，避免输出过多

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

			client, err := ssh.NewClient(h.Address, port, hostUser, hostKeyPath, password)
			if err != nil {
				mu.Lock()
				results[idx] = &ssh.PingResult{
					Host:     h.Address,
					Success:  false,
					Duration: 0,
					Error:    fmt.Errorf("创建客户端失败: %v", err),
				}
				mu.Unlock()
				if progressCallback != nil {
					progressCallback(hostAddr, fmt.Sprintf("连接失败: %v", err), 100, true)
				}
				return
			}

			result, err := client.Ping()
			if err != nil {
				mu.Lock()
				results[idx] = &ssh.PingResult{
					Host:     h.Address,
					Success:  false,
					Duration: 0,
					Error:    err,
				}
				mu.Unlock()
				if progressCallback != nil {
					progressCallback(hostAddr, fmt.Sprintf("测试失败: %v", err), 100, true)
				}
				return
			}

			mu.Lock()
			results[idx] = result
			mu.Unlock()

			if progressCallback != nil {
				if result.Success {
					progressCallback(hostAddr, "成功", 100, false)
				} else {
					progressCallback(hostAddr, "失败", 100, true)
				}
			}
		}(i, host)
	}

	wg.Wait()
	return results, nil
}
