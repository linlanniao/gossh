package controller

import (
	"fmt"

	"gossh/internal/config"
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
	} else {
		return nil, fmt.Errorf("必须指定主机列表（-f、-d、-H 或 ansible.cfg 中的 inventory）")
	}

	if len(hosts) == 0 {
		return nil, fmt.Errorf("主机列表为空")
	}

	return hosts, nil
}
