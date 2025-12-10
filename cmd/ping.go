package cmd

import (
	"gossh/internal/controller"
	"gossh/internal/view"

	"github.com/spf13/cobra"
)

// ping 命令没有额外的局部变量，全部使用全局变量

// pingCmd represents the ping command
var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "测试 SSH 连接是否成功",
	Long: `批量测试多台服务器的 SSH 连接是否可用。

示例:
  # 使用 -g 指定组名，只测试指定分组的主机
  gossh ping -i ansible_hosts -g test -u root
  gossh ping -i hosts.ini -g web_servers -u root -k ~/.ssh/id_rsa

  # 使用 -g all 测试所有分组的主机
  gossh ping -i hosts.txt -g all -u root -k ~/.ssh/id_rsa

  # 从目录读取所有 Ansible hosts 文件并聚合测试
  gossh ping -i ansible_hosts -g all -u root -k ~/.ssh/id_rsa

  # 从命令行参数指定主机测试连接（逗号分隔，也需要指定 -g）
  gossh ping -i "192.168.1.10,192.168.1.11" -g all -u root

  # 指定并发数
  gossh ping -i hosts.txt -g all -u root -f 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建 controller
		ctrl := controller.NewPingController()

		// 构建请求
		req := &controller.PingRequest{
			ConfigFile:  configFile,
			Inventory:   inventory,
			Group:       group,
			User:        user,
			KeyPath:     keyPath,
			Password:    password,
			Port:        port,
			Concurrency: forks,
			Timeout:     timeout,
		}

		// 执行 ping 测试
		resp, err := ctrl.Execute(req)
		if err != nil {
			return err
		}

		// 输出结果
		view.PrintPingResults(resp.Results, resp.TotalDuration, resp.Group, resp.Hosts)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
	// ping 命令使用全局参数，无需额外定义
}
