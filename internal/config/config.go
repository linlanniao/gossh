package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gossh/internal/executor"
)

// LoadHostsFromFile 从文件加载主机列表
// 支持格式：
//  1. 普通格式：
//     host1:port
//     host2
//     user@host3:port
//     user@host4
//  2. Ansible INI 格式：
//     [group_name]
//     host1
//     host2
//     如果指定了 group，只加载该分组的主机；如果未指定，加载所有分组的主机
func LoadHostsFromFile(filePath string) ([]executor.Host, error) {
	return LoadHostsFromFileWithGroup(filePath, "")
}

// parseGroups 解析组字符串，支持逗号分隔和 "all"
// 返回组列表，如果包含 "all" 或为空，返回空切片表示加载所有组
func parseGroups(groupStr string) []string {
	if groupStr == "" || groupStr == "all" {
		return []string{} // 空切片表示所有组
	}

	// 分割逗号分隔的组名
	parts := strings.Split(groupStr, ",")
	groups := make([]string, 0, len(parts))
	groupSet := make(map[string]bool)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// 如果包含 "all"，返回空切片表示所有组
		if part == "all" {
			return []string{}
		}
		// 去重
		if !groupSet[part] {
			groupSet[part] = true
			groups = append(groups, part)
		}
	}

	return groups
}

// isGroupMatch 检查主机组是否匹配指定的组列表
// 如果 groups 为空，表示匹配所有组
func isGroupMatch(hostGroup string, targetGroups []string) bool {
	if len(targetGroups) == 0 {
		return true // 空列表表示所有组
	}
	for _, tg := range targetGroups {
		if hostGroup == tg {
			return true
		}
	}
	return false
}

// LoadHostsFromFileWithGroup 从文件加载主机列表，支持指定分组
// group 为空字符串或 "all" 时加载所有分组的主机，支持逗号分隔的多个组
func LoadHostsFromFileWithGroup(filePath, group string) ([]executor.Host, error) {
	// 解析组列表
	targetGroups := parseGroups(group)

	// 加载所有主机和分组的映射关系
	hostsWithGroups, err := loadHostsFromFileWithGroups(filePath)
	if err != nil {
		return nil, err
	}

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

	// 构建主机列表，并填充分组信息
	var hosts []executor.Host
	hostMap := make(map[string]bool) // 用于去重

	for _, hwg := range hostsWithGroups {
		// 如果指定了分组，只处理匹配分组的主机
		if len(targetGroups) > 0 && !isGroupMatch(hwg.group, targetGroups) {
			continue
		}

		key := fmt.Sprintf("%s:%s", hwg.host.Address, hwg.host.Port)
		if !hostMap[key] {
			hostMap[key] = true
			host := hwg.host
			host.Groups = hostGroupsMap[key]
			hosts = append(hosts, host)
		}
	}

	// 如果指定了分组但没有找到匹配的主机
	if len(targetGroups) > 0 && len(hosts) == 0 {
		groupsStr := strings.Join(targetGroups, ",")
		return nil, fmt.Errorf("未找到分组 '%s' 或这些分组中没有主机", groupsStr)
	}

	return hosts, nil
}

