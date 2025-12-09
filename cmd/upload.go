package cmd

import (
	"gossh/internal/controller"
	"gossh/internal/view"

	"github.com/spf13/cobra"
)

var (
	uploadHostsFile   string
	uploadHostsDir    string
	uploadHostsString string
	uploadGroup       string
	uploadUser        string
	uploadKeyPath     string
	uploadPassword    string
	uploadPort        string
	uploadLocalPath   string
	uploadRemotePath  string
	uploadMode        string
	uploadConcurrency int
	uploadShowOutput  bool
	uploadLogDir      string
)

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "批量上传文件",
	Long: `批量 SSH 连接到多台服务器并上传文件。

示例:
  # 使用 -g 指定组名，只对指定分组的主机上传文件
  gossh upload -d ansible_hosts -g test -u root -l app.tar.gz -r /tmp/app.tar.gz
  gossh upload -f hosts.ini -g web_servers -u root -k ~/.ssh/id_rsa -l config.conf -r /etc/config.conf

  # 从文件读取主机列表上传文件
  gossh upload -f hosts.txt -u root -k ~/.ssh/id_rsa -l app.tar.gz -r /tmp/app.tar.gz

  # 从目录读取所有 Ansible hosts 文件并聚合
  gossh upload -d ansible_hosts -u root -k ~/.ssh/id_rsa -l app.tar.gz -r /tmp/app.tar.gz

  # 从命令行参数指定主机上传文件
  gossh upload -H "192.168.1.10,192.168.1.11" -u root -l app.tar.gz -r /tmp/app.tar.gz

  # 指定文件权限
  gossh upload -f hosts.txt -u root -l script.sh -r /tmp/script.sh --mode 0755

  # 指定并发数
  gossh upload -f hosts.txt -u root -l app.tar.gz -r /tmp/app.tar.gz --concurrency 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建 controller
		ctrl := controller.NewUploadController()

		// 构建请求
		req := &controller.UploadCommandRequest{
			HostsFile:   uploadHostsFile,
			HostsDir:    uploadHostsDir,
			HostsString: uploadHostsString,
			Group:       uploadGroup,
			User:        uploadUser,
			KeyPath:     uploadKeyPath,
			Password:    uploadPassword,
			Port:        uploadPort,
			LocalPath:   uploadLocalPath,
			RemotePath:  uploadRemotePath,
			Mode:        uploadMode,
			Concurrency: uploadConcurrency,
			ShowOutput:  uploadShowOutput,
			LogDir:      uploadLogDir,
		}

		// 执行命令
		resp, err := ctrl.Execute(req)
		if err != nil {
			return err
		}

		// 输出结果
		view.PrintRunResults(resp.Results, resp.TotalDuration, uploadShowOutput)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	// 主机列表相关参数
	uploadCmd.Flags().StringVarP(&uploadHostsFile, "file", "f", "", "主机列表文件路径（支持普通格式和 Ansible INI 格式）")
	uploadCmd.Flags().StringVarP(&uploadHostsDir, "dir", "d", "", "Ansible hosts 目录路径（读取目录下所有 .ini 文件并聚合）")
	uploadCmd.Flags().StringVarP(&uploadHostsString, "hosts", "H", "", "主机列表（逗号分隔），例如: 192.168.1.10,192.168.1.11")
	uploadCmd.Flags().StringVarP(&uploadGroup, "group", "g", "", "Ansible INI 格式的分组名称（仅在使用 -f 或 -d 参数时有效），例如: -g test 或 -g web_servers")

	// 认证相关参数
	uploadCmd.Flags().StringVarP(&uploadUser, "user", "u", "", "SSH 用户名（可从 ansible.cfg 的 remote_user 读取）")
	uploadCmd.Flags().StringVarP(&uploadKeyPath, "key", "k", "", "SSH 私钥路径（优先使用，可从 ansible.cfg 的 private_key_file 读取）")
	uploadCmd.Flags().StringVarP(&uploadPassword, "password", "p", "", "SSH 密码（如果未提供 key）")
	uploadCmd.Flags().StringVarP(&uploadPort, "port", "P", "22", "SSH 端口（默认: 22）")

	// 上传相关参数
	uploadCmd.Flags().StringVarP(&uploadLocalPath, "local", "l", "", "本地文件路径（必需）")
	uploadCmd.MarkFlagRequired("local")
	uploadCmd.Flags().StringVarP(&uploadRemotePath, "remote", "r", "", "远程文件路径（必需）")
	uploadCmd.MarkFlagRequired("remote")
	uploadCmd.Flags().StringVar(&uploadMode, "mode", "0644", "文件权限（默认: 0644）")
	uploadCmd.Flags().IntVar(&uploadConcurrency, "concurrency", 0, "并发执行数量（默认: 5，可从 ansible.cfg 的 forks 读取）")
	uploadCmd.Flags().BoolVar(&uploadShowOutput, "show-output", true, "显示命令输出（默认: true）")
	uploadCmd.Flags().StringVar(&uploadLogDir, "log-dir", "", "日志目录路径（可选，JSON 格式）。会自动生成文件名：upload-时间戳.log")
}
