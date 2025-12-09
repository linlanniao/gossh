package controller

import (
	"fmt"

	"gossh/internal/config"
	"gossh/internal/executor"
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
	ScriptPath  string
	Concurrency int
	ShowOutput  bool
}

// RunCommandResponse run 命令的响应
type RunCommandResponse struct {
	Results []*ssh.Result
}

// Execute 执行 run 命令
func (c *RunController) Execute(req *RunCommandRequest) (*RunCommandResponse, error) {
	// 合并配置（优先级：命令行参数 > ansible.cfg > 默认值）
	mergedReq := c.mergeConfig(req)

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
		mergedReq.ScriptPath,
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
	title := "执行命令"
	if mergedReq.ScriptPath != "" {
		title = "执行脚本"
	}
	progressTracker := view.NewProgressTracker(len(hosts), title)

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

	// 执行命令或脚本
	var results []*ssh.Result
	if mergedReq.Command != "" {
		results, err = exec.ExecuteCommand(mergedReq.Command, mergedReq.Concurrency, progressCallback)
	} else {
		results, err = exec.ExecuteScript(mergedReq.ScriptPath, mergedReq.Concurrency, progressCallback)
	}

	// 停止进度跟踪器
	progressTracker.Stop()

	if err != nil {
		return nil, fmt.Errorf("执行失败: %w", err)
	}

	return &RunCommandResponse{
		Results: results,
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
		ScriptPath:  req.ScriptPath,
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
	// 主机列表可以从命令行参数或 ansible.cfg 加载，所以这里不强制要求

	if req.Command == "" && req.ScriptPath == "" {
		return fmt.Errorf("必须指定要执行的命令（-c）或脚本（-s）")
	}

	if req.Command != "" && req.ScriptPath != "" {
		return fmt.Errorf("不能同时指定命令和脚本")
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
