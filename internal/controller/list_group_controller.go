package controller

import (
	"fmt"
	"os"
	"strings"

	"gossh/internal/config"
)

// ListGroupController 处理 list-group 命令的业务逻辑
type ListGroupController struct{}

// NewListGroupController 创建新的 ListGroupController
func NewListGroupController() *ListGroupController {
	return &ListGroupController{}
}

// ListGroupRequest list-group 命令的请求参数
type ListGroupRequest struct {
	ConfigFile string // ansible.cfg 配置文件路径
	Inventory  string // 主机列表（文件路径、目录路径或逗号分隔的主机列表）
}

// ListGroupResponse list-group 命令的响应
type ListGroupResponse struct {
	Groups []string
}

// Execute 执行 list-group 命令
func (c *ListGroupController) Execute(req *ListGroupRequest) (*ListGroupResponse, error) {
	// 验证参数
	if err := c.validateRequest(req); err != nil {
		return nil, err
	}

	// 加载组列表
	groups, err := c.loadGroups(req)
	if err != nil {
		return nil, err
	}

	return &ListGroupResponse{
		Groups: groups,
	}, nil
}

// validateRequest 验证请求参数
func (c *ListGroupController) validateRequest(req *ListGroupRequest) error {
	// 允许从命令行参数或 ansible.cfg 加载，所以这里不强制要求
	return nil
}

// loadGroups 加载组列表
func (c *ListGroupController) loadGroups(req *ListGroupRequest) ([]string, error) {
	// 如果未指定主机来源，尝试从 ansible.cfg 加载
	if req.Inventory == "" {
		return c.loadGroupsFromAnsibleConfig(req.ConfigFile)
	}

	// 从命令行参数加载组列表
	return c.loadGroupsFromConfig(req.Inventory)
}

// loadGroupsFromAnsibleConfig 从 ansible.cfg 配置文件加载组列表
func (c *ListGroupController) loadGroupsFromAnsibleConfig(configFile string) ([]string, error) {
	ansibleCfg, err := config.LoadAnsibleConfig(configFile)
	if err == nil && ansibleCfg.Inventory != "" {
		groups, err := config.LoadGroupsFromInventory(ansibleCfg.Inventory)
		if err != nil {
			return nil, fmt.Errorf("从 ansible.cfg inventory 加载组列表失败: %w", err)
		}
		return groups, nil
	}

	return nil, fmt.Errorf("必须指定主机列表（-i 或 ansible.cfg 中的 inventory）")
}

// loadGroupsFromConfig 从配置中加载组列表
// 支持从目录、文件或逗号分隔的字符串加载
func (c *ListGroupController) loadGroupsFromConfig(inventory string) ([]string, error) {
	if inventory == "" {
		return nil, nil
	}

	// 判断是文件、目录还是逗号分隔的主机列表
	info, err := os.Stat(inventory)
	if err == nil {
		// 路径存在，判断是文件还是目录
		if info.IsDir() {
			// 是目录
			groups, err := config.LoadGroupsFromDirectory(inventory)
			if err != nil {
				return nil, fmt.Errorf("从目录加载组列表失败: %w", err)
			}
			return groups, nil
		} else {
			// 是文件
			groups, err := config.LoadGroupsFromFile(inventory)
			if err != nil {
				return nil, fmt.Errorf("加载组列表失败: %w", err)
			}
			return groups, nil
		}
	}

	// 路径不存在，可能是逗号分隔的主机列表
	// 逗号分隔的主机列表没有分组信息，返回空列表
	if strings.Contains(inventory, ",") {
		return []string{}, nil
	}

	// 既不是文件/目录，也不是逗号分隔的列表，返回错误
	return nil, fmt.Errorf("无效的主机列表格式: %s（必须是文件路径、目录路径或逗号分隔的主机列表）", inventory)
}

