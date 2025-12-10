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
// 如果指定了 configPath，则使用该路径；否则按照以下顺序查找：
// 1. 环境变量 ANSIBLE_CONFIG
// 2. 当前目录和父目录中的 ansible.cfg
// 3. 用户主目录下的 .ansible.cfg
func LoadAnsibleConfig(configPath string) (*AnsibleConfig, error) {
	var cfgPath string
	var err error

	// 如果指定了配置文件路径，直接使用
	if configPath != "" {
		if _, err := os.Stat(configPath); err != nil {
			return nil, fmt.Errorf("配置文件不存在: %s", configPath)
		}
		cfgPath = configPath
	} else {
		// 首先检查环境变量 ANSIBLE_CONFIG
		if envCfg := os.Getenv("ANSIBLE_CONFIG"); envCfg != "" {
			if _, err := os.Stat(envCfg); err == nil {
				cfgPath = envCfg
			}
		}

		// 如果环境变量未设置或文件不存在，查找默认位置
		if cfgPath == "" {
			cfgPath, err = findAnsibleConfig()
			if err != nil {
				// 如果找不到配置文件，返回默认配置
				return &AnsibleConfig{}, nil
			}
		}
	}

	return parseAnsibleConfig(cfgPath)
}

// findAnsibleConfig 查找 ansible.cfg 文件
// 按照以下顺序查找：
// 1. 当前目录和父目录中的 ansible.cfg
// 2. 用户主目录下的 .ansible.cfg
func findAnsibleConfig() (string, error) {
	// 首先从当前目录开始，向上查找直到找到或到达根目录
	dir, err := os.Getwd()
	if err == nil {
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
	}

	// 如果当前目录和父目录中未找到，检查用户主目录
	homeDir, err := os.UserHomeDir()
	if err == nil {
		cfgPath := filepath.Join(homeDir, ".ansible.cfg")
		if _, err := os.Stat(cfgPath); err == nil {
			return cfgPath, nil
		}
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

	if err := parseConfigFile(file, config); err != nil {
		return nil, err
	}

	return config, nil
}

// parseConfigFile 解析配置文件内容
func parseConfigFile(file *os.File, config *AnsibleConfig) error {
	scanner := bufio.NewScanner(file)
	inDefaults := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查是否进入 [defaults] 部分
		if isSectionHeader(line) {
			inDefaults = isDefaultsSection(line)
			continue
		}

		// 只在 [defaults] 部分解析配置
		if !inDefaults {
			continue
		}

		// 解析配置项
		key, value := parseConfigLine(line)
		if key == "" {
			continue
		}

		applyConfigValue(config, key, value)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	return nil
}

// isSectionHeader 检查是否是节标题
func isSectionHeader(line string) bool {
	return strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")
}

// isDefaultsSection 检查是否是 [defaults] 节
func isDefaultsSection(line string) bool {
	section := strings.TrimSpace(line[1 : len(line)-1])
	return section == "defaults"
}

// parseConfigLine 解析配置行，返回 key 和 value
func parseConfigLine(line string) (string, string) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", ""
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	return key, value
}

// applyConfigValue 应用配置值到配置对象
func applyConfigValue(config *AnsibleConfig, key, value string) {
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

		hosts, err := loadHostsFromInventoryPath(filePath, group)
		if err != nil {
			// 如果某个路径加载失败，记录错误但继续处理其他文件
			fmt.Fprintf(os.Stderr, "警告: 从路径 %s 加载主机失败: %v\n", filePath, err)
			continue
		}

		// 聚合主机并去重
		allHosts = mergeHostsWithDedup(allHosts, hosts, hostMap)
	}

	if len(allHosts) == 0 {
		return nil, fmt.Errorf("从 inventory 配置中未找到有效的主机列表")
	}

	return allHosts, nil
}

// loadHostsFromInventoryPath 从单个 inventory 路径加载主机列表
// 支持文件和目录两种类型
func loadHostsFromInventoryPath(filePath, group string) ([]executor.Host, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("路径不存在: %w", err)
	}

	if info.IsDir() {
		return LoadHostsFromDirectory(filePath, group)
	}

	return loadHostsFromInventoryFile(filePath, group)
}

// loadHostsFromInventoryFile 从 inventory 文件加载主机列表
// group 支持逗号分隔的多个组
func loadHostsFromInventoryFile(filePath, group string) ([]executor.Host, error) {
	hostsWithGroups, err := loadHostsFromFileWithGroups(filePath)
	if err != nil {
		return nil, err
	}

	// 解析组列表
	targetGroups := parseGroups(group)

	// 构建主机到分组的映射（一个主机可能属于多个分组）
	hostGroupsMap := make(map[string][]string)
	for _, hwg := range hostsWithGroups {
		// 如果指定了分组，只处理匹配分组的主机
		if len(targetGroups) > 0 && !isGroupMatch(hwg.group, targetGroups) {
			continue
		}

		key := fmt.Sprintf("%s:%s", hwg.host.Address, hwg.host.Port)
		if hwg.group != "" {
			// 检查分组是否已存在，避免重复
			groups := hostGroupsMap[key]
			found := false
			for _, g := range groups {
				if g == hwg.group {
					found = true
					break
				}
			}
			if !found {
				hostGroupsMap[key] = append(groups, hwg.group)
			}
		}
	}

	// 根据分组筛选，并填充分组信息
	var hosts []executor.Host
	hostMap := make(map[string]bool) // 用于去重

	for _, hwg := range hostsWithGroups {
		if isGroupMatch(hwg.group, targetGroups) {
			key := fmt.Sprintf("%s:%s", hwg.host.Address, hwg.host.Port)
			if !hostMap[key] {
				hostMap[key] = true
				host := hwg.host
				host.Groups = hostGroupsMap[key]
				hosts = append(hosts, host)
			}
		}
	}

	return hosts, nil
}

// mergeHostsWithDedup 合并主机列表并去重
func mergeHostsWithDedup(allHosts, newHosts []executor.Host, hostMap map[string]bool) []executor.Host {
	for _, host := range newHosts {
		key := fmt.Sprintf("%s:%s", host.Address, host.Port)
		if !hostMap[key] {
			hostMap[key] = true
			allHosts = append(allHosts, host)
		}
	}
	return allHosts
}

// LoadGroupsFromInventory 从 inventory 配置加载组列表
// 支持多个文件（逗号分隔），会聚合所有组并去重
func LoadGroupsFromInventory(inventory string) ([]string, error) {
	if inventory == "" {
		return nil, fmt.Errorf("inventory 配置为空")
	}

	// 分割多个文件路径
	files := strings.Split(inventory, ",")
	var allGroups []string
	groupSet := make(map[string]bool) // 用于去重

	for _, filePath := range files {
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}

		groups, err := loadGroupsFromInventoryPath(filePath)
		if err != nil {
			// 如果某个路径加载失败，记录错误但继续处理其他文件
			fmt.Fprintf(os.Stderr, "警告: 从路径 %s 加载组列表失败: %v\n", filePath, err)
			continue
		}

		// 聚合组并去重
		for _, group := range groups {
			if !groupSet[group] {
				groupSet[group] = true
				allGroups = append(allGroups, group)
			}
		}
	}

	return allGroups, nil
}

// loadGroupsFromInventoryPath 从单个 inventory 路径加载组列表
// 支持文件和目录两种类型
func loadGroupsFromInventoryPath(filePath string) ([]string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("路径不存在: %w", err)
	}

	if info.IsDir() {
		return LoadGroupsFromDirectory(filePath)
	}

	return LoadGroupsFromFile(filePath)
}
