package controller

import (
	"fmt"

	"gossh/internal/config"
	"gossh/internal/executor"
)

// CommonConfig 公共配置结构
type CommonConfig struct {
	HostsFile   string
	HostsDir    string
	HostsString string
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
		HostsFile:   cfg.HostsFile,
		HostsDir:    cfg.HostsDir,
		HostsString: cfg.HostsString,
		Group:       cfg.Group,
		User:        cfg.User,
		KeyPath:     cfg.KeyPath,
		Password:    cfg.Password,
		Port:        cfg.Port,
		Concurrency: cfg.Concurrency,
	}

	ansibleCfg, err := config.LoadAnsibleConfig()
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
	if cfg.HostsDir == "" && cfg.HostsFile == "" && cfg.HostsString == "" {
		return loadHostsFromAnsibleConfig(cfg.Group, requireHosts)
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
func loadHostsFromAnsibleConfig(group string, requireHosts bool) ([]executor.Host, error) {
	ansibleCfg, err := config.LoadAnsibleConfig()
	if err == nil && ansibleCfg.Inventory != "" {
		hosts, err := config.LoadHostsFromInventory(ansibleCfg.Inventory, group)
		if err != nil {
			return nil, fmt.Errorf("从 ansible.cfg inventory 加载主机列表失败: %w", err)
		}
		return hosts, nil
	}

	if requireHosts {
		return nil, fmt.Errorf("必须指定主机列表（-f、-d、-H 或 ansible.cfg 中的 inventory）")
	}

	return nil, nil
}

// loadHostsFromConfig 从配置中加载主机列表
// 支持从目录、文件或字符串加载
func loadHostsFromConfig(cfg *CommonConfig) ([]executor.Host, error) {
	if cfg.HostsDir != "" {
		hosts, err := config.LoadHostsFromDirectory(cfg.HostsDir, cfg.Group)
		if err != nil {
			return nil, fmt.Errorf("从目录加载主机列表失败: %w", err)
		}
		return hosts, nil
	}

	if cfg.HostsFile != "" {
		hosts, err := config.LoadHostsFromFileWithGroup(cfg.HostsFile, cfg.Group)
		if err != nil {
			return nil, fmt.Errorf("加载主机列表失败: %w", err)
		}
		return hosts, nil
	}

	if cfg.HostsString != "" {
		hosts, err := config.LoadHostsFromString(cfg.HostsString)
		if err != nil {
			return nil, fmt.Errorf("解析主机列表失败: %w", err)
		}
		return hosts, nil
	}

	return nil, nil
}