// detectINIFormat 检测文件是否是 INI 格式
func detectINIFormat(file *os.File) (bool, error) {
	scanner := bufio.NewScanner(file)
	sectionPattern := regexp.MustCompile(`^\s*\[.+\]\s*$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 如果找到 [section] 格式，认为是 INI 格式
		if sectionPattern.MatchString(line) {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

// loadHostsFromINI 从 INI 格式文件加载主机列表
// targetGroups 为空切片时加载所有分组的主机
func loadHostsFromINI(file *os.File, targetGroups []string) ([]executor.Host, error) {
	var hosts []executor.Host
	scanner := bufio.NewScanner(file)
	sectionPattern := regexp.MustCompile(`^\s*\[(.+)\]\s*$`)
	currentGroup := ""
	loadAllGroups := len(targetGroups) == 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查是否是分组标记 [group]
		if matches := sectionPattern.FindStringSubmatch(line); matches != nil {
			currentGroup = strings.TrimSpace(matches[1])
			continue
		}

		// 如果指定了分组，只加载匹配分组的主机
		if !loadAllGroups && !isGroupMatch(currentGroup, targetGroups) {
			continue
		}

		// 解析主机行
		host := parseHostLine(line)
		hosts = append(hosts, host)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 如果指定了分组但没有找到匹配的主机
	if !loadAllGroups && len(hosts) == 0 {
		groupsStr := strings.Join(targetGroups, ",")
		return nil, fmt.Errorf("未找到分组 '%s' 或这些分组中没有主机", groupsStr)
	}

	return hosts, nil
}

// loadHostsFromPlain 从普通格式文件加载主机列表
func loadHostsFromPlain(file *os.File) ([]executor.Host, error) {
	var hosts []executor.Host
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // 跳过空行和注释
		}

		host := parseHostLine(line)
		hosts = append(hosts, host)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	return hosts, nil
}

// LoadHostsFromString 从字符串加载主机列表（逗号分隔）
func LoadHostsFromString(hostsStr string) ([]executor.Host, error) {
	if hostsStr == "" {
		return nil, fmt.Errorf("主机列表为空")
	}

	var hosts []executor.Host
	parts := strings.Split(hostsStr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		host := parseHostLine(part)
		hosts = append(hosts, host)
	}

	return hosts, nil
}

// parseHostLine 解析主机行
// 支持格式：
// - host:port
// - host
// - user@host:port
// - user@host
func parseHostLine(line string) executor.Host {
	host := executor.Host{
		Port: "22", // 默认 SSH 端口
	}

	// 检查是否有用户信息
	if idx := strings.Index(line, "@"); idx != -1 {
		host.User = line[:idx]
		line = line[idx+1:]
	}

	// 检查是否有端口
	if idx := strings.LastIndex(line, ":"); idx != -1 {
		host.Address = line[:idx]
		host.Port = line[idx+1:]
	} else {
		host.Address = line
	}

	return host
}

// hostWithGroup 用于存储主机和分组的映射关系
type hostWithGroup struct {
	host  executor.Host
	group string
}

// LoadHostsFromDirectory 从目录加载所有文件的主机列表并聚合
// 会递归读取目录下所有子文件（支持普通格式和 INI 格式），先聚合所有主机，然后根据分组筛选
// group 为空字符串或 "all" 时加载所有分组的主机，支持逗号分隔的多个组
func LoadHostsFromDirectory(dirPath, group string) ([]executor.Host, error) {
	// 检查目录是否存在
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("目录不存在或无法访问: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("路径不是目录: %s", dirPath)
	}

	// 解析组列表
	targetGroups := parseGroups(group)

	// 存储所有主机和分组的映射关系
	var allHostsWithGroup []hostWithGroup
	hostMap := make(map[string]bool) // 用于去重，key 格式: "address:port"

	// 支持的文件扩展名列表（空字符串表示无扩展名的文件也支持）
	supportedExts := map[string]bool{
		".ini":  true,
		".txt":  true,
		".conf": true,
		".hosts": true,
		"":      true, // 无扩展名的文件也支持
	}

	// 遍历目录中的所有文件（递归），先聚合所有主机
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 跳过隐藏文件（以 . 开头的文件）
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// 检查文件扩展名
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if !supportedExts[ext] {
			return nil
		}

		// 加载文件中的所有主机和分组信息（不指定分组，加载所有）
		hostsWithGroups, err := loadHostsFromFileWithGroups(path)
		if err != nil {
			// 如果某个文件读取失败，记录错误但继续处理其他文件
			fmt.Fprintf(os.Stderr, "警告: 读取文件 %s 失败: %v\n", path, err)
			return nil
		}

		// 聚合主机并去重
		for _, hwg := range hostsWithGroups {
			key := fmt.Sprintf("%s:%s", hwg.host.Address, hwg.host.Port)
			if !hostMap[key] {
				hostMap[key] = true
				allHostsWithGroup = append(allHostsWithGroup, hwg)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历目录失败: %w", err)
	}

	// 构建主机到分组的映射（一个主机可能属于多个分组）
	hostGroupsMap := make(map[string][]string)
	for _, hwg := range allHostsWithGroup {
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

	// 根据分组筛选主机，并填充分组信息
	var resultHosts []executor.Host
	resultHostMap := make(map[string]bool) // 用于去重

	if len(targetGroups) == 0 {
		// 返回所有主机
		for _, hwg := range allHostsWithGroup {
			key := fmt.Sprintf("%s:%s", hwg.host.Address, hwg.host.Port)
			if !resultHostMap[key] {
				resultHostMap[key] = true
				host := hwg.host
				host.Groups = hostGroupsMap[key]
				resultHosts = append(resultHosts, host)
			}
		}
	} else {
		// 只返回匹配分组的主机
		for _, hwg := range allHostsWithGroup {
			if isGroupMatch(hwg.group, targetGroups) {
				key := fmt.Sprintf("%s:%s", hwg.host.Address, hwg.host.Port)
				if !resultHostMap[key] {
					resultHostMap[key] = true
					host := hwg.host
					host.Groups = hostGroupsMap[key]
					resultHosts = append(resultHosts, host)
				}
			}
		}
	}

	if len(resultHosts) == 0 {
		if len(targetGroups) == 0 {
			return nil, fmt.Errorf("目录中没有找到有效的主机列表")
		}
		groupsStr := strings.Join(targetGroups, ",")
		return nil, fmt.Errorf("未找到分组 '%s' 或这些分组中没有主机", groupsStr)
	}

	return resultHosts, nil
}

// loadHostsFromFileWithGroups 从文件加载所有主机和分组的映射关系
func loadHostsFromFileWithGroups(filePath string) ([]hostWithGroup, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 检测是否是 INI 格式
	isINI, err := detectINIFormat(file)
	if err != nil {
		return nil, fmt.Errorf("检测文件格式失败: %w", err)
	}

	// 重置文件指针
	file.Seek(0, 0)

	if isINI {
		return loadHostsFromINIWithGroups(file)
	}

	// 普通格式：没有分组信息，使用空字符串作为分组
	return loadHostsFromPlainWithGroups(file)
}

// loadHostsFromINIWithGroups 从 INI 格式文件加载所有主机和分组的映射关系
func loadHostsFromINIWithGroups(file *os.File) ([]hostWithGroup, error) {
	var hostsWithGroups []hostWithGroup
	scanner := bufio.NewScanner(file)
	sectionPattern := regexp.MustCompile(`^\s*\[(.+)\]\s*$`)
	currentGroup := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查是否是分组标记 [group]
		if matches := sectionPattern.FindStringSubmatch(line); matches != nil {
			currentGroup = strings.TrimSpace(matches[1])
			continue
		}

		// 解析主机行
		host := parseHostLine(line)
		hostsWithGroups = append(hostsWithGroups, hostWithGroup{
			host:  host,
			group: currentGroup,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	return hostsWithGroups, nil
}

// loadHostsFromPlainWithGroups 从普通格式文件加载主机（没有分组信息）
func loadHostsFromPlainWithGroups(file *os.File) ([]hostWithGroup, error) {
	var hostsWithGroups []hostWithGroup
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // 跳过空行和注释
		}

		host := parseHostLine(line)
		hostsWithGroups = append(hostsWithGroups, hostWithGroup{
			host:  host,
			group: "", // 普通格式没有分组
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	return hostsWithGroups, nil
}

// LoadGroupsFromFile 从文件加载所有组名列表
func LoadGroupsFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 检测是否是 INI 格式
	isINI, err := detectINIFormat(file)
	if err != nil {
		return nil, fmt.Errorf("检测文件格式失败: %w", err)
	}

	// 重置文件指针
	file.Seek(0, 0)

	if isINI {
		return loadGroupsFromINI(file)
	}

	// 普通格式没有分组，返回空列表
	return []string{}, nil
}

// loadGroupsFromINI 从 INI 格式文件加载所有组名
func loadGroupsFromINI(file *os.File) ([]string, error) {
	var groups []string
	groupSet := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	sectionPattern := regexp.MustCompile(`^\s*\[(.+)\]\s*$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查是否是分组标记 [group]
		if matches := sectionPattern.FindStringSubmatch(line); matches != nil {
			groupName := strings.TrimSpace(matches[1])
			// 去重
			if !groupSet[groupName] {
				groupSet[groupName] = true
				groups = append(groups, groupName)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	return groups, nil
}

// LoadGroupsFromDirectory 从目录加载所有组名列表（递归读取所有文件）
func LoadGroupsFromDirectory(dirPath string) ([]string, error) {
	// 检查目录是否存在
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("目录不存在或无法访问: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("路径不是目录: %s", dirPath)
	}

	groupSet := make(map[string]bool)
	var allGroups []string

	// 支持的文件扩展名列表
	supportedExts := map[string]bool{
		".ini":   true,
		".txt":   true,
		".conf":  true,
		".hosts": true,
		"":       true, // 无扩展名的文件也支持
	}

	// 遍历目录中的所有文件（递归）
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 跳过隐藏文件（以 . 开头的文件）
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// 检查文件扩展名
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if !supportedExts[ext] {
			return nil
		}

		// 从文件加载组名
		groups, err := LoadGroupsFromFile(path)
		if err != nil {
			// 如果某个文件读取失败，记录错误但继续处理其他文件
			fmt.Fprintf(os.Stderr, "警告: 从文件 %s 加载组名失败: %v\n", path, err)
			return nil
		}

		// 聚合组名并去重
		for _, group := range groups {
			if !groupSet[group] {
				groupSet[group] = true
				allGroups = append(allGroups, group)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历目录失败: %w", err)
	}

	return allGroups, nil
}
