package cmd

import (
	"gossh/internal/controller"
	"gossh/internal/view"

	"github.com/spf13/cobra"
)

var (
	command    string
	become     bool
	becomeUser string
	showOutput bool
	logDir     string
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "批量执行命令",
	Long: `批量 SSH 连接到多台服务器并执行命令。

示例:
  # 使用 -g 指定组名，只对指定分组的主机执行命令
  gossh run -i ansible_hosts -g test -u root -c "df -h"
  gossh run -i hosts.ini -g web_servers -u root -k ~/.ssh/id_rsa -c "uptime"

  # 使用 -g all 选择所有分组的主机
  gossh run -i hosts.txt -g all -u root -k ~/.ssh/id_rsa -c "uptime"

  # 从目录读取所有 Ansible hosts 文件并聚合，选择所有分组
  gossh run -i ansible_hosts -g all -u root -k ~/.ssh/id_rsa -c "uptime"

  # 从命令行参数指定主机执行命令（逗号分隔，也需要指定 -g）
  gossh run -i "192.168.1.10,192.168.1.11" -g all -u root -c "df -h"

  # 使用 become 模式（sudo）执行命令
  gossh run -i hosts.txt -g all -u root -c "systemctl restart nginx" --become
  gossh run -i hosts.txt -g all -u root -c "whoami" --become --become-user appuser

  # 指定并发数
  gossh run -i hosts.txt -g all -u root -c "ls -la" -f 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建 controller
		ctrl := controller.NewRunController()

		// 构建请求
		req := &controller.RunCommandRequest{
			ConfigFile:  configFile,
			Inventory:   inventory,
			Group:       group,
			User:        user,
			KeyPath:     keyPath,
			Password:    password,
			Port:        port,
			Command:     command,
			Become:      become,
			BecomeUser:  becomeUser,
			Concurrency: forks,
			ShowOutput:  showOutput,
			LogDir:      logDir,
		}

		// 执行命令
		resp, err := ctrl.Execute(req)
		if err != nil {
			return err
		}

		// 输出结果
		view.PrintRunResults(resp.Results, resp.TotalDuration, showOutput)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// 执行相关参数
	runCmd.Flags().StringVarP(&command, "command", "c", "", "要执行的命令（必需）")
	runCmd.MarkFlagRequired("command")
	runCmd.Flags().BoolVar(&become, "become", false, "使用 sudo 执行命令（类似 ansible 的 become）")
	runCmd.Flags().StringVar(&becomeUser, "become-user", "", "使用 sudo 切换到指定用户执行命令（默认: root）")
	runCmd.Flags().BoolVar(&showOutput, "show-output", true, "显示命令输出（默认: true）")
	runCmd.Flags().StringVar(&logDir, "log-dir", "", "日志目录路径（可选，JSON 格式）。会自动生成文件名：run-时间戳.log")
}
