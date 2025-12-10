package controller

import (
	"fmt"
	"time"

	"gossh/internal/executor"
	"gossh/internal/logger"
	"gossh/internal/ssh"
	"gossh/internal/view"
)

// RunController 处理 run 命令的业务逻辑
type RunController struct{}

// NewRunController 创建新的 RunController
func NewRunController() *RunController {
	return &RunController{}
}

// RunCommandRequest run 命令的请求参数
type RunCommandRequest struct {
	ConfigFile  string // ansible.cfg 配置文件路径
	Inventory   string // 主机列表（文件路径、目录路径或逗号分隔的主机列表）
	Group       string // Ansible INI 格式的分组名称
	User        string
	KeyPath     string
	Password    string
	Port        string
	Command     string
	Become      bool
	BecomeUser  string
	Concurrency int
	ShowOutput  bool
	LogDir      string
	Limit       int
	Offset      int
}

// RunCommandResponse run 命令的响应
type RunCommandResponse struct {
	Results       []*ssh.Result
	TotalDuration time.Duration  // 总执行时间（从开始到所有任务完成）
	Group         string          // 分组名称（用户指定的）
	Hosts         []executor.Host // 主机列表（包含分组信息）
}

// Execute 执行 run 命令
func (c *RunController) Execute(req *RunCommandRequest) (*RunCommandResponse, error) {
	// 合并配置（优先级：命令行参数 > ansible.cfg > 默认值）
	mergedReq := c.mergeConfig(req)

	// 创建日志记录器
	log, err := logger.NewLogger(mergedReq.LogDir, "run")
	if err != nil {
		return nil, fmt.Errorf("创建日志记录器失败: %w", err)
	}
	defer log.Close()

	// 打印当前配置参数
	view.PrintRunConfig(
		mergedReq.Inventory,
		mergedReq.Group,
		mergedReq.User,
		mergedReq.KeyPath,
		mergedReq.Password,
		mergedReq.Port,
		mergedReq.Command,
		mergedReq.Become,
		mergedReq.BecomeUser,
		mergedReq.Concurrency,
		mergedReq.ShowOutput,
	)

	// 记录命令开始
	log.LogCommandStart("run", map[string]interface{}{
		"inventory":   mergedReq.Inventory,
		"group":       mergedReq.Group,
		"user":        mergedReq.User,
		"key_path":    mergedReq.KeyPath,
		"port":        mergedReq.Port,
		"command":     mergedReq.Command,
		"become":      mergedReq.Become,
		"become_user": mergedReq.BecomeUser,
		"concurrency": mergedReq.Concurrency,
		"show_output": mergedReq.ShowOutput,
	})

	// 验证参数
	if err := c.validateRequest(mergedReq); err != nil {
		log.LogError("参数验证失败", err)
		return nil, err
	}

	// 加载主机列表
	hosts, err := c.loadHosts(mergedReq)
	if err != nil {
		log.LogError("加载主机列表失败", err)
		return nil, err
	}

	// 应用 offset 和 limit
	hosts = c.applyLimitAndOffset(hosts, mergedReq.Offset, mergedReq.Limit)

	// 记录主机列表
	hostAddresses := make([]string, len(hosts))
	for i, h := range hosts {
		hostAddresses[i] = h.Address
	}
	log.LogHosts(hostAddresses)

	// 设置默认端口
	port := mergedReq.Port
	if port == "" {
		port = "22"
	}

	// 创建进度跟踪器
	progressTracker := view.NewProgressTracker(len(hosts), "执行命令")

	// 创建执行器
	exec := executor.NewExecutor(hosts, mergedReq.User, mergedReq.KeyPath, mergedReq.Password, port)

	// 记录开始时间
	startTime := time.Now()

	// 执行命令
	var results []*ssh.Result
	results, err = exec.ExecuteCommandWithBecome(
		mergedReq.Command,
		mergedReq.Concurrency,
		mergedReq.Become,
		mergedReq.BecomeUser,
		progressTracker,
	)

	// 记录结束时间并计算总耗时
	totalDuration := time.Since(startTime)

	// 停止进度跟踪器
	progressTracker.Stop()

	// 记录每个主机的执行结果
	successCount := 0
	for _, result := range results {
		success := result.ExitCode == 0 && result.Error == nil
		if success {
			successCount++
		}
		log.LogHostResult(
			result.Host,
			result.Command,
			result.ExitCode,
			result.Duration,
			success,
			result.Stdout,
			result.Stderr,
			result.Error,
		)
	}

	// 记录命令结束
	commandSuccess := err == nil && successCount == len(results)
	if err != nil {
		log.LogCommandEnd("run", totalDuration, false, err)
		return nil, fmt.Errorf("执行失败: %w", err)
	}

	log.LogCommandEnd("run", totalDuration, commandSuccess, nil)

	return &RunCommandResponse{
		Results:       results,
		TotalDuration: totalDuration,
		Group:         mergedReq.Group,
	}, nil
}

// mergeConfig 合并配置（优先级：命令行参数 > ansible.cfg > 默认值）
func (c *RunController) mergeConfig(req *RunCommandRequest) *RunCommandRequest {
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

	return &RunCommandRequest{
		ConfigFile:  req.ConfigFile,
		Inventory:   commonCfg.Inventory,
		Group:       commonCfg.Group,
		User:        commonCfg.User,
		KeyPath:     commonCfg.KeyPath,
		Password:    commonCfg.Password,
		Port:        commonCfg.Port,
		Command:     req.Command,
		Become:      req.Become,
		BecomeUser:  req.BecomeUser,
		Concurrency: commonCfg.Concurrency,
		ShowOutput:  req.ShowOutput,
		LogDir:      req.LogDir,
		Limit:       req.Limit,
		Offset:      req.Offset,
	}
}

// validateRequest 验证请求参数
func (c *RunController) validateRequest(req *RunCommandRequest) error {
	if req.Command == "" {
		return fmt.Errorf("必须指定要执行的命令（-c）")
	}

	if req.User == "" {
		return fmt.Errorf("必须指定用户名（-u 或 ansible.cfg 中的 remote_user）")
	}

	return nil
}

// loadHosts 加载主机列表
func (c *RunController) loadHosts(req *RunCommandRequest) ([]executor.Host, error) {
	return LoadHosts(&CommonConfig{
		ConfigFile: req.ConfigFile,
		Inventory:  req.Inventory,
		Group:      req.Group,
	}, true)
}

// applyLimitAndOffset 应用 limit 和 offset 来过滤主机列表
func (c *RunController) applyLimitAndOffset(hosts []executor.Host, offset, limit int) []executor.Host {
	total := len(hosts)
	if total == 0 {
		return hosts
	}

	// 应用 offset
	if offset > 0 {
		if offset >= total {
			return []executor.Host{}
		}
		hosts = hosts[offset:]
	}

	// 应用 limit
	if limit > 0 && limit < len(hosts) {
		hosts = hosts[:limit]
	}

	return hosts
}
