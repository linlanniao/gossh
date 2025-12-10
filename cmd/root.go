/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// 全局参数（所有子命令都可以访问）
var (
	configFile string        // 配置文件路径
	inventory  string        // 主机列表（文件路径、目录路径或逗号分隔的主机列表）
	group      string        // Ansible INI 格式的分组名称
	user       string        // SSH 用户名
	keyPath    string        // SSH 私钥路径
	password   string        // SSH 密码
	port       string        // SSH 端口
	forks      int           // 并发数（类似 ansible 的 -f --forks）
	timeout    time.Duration // 连接超时时间（类似 ansible 的 -T --timeout）
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gossh",
	Short: "批量 SSH 连接工具",
	Long: `gossh 是一个类似 ansible 的批量 SSH 连接工具，支持批量连接到多台 Linux 服务器执行命令或脚本。

主要特性:
  - 支持 SSH key 认证和密码认证
  - 并发执行，提高效率
  - 支持从文件或命令行参数读取主机列表
  - 支持执行命令和脚本文件
  - 详细的执行结果输出

使用示例:
  gossh run -i hosts.txt -g all -u root -k ~/.ssh/id_rsa -c "uptime"
  gossh run -i "192.168.1.10,192.168.1.11" -g all -u root -c "df -h"`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// list-group 命令不需要 group 参数，跳过验证
		if cmd.Name() == "list-group" {
			return nil
		}
		// 如果 inventory 是文件或目录路径，则需要 group 参数
		// 如果 inventory 是IP地址（直接指定），则不需要 group 参数
		if isInventoryFileOrDir(inventory) && group == "" {
			return fmt.Errorf("必须指定分组名称 (-g)。使用 -g all 表示选择所有分组，支持逗号分隔的多个组，例如: -g test,web_servers")
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// 配置文件参数
	rootCmd.PersistentFlags().StringVar(&configFile, "config-file", "", "指定 ansible.cfg 配置文件路径。如果未指定，将按以下顺序查找：1) 环境变量 ANSIBLE_CONFIG 2) 当前目录及父目录的 ansible.cfg 3) ~/.ansible.cfg")

	// 主机列表相关参数
	rootCmd.PersistentFlags().StringVarP(&inventory, "inventory", "i", "", "主机列表（文件路径、目录路径或逗号分隔的主机列表）。如果指定目录，会递归读取目录下所有子文件并聚合，例如: -i hosts.ini 或 -i hosts_dir/ 或 -i 192.168.1.10,192.168.1.11")
	rootCmd.PersistentFlags().StringVarP(&group, "group", "g", "", "Ansible INI 格式的分组名称（必需）。使用 -g all 表示选择所有分组，支持逗号分隔的多个组，例如: -g test 或 -g web_servers 或 -g all 或 -g test,web_servers")

	// 认证相关参数
	rootCmd.PersistentFlags().StringVarP(&user, "user", "u", "", "SSH 用户名（可从 ansible.cfg 的 remote_user 读取）")
	rootCmd.PersistentFlags().StringVarP(&keyPath, "key", "k", "", "SSH 私钥路径（优先使用，可从 ansible.cfg 的 private_key_file 读取）")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "SSH 密码（如果未提供 key）")
	rootCmd.PersistentFlags().StringVarP(&port, "port", "P", "22", "SSH 端口（默认: 22）")

	// 执行相关参数
	rootCmd.PersistentFlags().IntVarP(&forks, "forks", "f", 0, "并发执行数量（默认: 5，可从 ansible.cfg 的 forks 读取）")
	rootCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "T", 0, "连接超时时间（默认: 30s，可从 ansible.cfg 的 timeout 读取），例如: 30s, 1m, 2m30s")
}

// isInventoryFileOrDir 判断 inventory 是否是文件或目录路径
// 如果 inventory 是文件或目录路径，返回 true
// 如果 inventory 是IP地址或逗号分隔的主机列表，返回 false
func isInventoryFileOrDir(inventory string) bool {
	if inventory == "" {
		return false
	}
	// 检查是否是文件或目录路径
	_, err := os.Stat(inventory)
	return err == nil
}
