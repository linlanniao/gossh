package controller

import (
	"fmt"
	"time"

	"gossh/internal/config"
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
	HostsFile   string
	HostsDir    string // Ansible hosts 目录路径
	HostsString string
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
}

// RunCommandResponse run 命令的响应
type RunCommandResponse struct {
	Results       []*ssh.Result
	TotalDuration time.Duration // 总执行时间（从开始到所有任务完成）
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
		mergedReq.HostsFile,
		mergedReq.HostsDir,
		mergedReq.HostsString,
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
		"hosts_file":   mergedReq.HostsFile,
		"hosts_dir":    mergedReq.HostsDir,
		"hosts_string": mergedReq.HostsString,
		"group":        mergedReq.Group,
		"user":         mergedReq.User,
		"key_path":     mergedReq.KeyPath,
		"port":         mergedReq.Port,
		"command":      mergedReq.Command,
		"become":       mergedReq.Become,
		"become_user":  mergedReq.BecomeUser,
		"concurrency":  mergedReq.Concurrency,
		"show_output":  mergedReq.ShowOutput,
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
		progressCallback,
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
	}, nil
}

// mergeConfig 合并配置（优先级：命令行参数 > ansible.cfg > 默认值）
func (c *RunController) mergeConfig(req *RunCommandRequest) *RunCommandRequest {
	merged := &RunCommandRequest{
		HostsFile:   req.HostsFile,
		HostsDir:    req.HostsDir,
		HostsString: req.HostsString,
		Group:       req.Group,
		User:        req.User,
		KeyPath:     req.KeyPath,
		Password:    req.Password,
		Port:        req.Port,
		Command:     req.Command,
		Become:      req.Become,
		BecomeUser:  req.BecomeUser,
		Concurrency: req.Concurrency,
		ShowOutput:  req.ShowOutput,
		LogDir:      req.LogDir,
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
	if merged.HostsFile == "" && merged.HostsDir == "" && merged.HostsString == "" {
		// 如果命令行没有指定主机来源，使用 ansible.cfg 的 inventory
		merged.HostsDir = ""    // 标记使用 inventory
		merged.HostsFile = ""   // 标记使用 inventory
		merged.HostsString = "" // 标记使用 inventory
	}

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
