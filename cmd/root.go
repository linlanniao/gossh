/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// configFile 配置文件路径（全局变量，所有子命令都可以访问）
var configFile string

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
  gossh run -i hosts.txt -u root -k ~/.ssh/id_rsa -c "uptime"
  gossh run -i "192.168.1.10,192.168.1.11" -u root -c "df -h"`,
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
	// 添加全局配置文件参数（所有子命令都可以使用）
	// 类似 ansible 的 ANSIBLE_CONFIG 环境变量，但通过命令行参数指定
	rootCmd.PersistentFlags().StringVar(&configFile, "config-file", "", "指定 ansible.cfg 配置文件路径。如果未指定，将按以下顺序查找：1) 环境变量 ANSIBLE_CONFIG 2) 当前目录及父目录的 ansible.cfg 3) ~/.ansible.cfg")
}
