package cmd

import (
	"gossh/internal/controller"
	"gossh/internal/view"

	"github.com/spf13/cobra"
)

var (
	hostsFile   string
	hostsDir    string // Ansible hosts 目录路径
	hostsString string
	group       string // Ansible INI 格式的分组名称
	user        string
	keyPath     string
	password    string
	port        string
	command     string
	become      bool
	becomeUser  string
	concurrency int
	showOutput  bool
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "批量执行命令",
	Long: `批量 SSH 连接到多台服务器并执行命令。

示例:
  # 使用 -g 指定组名，只对指定分组的主机执行命令
  gossh run -d ansible_hosts -g test -u root -c "df -h"
  gossh run -f hosts.ini -g web_servers -u root -k ~/.ssh/id_rsa -c "uptime"

  # 从文件读取主机列表执行命令
  gossh run -f hosts.txt -u root -k ~/.ssh/id_rsa -c "uptime"

  # 从目录读取所有 Ansible hosts 文件并聚合
  gossh run -d ansible_hosts -u root -k ~/.ssh/id_rsa -c "uptime"

  # 从命令行参数指定主机执行命令
  gossh run -H "192.168.1.10,192.168.1.11" -u root -c "df -h"

  # 使用 become 模式（sudo）执行命令
  gossh run -f hosts.txt -u root -c "systemctl restart nginx" --become
  gossh run -f hosts.txt -u root -c "whoami" --become --become-user appuser

  # 指定并发数
  gossh run -f hosts.txt -u root -c "ls -la" --concurrency 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建 controller
		ctrl := controller.NewRunController()

		// 构建请求
		req := &controller.RunCommandRequest{
			HostsFile:   hostsFile,
			HostsDir:    hostsDir,
			HostsString: hostsString,
			Group:       group,
			User:        user,
			KeyPath:     keyPath,
			Password:    password,
			Port:        port,
			Command:     command,
			Become:      become,
			BecomeUser:  becomeUser,
			Concurrency: concurrency,
			ShowOutput:  showOutput,
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

	// 主机列表相关参数
	runCmd.Flags().StringVarP(&hostsFile, "file", "f", "", "主机列表文件路径（支持普通格式和 Ansible INI 格式）")
	runCmd.Flags().StringVarP(&hostsDir, "dir", "d", "", "Ansible hosts 目录路径（读取目录下所有 .ini 文件并聚合）")
	runCmd.Flags().StringVarP(&hostsString, "hosts", "H", "", "主机列表（逗号分隔），例如: 192.168.1.10,192.168.1.11")
	runCmd.Flags().StringVarP(&group, "group", "g", "", "Ansible INI 格式的分组名称（仅在使用 -f 或 -d 参数时有效），例如: -g test 或 -g web_servers")

	// 认证相关参数
	runCmd.Flags().StringVarP(&user, "user", "u", "", "SSH 用户名（可从 ansible.cfg 的 remote_user 读取）")
	runCmd.Flags().StringVarP(&keyPath, "key", "k", "", "SSH 私钥路径（优先使用，可从 ansible.cfg 的 private_key_file 读取）")
	runCmd.Flags().StringVarP(&password, "password", "p", "", "SSH 密码（如果未提供 key）")
	runCmd.Flags().StringVarP(&port, "port", "P", "22", "SSH 端口（默认: 22）")

	// 执行相关参数
	runCmd.Flags().StringVarP(&command, "command", "c", "", "要执行的命令（必需）")
	runCmd.MarkFlagRequired("command")
	runCmd.Flags().BoolVar(&become, "become", false, "使用 sudo 执行命令（类似 ansible 的 become）")
	runCmd.Flags().StringVar(&becomeUser, "become-user", "", "使用 sudo 切换到指定用户执行命令（默认: root）")
	runCmd.Flags().IntVar(&concurrency, "concurrency", 0, "并发执行数量（默认: 5，可从 ansible.cfg 的 forks 读取）")
	runCmd.Flags().BoolVar(&showOutput, "show-output", true, "显示命令输出（默认: true）")
}
