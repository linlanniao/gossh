package cmd

import (
	"gossh/internal/controller"
	"gossh/internal/view"

	"github.com/spf13/cobra"
)

var (
	listFormat string // 输出格式: ip, full, json
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有主机 IP 地址",
	Long: `列出主机列表中的所有主机 IP 地址。

示例:
  # 使用 -g 指定组名，只列出指定分组的主机
  gossh list -i ansible_hosts -g test
  gossh list -i hosts.ini -g web_servers

  # 使用 -g all 列出所有分组的主机
  gossh list -i hosts.txt -g all

  # 从目录列出所有主机
  gossh list -i ansible_hosts -g all

  # 从命令行参数列出主机（逗号分隔，也需要指定 -g）
  gossh list -i "192.168.1.10,192.168.1.11" -g all

  # 指定输出格式（ip/full/json）
  gossh list -i ansible_hosts -g test --format full`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建 controller
		ctrl := controller.NewListController()

		// 构建请求
		req := &controller.ListRequest{
			ConfigFile: configFile,
			Inventory:  inventory,
			Group:      group,
			Format:     listFormat,
		}

		// 执行 list 命令
		resp, err := ctrl.Execute(req)
		if err != nil {
			return err
		}

		// 输出结果
		view.PrintListResults(resp.Hosts, listFormat)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// 输出格式参数
	listCmd.Flags().StringVar(&listFormat, "format", "ip", "输出格式: ip（仅IP地址）、full（完整信息）、json（JSON格式）")
}
