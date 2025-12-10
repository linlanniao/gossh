package cmd

import (
	"gossh/internal/controller"
	"gossh/internal/view"

	"github.com/spf13/cobra"
)

var (
	scriptPath       string
	scriptBecome     bool
	scriptBecomeUser string
	scriptShowOutput bool
	scriptLogDir     string
)

// scriptCmd represents the script command
var scriptCmd = &cobra.Command{
	Use:   "script",
	Short: "批量执行脚本文件",
	Long: `批量 SSH 连接到多台服务器并执行脚本文件。
脚本会先上传到远程主机的临时目录，然后执行，执行完成后自动清理临时文件。

示例:
  # 使用 -g 指定组名，只对指定分组的主机执行脚本
  gossh script -i ansible_hosts -g test -u root -s deploy.sh
  gossh script -i hosts.ini -g web_servers -u root -k ~/.ssh/id_rsa -s backup.sh

  # 使用 -g all 选择所有分组的主机执行脚本
  gossh script -i hosts.txt -g all -u root -k ~/.ssh/id_rsa -s deploy.sh

  # 从目录读取所有 Ansible hosts 文件并聚合
  gossh script -i ansible_hosts -g all -u root -k ~/.ssh/id_rsa -s deploy.sh

  # 从命令行参数指定主机执行脚本（逗号分隔，也需要指定 -g）
  gossh script -i "192.168.1.10,192.168.1.11" -g all -u root -s deploy.sh

  # 使用 become 模式（sudo）执行脚本
  gossh script -i hosts.txt -g all -u root -s deploy.sh --become
  gossh script -i hosts.txt -g all -u root -s deploy.sh --become --become-user appuser

  # 指定并发数
  gossh script -i hosts.txt -g all -u root -s deploy.sh -f 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建 controller
		ctrl := controller.NewScriptController()

		// 构建请求
		req := &controller.ScriptCommandRequest{
			ConfigFile:  configFile,
			Inventory:   inventory,
			Group:       group,
			User:        user,
			KeyPath:     keyPath,
			Password:    password,
			Port:        port,
			ScriptPath:  scriptPath,
			Become:      scriptBecome,
			BecomeUser:  scriptBecomeUser,
			Concurrency: forks,
			ShowOutput:  scriptShowOutput,
			LogDir:      scriptLogDir,
		}

		// 执行命令
		resp, err := ctrl.Execute(req)
		if err != nil {
			return err
		}

		// 输出结果
		view.PrintRunResults(resp.Results, resp.TotalDuration, scriptShowOutput, resp.Group, resp.Hosts)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(scriptCmd)

	// 执行相关参数
	scriptCmd.Flags().StringVarP(&scriptPath, "script", "s", "", "要执行的脚本文件路径（必需）")
	scriptCmd.MarkFlagRequired("script")
	scriptCmd.Flags().BoolVar(&scriptBecome, "become", false, "使用 sudo 执行脚本（类似 ansible 的 become）")
	scriptCmd.Flags().StringVar(&scriptBecomeUser, "become-user", "", "使用 sudo 切换到指定用户执行脚本（默认: root）")
	scriptCmd.Flags().BoolVar(&scriptShowOutput, "show-output", true, "显示命令输出（默认: true）")
	scriptCmd.Flags().StringVar(&scriptLogDir, "log-dir", "", "日志目录路径（可选，JSON 格式）。会自动生成文件名：script-时间戳.log")
}
