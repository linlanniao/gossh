package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gossh/internal/executor"
)

// AnsibleConfig Ansible 配置文件结构
type AnsibleConfig struct {
	Inventory      string // inventory 文件路径（可以是多个，逗号分隔）
	PrivateKeyFile string // private_key_file
	RemoteUser     string // remote_user
	Forks          int    // forks
	Timeout        int    // timeout
}

// LoadAnsibleConfig 加载 ansible.cfg 配置文件
// 会在当前目录和父目录中查找 ansible.cfg 文件
func LoadAnsibleConfig() (*AnsibleConfig, error) {
	// 查找 ansible.cfg 文件
	cfgPath, err := findAnsibleConfig()
	if err != nil {
		// 如果找不到配置文件，返回默认配置
		return &AnsibleConfig{}, nil
	}

	return parseAnsibleConfig(cfgPath)
}

// findAnsibleConfig 查找 ansible.cfg 文件
// 从当前目录开始，向上查找直到找到或到达根目录
func findAnsibleConfig() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		cfgPath := filepath.Join(dir, "ansible.cfg")
		if _, err := os.Stat(cfgPath); err == nil {
			return cfgPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// 已到达根目录
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("未找到 ansible.cfg 文件")
}

// parseAnsibleConfig 解析 ansible.cfg 文件
func parseAnsibleConfig(cfgPath string) (*AnsibleConfig, error) {
	file, err := os.Open(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("打开配置文件失败: %w", err)
	}
	defer file.Close()

	config := &AnsibleConfig{
		Forks:   5,  // 默认值
		Timeout: 30, // 默认值
	}

	scanner := bufio.NewScanner(file)
	inDefaults := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查是否进入 [defaults] 部分
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.TrimSpace(line[1 : len(line)-1])
			inDefaults = (section == "defaults")
			continue
		}

		// 只在 [defaults] 部分解析配置
		if !inDefaults {
			continue
		}

		// 解析配置项
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "inventory":
			config.Inventory = value
		case "private_key_file":
			config.PrivateKeyFile = value
		case "remote_user":
			config.RemoteUser = value
		case "forks":
			if forks, err := strconv.Atoi(value); err == nil {
				config.Forks = forks
			}
		case "timeout":
			if timeout, err := strconv.Atoi(value); err == nil {
				config.Timeout = timeout
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	return config, nil
}

// LoadHostsFromInventory 从 inventory 配置加载主机列表
// 支持多个文件（逗号分隔），会聚合所有主机
func LoadHostsFromInventory(inventory string, group string) ([]executor.Host, error) {
	if inventory == "" {
		return nil, fmt.Errorf("inventory 配置为空")
	}

	// 分割多个文件路径
	files := strings.Split(inventory, ",")
	var allHosts []executor.Host
	hostMap := make(map[string]bool) // 用于去重

	for _, filePath := range files {
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}

		// 检查是否是目录
		info, err := os.Stat(filePath)
		if err != nil {
			// 文件不存在，跳过
			continue
		}

		if info.IsDir() {
			// 如果是目录，使用目录加载方式
			hosts, err := LoadHostsFromDirectory(filePath, group)
			if err != nil {
				// 如果某个目录加载失败，记录错误但继续处理其他文件
				fmt.Fprintf(os.Stderr, "警告: 从目录 %s 加载主机失败: %v\n", filePath, err)
				continue
			}

			// 聚合主机并去重
			for _, host := range hosts {
				key := fmt.Sprintf("%s:%s", host.Address, host.Port)
				if !hostMap[key] {
					hostMap[key] = true
					allHosts = append(allHosts, host)
				}
			}
		} else {
			// 如果是文件，加载文件中的所有主机
			hostsWithGroups, err := loadHostsFromFileWithGroups(filePath)
			if err != nil {
				// 如果某个文件加载失败，记录错误但继续处理其他文件
				fmt.Fprintf(os.Stderr, "警告: 从文件 %s 加载主机失败: %v\n", filePath, err)
				continue
			}

			// 根据分组筛选
			for _, hwg := range hostsWithGroups {
				if group == "" || group == "all" || hwg.group == group {
					key := fmt.Sprintf("%s:%s", hwg.host.Address, hwg.host.Port)
					if !hostMap[key] {
						hostMap[key] = true
						allHosts = append(allHosts, hwg.host)
					}
				}
			}
		}
	}

	if len(allHosts) == 0 {
		return nil, fmt.Errorf("从 inventory 配置中未找到有效的主机列表")
	}

	return allHosts, nil
}
