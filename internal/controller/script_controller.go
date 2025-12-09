package controller

import (
	"fmt"
	"time"

	"gossh/internal/config"
	"gossh/internal/executor"
	"gossh/internal/ssh"
	"gossh/internal/view"
)

// ScriptController 处理 script 命令的业务逻辑
type ScriptController struct{}

// NewScriptController 创建新的 ScriptController
func NewScriptController() *ScriptController {
	return &ScriptController{}
}

// ScriptCommandRequest script 命令的请求参数
type ScriptCommandRequest struct {
	HostsFile   string
	HostsDir    string // Ansible hosts 目录路径
	HostsString string
	Group       string // Ansible INI 格式的分组名称
	User        string
	KeyPath     string
	Password    string
	Port        string
	ScriptPath  string
	Become      bool
	BecomeUser  string
	Concurrency int
	ShowOutput  bool
}

// ScriptCommandResponse script 命令的响应
type ScriptCommandResponse struct {
	Results       []*ssh.Result
	TotalDuration time.Duration
}

// Execute 执行 script 命令
func (c *ScriptController) Execute(req *ScriptCommandRequest) (*ScriptCommandResponse, error) {
	// 合并配置（优先级：命令行参数 > ansible.cfg > 默认值）
	mergedReq := c.mergeConfig(req)

	// 打印当前配置参数
	view.PrintScriptConfig(
		mergedReq.HostsFile,
		mergedReq.HostsDir,
		mergedReq.HostsString,
		mergedReq.Group,
		mergedReq.User,
		mergedReq.KeyPath,
		mergedReq.Password,
		mergedReq.Port,
		mergedReq.ScriptPath,
		mergedReq.Become,
		mergedReq.BecomeUser,
		mergedReq.Concurrency,
		mergedReq.ShowOutput,
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
	progressTracker := view.NewProgressTracker(len(hosts), "执行脚本")

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

	// 执行脚本
	results, err := exec.ExecuteScriptWithBecome(
		mergedReq.ScriptPath,
		mergedReq.Concurrency,
		mergedReq.Become,
		mergedReq.BecomeUser,
		progressCallback,
	)

	// 记录结束时间并计算总耗时
	totalDuration := time.Since(startTime)

	// 停止进度跟踪器
	progressTracker.Stop()

	if err != nil {
		return nil, fmt.Errorf("执行失败: %w", err)
	}

	return &ScriptCommandResponse{
		Results:       results,
		TotalDuration: totalDuration,
	}, nil
}

// mergeConfig 合并配置（优先级：命令行参数 > ansible.cfg > 默认值）
func (c *ScriptController) mergeConfig(req *ScriptCommandRequest) *ScriptCommandRequest {
	merged := &ScriptCommandRequest{
		HostsFile:   req.HostsFile,
		HostsDir:    req.HostsDir,
		HostsString: req.HostsString,
		Group:       req.Group,
		User:        req.User,
		KeyPath:     req.KeyPath,
		Password:    req.Password,
		Port:        req.Port,
		ScriptPath:  req.ScriptPath,
		Become:      req.Become,
		BecomeUser:  req.BecomeUser,
		Concurrency: req.Concurrency,
		ShowOutput:  req.ShowOutput,
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
		merged.HostsDir = ""
		merged.HostsFile = ""
		merged.HostsString = ""
	}

	if merged.User == "" {
		merged.User = ansibleCfg.RemoteUser
	}

	if merged.KeyPath == "" {
		merged.KeyPath = ansibleCfg.PrivateKeyFile
	}

	if merged.Port == "" || merged.Port == "22" {
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
func (c *ScriptController) validateRequest(req *ScriptCommandRequest) error {
	if req.ScriptPath == "" {
		return fmt.Errorf("必须指定要执行的脚本文件路径（-s）")
	}

	if req.User == "" {
		return fmt.Errorf("必须指定用户名（-u 或 ansible.cfg 中的 remote_user）")
	}

	return nil
}

// loadHosts 加载主机列表
func (c *ScriptController) loadHosts(req *ScriptCommandRequest) ([]executor.Host, error) {
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
