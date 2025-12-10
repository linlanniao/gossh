package cmd

import (
	"gossh/internal/controller"
	"gossh/internal/view"
	"time"

	"github.com/spf13/cobra"
)

var (
	pingInventory   string // 主机列表（文件路径、目录路径或逗号分隔的主机列表）
	pingGroup       string // Ansible INI 格式的分组名称
	pingUser        string
	pingKeyPath     string
	pingPassword    string
	pingPort        string
	pingConcurrency int
	pingTimeout     time.Duration // 连接超时时间
)

// pingCmd represents the ping command
var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "测试 SSH 连接是否成功",
	Long: `批量测试多台服务器的 SSH 连接是否可用。

示例:
  # 使用 -g 指定组名，只测试指定分组的主机
  gossh ping -i ansible_hosts -g test -u root
  gossh ping -i hosts.ini -g web_servers -u root -k ~/.ssh/id_rsa

  # 从文件读取主机列表测试连接
  gossh ping -i hosts.txt -u root -k ~/.ssh/id_rsa

  # 从目录读取所有 Ansible hosts 文件并聚合测试
  gossh ping -i ansible_hosts -u root -k ~/.ssh/id_rsa

  # 从命令行参数指定主机测试连接（逗号分隔）
  gossh ping -i "192.168.1.10,192.168.1.11" -u root

  # 指定并发数
  gossh ping -i hosts.txt -u root --concurrency 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建 controller
		ctrl := controller.NewPingController()

		// 构建请求
		req := &controller.PingRequest{
			ConfigFile:  configFile,
			Inventory:   pingInventory,
			Group:       pingGroup,
			User:        pingUser,
			KeyPath:     pingKeyPath,
			Password:    pingPassword,
			Port:        pingPort,
			Concurrency: pingConcurrency,
			Timeout:     pingTimeout,
		}

		// 执行 ping 测试
		resp, err := ctrl.Execute(req)
		if err != nil {
			return err
		}

		// 输出结果
		view.PrintPingResults(resp.Results, resp.TotalDuration)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)

	// 主机列表相关参数
	pingCmd.Flags().StringVarP(&pingInventory, "inventory", "i", "", "主机列表（文件路径、目录路径或逗号分隔的主机列表）。如果指定目录，会递归读取目录下所有子文件并聚合，例如: -i hosts.ini 或 -i hosts_dir/ 或 -i 192.168.1.10,192.168.1.11")
	pingCmd.Flags().StringVarP(&pingGroup, "group", "g", "", "Ansible INI 格式的分组名称（仅在使用 -i 参数指定文件或目录时有效），例如: -g test 或 -g web_servers")

	// 认证相关参数
	pingCmd.Flags().StringVarP(&pingUser, "user", "u", "", "SSH 用户名（可从 ansible.cfg 的 remote_user 读取）")
	pingCmd.Flags().StringVarP(&pingKeyPath, "key", "k", "", "SSH 私钥路径（优先使用，可从 ansible.cfg 的 private_key_file 读取）")
	pingCmd.Flags().StringVarP(&pingPassword, "password", "p", "", "SSH 密码（如果未提供 key）")
	pingCmd.Flags().StringVarP(&pingPort, "port", "P", "22", "SSH 端口（默认: 22）")

	// 执行相关参数
	pingCmd.Flags().IntVar(&pingConcurrency, "concurrency", 0, "并发执行数量（默认: 5，可从 ansible.cfg 的 forks 读取）")
	pingCmd.Flags().DurationVar(&pingTimeout, "timeout", 0, "连接超时时间（默认: 30s，可从 ansible.cfg 的 timeout 读取），例如: 30s, 1m, 2m30s")
}
