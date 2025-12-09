package controller

import (
	"gossh/internal/executor"
	"gossh/internal/view"
)

// ListController 处理 list 命令的业务逻辑
type ListController struct{}

// NewListController 创建新的 ListController
func NewListController() *ListController {
	return &ListController{}
}

// ListRequest list 命令的请求参数
type ListRequest struct {
	HostsFile   string
	HostsDir    string // Ansible hosts 目录路径
	HostsString string
	Group       string // Ansible INI 格式的分组名称
	Format      string // 输出格式: ip, full, json
}

// ListResponse list 命令的响应
type ListResponse struct {
	Hosts []executor.Host
}

// Execute 执行 list 命令
func (c *ListController) Execute(req *ListRequest) (*ListResponse, error) {
	// 打印当前配置参数
	view.PrintListConfig(
		req.HostsFile,
		req.HostsDir,
		req.HostsString,
		req.Group,
		req.Format,
	)

	// 验证参数
	if err := c.validateRequest(req); err != nil {
		return nil, err
	}

	// 加载主机列表
	hosts, err := c.loadHosts(req)
	if err != nil {
		return nil, err
	}

	return &ListResponse{
		Hosts: hosts,
	}, nil
}

// validateRequest 验证请求参数
func (c *ListController) validateRequest(req *ListRequest) error {
	// 允许从命令行参数或 ansible.cfg 加载，所以这里不强制要求
	return nil
}

// loadHosts 加载主机列表
func (c *ListController) loadHosts(req *ListRequest) ([]executor.Host, error) {
	return LoadHosts(&CommonConfig{
		HostsFile:   req.HostsFile,
		HostsDir:    req.HostsDir,
		HostsString: req.HostsString,
		Group:       req.Group,
	}, true)
}
