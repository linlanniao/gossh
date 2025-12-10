package cmd

import (
	"gossh/internal/controller"
	"gossh/internal/view"

	"github.com/spf13/cobra"
)

var (
	listInventory string // 主机列表（文件路径、目录路径或逗号分隔的主机列表）
	listGroup     string // Ansible INI 格式的分组名称
	listFormat    string // 输出格式: ip, full, json
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

  # 从文件列出所有主机
  gossh list -i hosts.txt

  # 从目录列出所有主机
  gossh list -i ansible_hosts

  # 从命令行参数列出主机（逗号分隔）
  gossh list -i "192.168.1.10,192.168.1.11"

  # 指定输出格式（ip/full/json）
  gossh list -i ansible_hosts -g test --format full`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建 controller
		ctrl := controller.NewListController()

		// 构建请求
		req := &controller.ListRequest{
			ConfigFile: configFile,
			Inventory:  listInventory,
			Group:      listGroup,
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

	// 主机列表相关参数
	listCmd.Flags().StringVarP(&listInventory, "inventory", "i", "", "主机列表（文件路径、目录路径或逗号分隔的主机列表）。如果指定目录，会递归读取目录下所有子文件并聚合，例如: -i hosts.ini 或 -i hosts_dir/ 或 -i 192.168.1.10,192.168.1.11")
	listCmd.Flags().StringVarP(&listGroup, "group", "g", "", "Ansible INI 格式的分组名称（仅在使用 -i 参数指定文件或目录时有效），例如: -g test 或 -g web_servers")

	// 输出格式参数
	listCmd.Flags().StringVar(&listFormat, "format", "ip", "输出格式: ip（仅IP地址）、full（完整信息）、json（JSON格式）")
}
