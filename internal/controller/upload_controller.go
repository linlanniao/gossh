package controller

import (
	"fmt"
	"time"

	"gossh/internal/config"
	"gossh/internal/executor"
	"gossh/internal/ssh"
	"gossh/internal/view"
)

// UploadController 处理 upload 命令的业务逻辑
type UploadController struct{}

// NewUploadController 创建新的 UploadController
func NewUploadController() *UploadController {
	return &UploadController{}
}

// UploadCommandRequest upload 命令的请求参数
type UploadCommandRequest struct {
	HostsFile   string
	HostsDir    string // Ansible hosts 目录路径
	HostsString string
	Group       string // Ansible INI 格式的分组名称
	User        string
	KeyPath     string
	Password    string
	Port        string
	LocalPath   string
	RemotePath  string
	Mode        string
	Concurrency int
	ShowOutput  bool
}

// UploadCommandResponse upload 命令的响应
type UploadCommandResponse struct {
	Results       []*ssh.Result
	TotalDuration time.Duration
}

// Execute 执行 upload 命令
func (c *UploadController) Execute(req *UploadCommandRequest) (*UploadCommandResponse, error) {
	// 合并配置（优先级：命令行参数 > ansible.cfg > 默认值）
	mergedReq := c.mergeConfig(req)

	// 打印当前配置参数
	view.PrintUploadConfig(
		mergedReq.HostsFile,
		mergedReq.HostsDir,
		mergedReq.HostsString,
		mergedReq.Group,
		mergedReq.User,
		mergedReq.KeyPath,
		mergedReq.Password,
		mergedReq.Port,
		mergedReq.LocalPath,
		mergedReq.RemotePath,
		mergedReq.Mode,
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

	// 设置默认文件权限
	mode := mergedReq.Mode
	if mode == "" {
		mode = "0644"
	}

	// 创建进度跟踪器
	progressTracker := view.NewProgressTracker(len(hosts), "上传文件")

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

	// 上传文件
	results, err := exec.UploadFile(
		mergedReq.LocalPath,
		mergedReq.RemotePath,
		mode,
		mergedReq.Concurrency,
		progressCallback,
	)

	// 记录结束时间并计算总耗时
	totalDuration := time.Since(startTime)

	// 停止进度跟踪器
	progressTracker.Stop()

	if err != nil {
		return nil, fmt.Errorf("上传失败: %w", err)
	}

	return &UploadCommandResponse{
		Results:       results,
		TotalDuration: totalDuration,
	}, nil
}

// mergeConfig 合并配置（优先级：命令行参数 > ansible.cfg > 默认值）
func (c *UploadController) mergeConfig(req *UploadCommandRequest) *UploadCommandRequest {
	merged := &UploadCommandRequest{
		HostsFile:   req.HostsFile,
		HostsDir:    req.HostsDir,
		HostsString: req.HostsString,
		Group:       req.Group,
		User:        req.User,
		KeyPath:     req.KeyPath,
		Password:    req.Password,
		Port:        req.Port,
		LocalPath:   req.LocalPath,
		RemotePath:  req.RemotePath,
		Mode:        req.Mode,
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
func (c *UploadController) validateRequest(req *UploadCommandRequest) error {
	if req.LocalPath == "" {
		return fmt.Errorf("必须指定本地文件路径（-l）")
	}

	if req.RemotePath == "" {
		return fmt.Errorf("必须指定远程文件路径（-r）")
	}

	if req.User == "" {
		return fmt.Errorf("必须指定用户名（-u 或 ansible.cfg 中的 remote_user）")
	}

	return nil
}

// loadHosts 加载主机列表
func (c *UploadController) loadHosts(req *UploadCommandRequest) ([]executor.Host, error) {
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
