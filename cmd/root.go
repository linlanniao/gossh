/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
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
  gossh run -f hosts.txt -u root -k ~/.ssh/id_rsa -c "uptime"
  gossh run -H "192.168.1.10,192.168.1.11" -u root -c "df -h"`,
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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gossh.yaml)")
}
