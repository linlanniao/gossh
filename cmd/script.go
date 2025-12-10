package cmd

import (
	"gossh/internal/controller"
	"gossh/internal/view"

	"github.com/spf13/cobra"
)

var (
	scriptInventory   string // 主机列表（文件路径、目录路径或逗号分隔的主机列表）
	scriptGroup       string
	scriptUser        string
	scriptKeyPath     string
	scriptPassword    string
	scriptPort        string
	scriptPath        string
	scriptBecome      bool
	scriptBecomeUser  string
	scriptConcurrency int
	scriptShowOutput  bool
	scriptLogDir      string
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

  # 从文件读取主机列表执行脚本
  gossh script -i hosts.txt -u root -k ~/.ssh/id_rsa -s deploy.sh

  # 从目录读取所有 Ansible hosts 文件并聚合
  gossh script -i ansible_hosts -u root -k ~/.ssh/id_rsa -s deploy.sh

  # 从命令行参数指定主机执行脚本（逗号分隔）
  gossh script -i "192.168.1.10,192.168.1.11" -u root -s deploy.sh

  # 使用 become 模式（sudo）执行脚本
  gossh script -i hosts.txt -u root -s deploy.sh --become
  gossh script -i hosts.txt -u root -s deploy.sh --become --become-user appuser

  # 指定并发数
  gossh script -i hosts.txt -u root -s deploy.sh --concurrency 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建 controller
		ctrl := controller.NewScriptController()

		// 构建请求
		req := &controller.ScriptCommandRequest{
			ConfigFile:  configFile,
			Inventory:   scriptInventory,
			Group:       scriptGroup,
			User:        scriptUser,
			KeyPath:     scriptKeyPath,
			Password:    scriptPassword,
			Port:        scriptPort,
			ScriptPath:  scriptPath,
			Become:      scriptBecome,
			BecomeUser:  scriptBecomeUser,
			Concurrency: scriptConcurrency,
			ShowOutput:  scriptShowOutput,
			LogDir:      scriptLogDir,
		}

		// 执行命令
		resp, err := ctrl.Execute(req)
		if err != nil {
			return err
		}

		// 输出结果
		view.PrintRunResults(resp.Results, resp.TotalDuration, scriptShowOutput)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(scriptCmd)

	// 主机列表相关参数
	scriptCmd.Flags().StringVarP(&scriptInventory, "inventory", "i", "", "主机列表（文件路径、目录路径或逗号分隔的主机列表）。如果指定目录，会递归读取目录下所有子文件并聚合，例如: -i hosts.ini 或 -i hosts_dir/ 或 -i 192.168.1.10,192.168.1.11")
	scriptCmd.Flags().StringVarP(&scriptGroup, "group", "g", "", "Ansible INI 格式的分组名称（仅在使用 -i 参数指定文件或目录时有效），例如: -g test 或 -g web_servers")

	// 认证相关参数
	scriptCmd.Flags().StringVarP(&scriptUser, "user", "u", "", "SSH 用户名（可从 ansible.cfg 的 remote_user 读取）")
	scriptCmd.Flags().StringVarP(&scriptKeyPath, "key", "k", "", "SSH 私钥路径（优先使用，可从 ansible.cfg 的 private_key_file 读取）")
	scriptCmd.Flags().StringVarP(&scriptPassword, "password", "p", "", "SSH 密码（如果未提供 key）")
	scriptCmd.Flags().StringVarP(&scriptPort, "port", "P", "22", "SSH 端口（默认: 22）")

	// 执行相关参数
	scriptCmd.Flags().StringVarP(&scriptPath, "script", "s", "", "要执行的脚本文件路径（必需）")
	scriptCmd.MarkFlagRequired("script")
	scriptCmd.Flags().BoolVar(&scriptBecome, "become", false, "使用 sudo 执行脚本（类似 ansible 的 become）")
	scriptCmd.Flags().StringVar(&scriptBecomeUser, "become-user", "", "使用 sudo 切换到指定用户执行脚本（默认: root）")
	scriptCmd.Flags().IntVar(&scriptConcurrency, "concurrency", 0, "并发执行数量（默认: 5，可从 ansible.cfg 的 forks 读取）")
	scriptCmd.Flags().BoolVar(&scriptShowOutput, "show-output", true, "显示命令输出（默认: true）")
	scriptCmd.Flags().StringVar(&scriptLogDir, "log-dir", "", "日志目录路径（可选，JSON 格式）。会自动生成文件名：script-时间戳.log")
}
