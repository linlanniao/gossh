package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gossh/internal/executor"
)

// LoadHostGroupsMap 加载主机到分组的映射关系
// 返回 map[string][]string，key 是主机地址（格式: "address:port"），value 是该主机所属的所有分组列表
// 如果 inventory 是文件或目录，从文件中读取；如果是逗号分隔的IP地址，返回空映射
func LoadHostGroupsMap(inventory, group string) (map[string][]string, error) {
	hostGroupsMap := make(map[string][]string)

	// 判断是文件、目录还是逗号分隔的主机列表
	info, err := os.Stat(inventory)
	if err == nil {
		// 路径存在，是文件或目录
		if info.IsDir() {
			return loadHostGroupsMapFromDirectory(inventory, group)
		} else {
			return loadHostGroupsMapFromFile(inventory, group)
		}
	}

	// 路径不存在，可能是逗号分隔的主机列表，没有分组信息
	if strings.Contains(inventory, ",") {
		return hostGroupsMap, nil
	}

	// 单个IP地址，也没有分组信息
	return hostGroupsMap, nil
}

// loadHostGroupsMapFromFile 从文件加载主机到分组的映射关系
func loadHostGroupsMapFromFile(filePath, group string) (map[string][]string, error) {
	hostGroupsMap := make(map[string][]string)

	// 加载所有主机和分组的映射关系
	hostsWithGroups, err := loadHostsFromFileWithGroups(filePath)
	if err != nil {
		return nil, err
	}

	// 解析组列表
	targetGroups := parseGroups(group)

	// 构建主机到分组的映射（一个主机可能属于多个分组）
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

	return hostGroupsMap, nil
}

// loadHostGroupsMapFromDirectory 从目录加载主机到分组的映射关系
func loadHostGroupsMapFromDirectory(dirPath, group string) (map[string][]string, error) {
	hostGroupsMap := make(map[string][]string)

	// 解析组列表
	targetGroups := parseGroups(group)

	// 支持的文件扩展名列表
	supportedExts := map[string]bool{
		".ini":   true,
		".txt":   true,
		".conf":  true,
		".hosts": true,
		"":       true, // 无扩展名的文件也支持
	}

	// 遍历目录中的所有文件（递归）
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 跳过隐藏文件
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// 检查文件扩展名
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if !supportedExts[ext] {
			return nil
		}

		// 加载文件中的所有主机和分组信息
		hostsWithGroups, err := loadHostsFromFileWithGroups(path)
		if err != nil {
			// 如果某个文件读取失败，记录错误但继续处理其他文件
			fmt.Fprintf(os.Stderr, "警告: 读取文件 %s 失败: %v\n", path, err)
			return nil
		}

		// 构建主机到分组的映射
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

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历目录失败: %w", err)
	}

	return hostGroupsMap, nil
}

// GetHostGroups 获取主机所属的分组列表
// host 是主机地址（IP或域名），port 是端口，hostGroupsMap 是主机到分组的映射
func GetHostGroups(host executor.Host, hostGroupsMap map[string][]string) []string {
	key := fmt.Sprintf("%s:%s", host.Address, host.Port)
	groups, exists := hostGroupsMap[key]
	if !exists || len(groups) == 0 {
		return []string{}
	}
	return groups
}

// FormatHostGroups 格式化主机分组列表为字符串
// 如果分组列表为空，返回 "-"
func FormatHostGroups(groups []string) string {
	if len(groups) == 0 {
		return "-"
	}
	return strings.Join(groups, ",")
}

