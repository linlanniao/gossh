package cmd

import (
	"gossh/internal/controller"
	"gossh/internal/view"

	"github.com/spf13/cobra"
)

var (
	listHostsFile   string
	listHostsDir    string // Ansible hosts 目录路径
	listHostsString string
	listGroup       string // Ansible INI 格式的分组名称
	listFormat      string // 输出格式: ip, full, json
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有主机 IP 地址",
	Long: `列出主机列表中的所有主机 IP 地址。

示例:
  # 使用 -g 指定组名，只列出指定分组的主机
  gossh list -d ansible_hosts -g test
  gossh list -f hosts.ini -g web_servers

  # 从文件列出所有主机
  gossh list -f hosts.txt

  # 从目录列出所有主机
  gossh list -d ansible_hosts

  # 从命令行参数列出主机
  gossh list -H "192.168.1.10,192.168.1.11"

  # 指定输出格式（ip/full/json）
  gossh list -d ansible_hosts -g test --format full`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建 controller
		ctrl := controller.NewListController()

		// 构建请求
		req := &controller.ListRequest{
			HostsFile:   listHostsFile,
			HostsDir:    listHostsDir,
			HostsString: listHostsString,
			Group:       listGroup,
			Format:      listFormat,
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
	listCmd.Flags().StringVarP(&listHostsFile, "file", "f", "", "主机列表文件路径（支持普通格式和 Ansible INI 格式）")
	listCmd.Flags().StringVarP(&listHostsDir, "dir", "d", "", "Ansible hosts 目录路径（读取目录下所有 .ini 文件并聚合）")
	listCmd.Flags().StringVarP(&listHostsString, "hosts", "H", "", "主机列表（逗号分隔），例如: 192.168.1.10,192.168.1.11")
	listCmd.Flags().StringVarP(&listGroup, "group", "g", "", "Ansible INI 格式的分组名称（仅在使用 -f 或 -d 参数时有效），例如: -g test 或 -g web_servers")

	// 输出格式参数
	listCmd.Flags().StringVar(&listFormat, "format", "ip", "输出格式: ip（仅IP地址）、full（完整信息）、json（JSON格式）")
}

