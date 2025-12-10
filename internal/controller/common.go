package controller

import (
	"fmt"
	"os"
	"strings"

	"gossh/internal/config"
	"gossh/internal/executor"
)

// CommonConfig 公共配置结构
type CommonConfig struct {
	ConfigFile  string // ansible.cfg 配置文件路径
	Inventory   string // 主机列表（文件路径、目录路径或逗号分隔的主机列表）
	Group       string
	User        string
	KeyPath     string
	Password    string
	Port        string
	Concurrency int
}

// MergeCommonConfig 合并公共配置（优先级：命令行参数 > ansible.cfg > 默认值）
func MergeCommonConfig(cfg *CommonConfig) *CommonConfig {
	merged := &CommonConfig{
		ConfigFile:  cfg.ConfigFile,
		Inventory:   cfg.Inventory,
		Group:       cfg.Group,
		User:        cfg.User,
		KeyPath:     cfg.KeyPath,
		Password:    cfg.Password,
		Port:        cfg.Port,
		Concurrency: cfg.Concurrency,
	}

	ansibleCfg, err := config.LoadAnsibleConfig(cfg.ConfigFile)
	if err != nil {
		if merged.Concurrency == 0 {
			merged.Concurrency = 5
		}
		return merged
	}

	if merged.User == "" {
		merged.User = ansibleCfg.RemoteUser
	}

	if merged.KeyPath == "" {
		merged.KeyPath = ansibleCfg.PrivateKeyFile
	}

	if merged.Port == "" {
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

// LoadHosts 加载主机列表（公共方法）
// 优先级：命令行参数 > ansible.cfg inventory > 错误
func LoadHosts(cfg *CommonConfig, requireHosts bool) ([]executor.Host, error) {
	// 如果未指定主机来源，尝试从 ansible.cfg 加载
	if cfg.Inventory == "" {
		return loadHostsFromAnsibleConfig(cfg.ConfigFile, cfg.Group, requireHosts)
	}

	// 从命令行参数加载主机列表
	hosts, err := loadHostsFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	if len(hosts) == 0 {
		return nil, fmt.Errorf("主机列表为空")
	}

	return hosts, nil
}

// loadHostsFromAnsibleConfig 从 ansible.cfg 配置文件加载主机列表
func loadHostsFromAnsibleConfig(configFile, group string, requireHosts bool) ([]executor.Host, error) {
	ansibleCfg, err := config.LoadAnsibleConfig(configFile)
	if err == nil && ansibleCfg.Inventory != "" {
		hosts, err := config.LoadHostsFromInventory(ansibleCfg.Inventory, group)
		if err != nil {
			return nil, fmt.Errorf("从 ansible.cfg inventory 加载主机列表失败: %w", err)
		}
		return hosts, nil
	}

	if requireHosts {
		return nil, fmt.Errorf("必须指定主机列表（-i 或 ansible.cfg 中的 inventory）")
	}

	return nil, nil
}

// loadHostsFromConfig 从配置中加载主机列表
// 支持从目录、文件或逗号分隔的字符串加载
func loadHostsFromConfig(cfg *CommonConfig) ([]executor.Host, error) {
	if cfg.Inventory == "" {
		return nil, nil
	}

	// 判断是文件、目录还是逗号分隔的主机列表
	info, err := os.Stat(cfg.Inventory)
	if err == nil {
		// 路径存在，判断是文件还是目录
		if info.IsDir() {
			// 是目录
			hosts, err := config.LoadHostsFromDirectory(cfg.Inventory, cfg.Group)
			if err != nil {
				return nil, fmt.Errorf("从目录加载主机列表失败: %w", err)
			}
			return hosts, nil
		} else {
			// 是文件
			hosts, err := config.LoadHostsFromFileWithGroup(cfg.Inventory, cfg.Group)
			if err != nil {
				return nil, fmt.Errorf("加载主机列表失败: %w", err)
			}
			return hosts, nil
		}
	}

	// 路径不存在，可能是逗号分隔的主机列表
	if strings.Contains(cfg.Inventory, ",") {
		hosts, err := config.LoadHostsFromString(cfg.Inventory)
		if err != nil {
			return nil, fmt.Errorf("解析主机列表失败: %w", err)
		}
		return hosts, nil
	}

	// 既不是文件/目录，也不是逗号分隔的列表，返回错误
	return nil, fmt.Errorf("无效的主机列表格式: %s（必须是文件路径、目录路径或逗号分隔的主机列表）", cfg.Inventory)
}
