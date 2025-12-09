package controller

import (
	"fmt"
	"time"

	"gossh/internal/executor"
	"gossh/internal/logger"
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
	LogDir      string
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

	// 创建日志记录器
	log, err := logger.NewLogger(mergedReq.LogDir, "upload")
	if err != nil {
		return nil, fmt.Errorf("创建日志记录器失败: %w", err)
	}
	defer log.Close()

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

	// 记录命令开始
	log.LogCommandStart("upload", map[string]interface{}{
		"hosts_file":   mergedReq.HostsFile,
		"hosts_dir":    mergedReq.HostsDir,
		"hosts_string": mergedReq.HostsString,
		"group":        mergedReq.Group,
		"user":         mergedReq.User,
		"key_path":     mergedReq.KeyPath,
		"port":         mergedReq.Port,
		"local_path":   mergedReq.LocalPath,
		"remote_path":  mergedReq.RemotePath,
		"mode":         mergedReq.Mode,
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

	// 设置默认文件权限
	mode := mergedReq.Mode
	if mode == "" {
		mode = "0644"
	}

	// 创建进度跟踪器
	progressTracker := view.NewProgressTracker(len(hosts), "上传文件")

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
		log.LogCommandEnd("upload", totalDuration, false, err)
		return nil, fmt.Errorf("上传失败: %w", err)
	}

	log.LogCommandEnd("upload", totalDuration, commandSuccess, nil)

	return &UploadCommandResponse{
		Results:       results,
		TotalDuration: totalDuration,
	}, nil
}

// mergeConfig 合并配置（优先级：命令行参数 > ansible.cfg > 默认值）
func (c *UploadController) mergeConfig(req *UploadCommandRequest) *UploadCommandRequest {
	commonCfg := MergeCommonConfig(&CommonConfig{
		HostsFile:   req.HostsFile,
		HostsDir:    req.HostsDir,
		HostsString: req.HostsString,
		Group:       req.Group,
		User:        req.User,
		KeyPath:     req.KeyPath,
		Password:    req.Password,
		Port:        req.Port,
		Concurrency: req.Concurrency,
	})

	return &UploadCommandRequest{
		HostsFile:   commonCfg.HostsFile,
		HostsDir:    commonCfg.HostsDir,
		HostsString: commonCfg.HostsString,
		Group:       commonCfg.Group,
		User:        commonCfg.User,
		KeyPath:     commonCfg.KeyPath,
		Password:    commonCfg.Password,
		Port:        commonCfg.Port,
		LocalPath:   req.LocalPath,
		RemotePath:  req.RemotePath,
		Mode:        req.Mode,
		Concurrency: commonCfg.Concurrency,
		ShowOutput:  req.ShowOutput,
		LogDir:      req.LogDir,
	}
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
	return LoadHosts(&CommonConfig{
		HostsFile:   req.HostsFile,
		HostsDir:    req.HostsDir,
		HostsString: req.HostsString,
		Group:       req.Group,
	}, true)
}
