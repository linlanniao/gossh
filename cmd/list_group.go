package cmd

import (
	"gossh/internal/controller"
	"gossh/internal/view"

	"github.com/spf13/cobra"
)

var (
	listGroupOneLine bool // 是否一行输出（逗号分隔）
)

// listGroupCmd represents the list-group command
var listGroupCmd = &cobra.Command{
	Use:   "list-group",
	Short: "列出所有组名",
	Long: `列出主机列表中的所有组名。

示例:
  # 从文件列出所有组
  gossh list-group -i ansible_hosts
  gossh list-group -i hosts.ini

  # 从目录列出所有组（递归读取所有文件）
  gossh list-group -i hosts_dir/

  # 从 ansible.cfg 配置的 inventory 列出所有组
  gossh list-group`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建 controller
		ctrl := controller.NewListGroupController()

		// 构建请求
		req := &controller.ListGroupRequest{
			ConfigFile: configFile,
			Inventory:  inventory,
		}

		// 执行 list-group 命令
		resp, err := ctrl.Execute(req)
		if err != nil {
			return err
		}

		// 输出结果
		view.PrintGroupResults(resp.Groups, listGroupOneLine)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listGroupCmd)

	// 一行输出参数
	listGroupCmd.Flags().BoolVar(&listGroupOneLine, "one-line", false, "一行输出（逗号分隔）")
}

