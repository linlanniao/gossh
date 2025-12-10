package cmd

import (
	"gossh/internal/controller"
	"gossh/internal/view"

	"github.com/spf13/cobra"
)

var (
	uploadLocalPath  string
	uploadRemotePath string
	uploadMode       string
	uploadShowOutput bool
	uploadLogDir     string
	uploadLimit      int
	uploadOffset     int
	uploadBackup     bool
	uploadForce      bool
)

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "批量上传文件",
	Long: `批量 SSH 连接到多台服务器并上传文件。

示例:
  # 使用 -g 指定组名，只对指定分组的主机上传文件
  gossh upload -i ansible_hosts -g test -u root -l app.tar.gz -r /tmp/app.tar.gz
  gossh upload -i hosts.ini -g web_servers -u root -k ~/.ssh/id_rsa -l config.conf -r /etc/config.conf

  # 使用 -g all 选择所有分组的主机上传文件
  gossh upload -i hosts.txt -g all -u root -k ~/.ssh/id_rsa -l app.tar.gz -r /tmp/app.tar.gz

  # 从目录读取所有 Ansible hosts 文件并聚合
  gossh upload -i ansible_hosts -g all -u root -k ~/.ssh/id_rsa -l app.tar.gz -r /tmp/app.tar.gz

  # 从命令行参数指定主机上传文件（逗号分隔，也需要指定 -g）
  gossh upload -i "192.168.1.10,192.168.1.11" -g all -u root -l app.tar.gz -r /tmp/app.tar.gz

  # 指定文件权限
  gossh upload -i hosts.txt -g all -u root -l script.sh -r /tmp/script.sh --mode 0755

  # 指定并发数
  gossh upload -i hosts.txt -g all -u root -l app.tar.gz -r /tmp/app.tar.gz -f 10

  # 限制执行的主机数量（只执行前 5 台主机）
  gossh upload -i hosts.txt -g all -u root -l app.tar.gz -r /tmp/app.tar.gz --limit 5

  # 跳过前 3 台主机，然后执行接下来的 5 台
  gossh upload -i hosts.txt -g all -u root -l app.tar.gz -r /tmp/app.tar.gz --offset 3 --limit 5

  # 如果文件已存在，先备份再上传（备份文件名格式: 原文件名.backup.YYYYMMDD-HHMMSS）
  gossh upload -i hosts.txt -g all -u root -l config.conf -r /etc/config.conf --backup --force

  # 强制覆盖已存在的文件（不备份）
  gossh upload -i hosts.txt -g all -u root -l app.tar.gz -r /tmp/app.tar.gz --force

  # 默认行为：如果文件已存在则跳过（不覆盖）
  gossh upload -i hosts.txt -g all -u root -l app.tar.gz -r /tmp/app.tar.gz`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建 controller
		ctrl := controller.NewUploadController()

		// 构建请求
		req := &controller.UploadCommandRequest{
			ConfigFile:  configFile,
			Inventory:   inventory,
			Group:       group,
			User:        user,
			KeyPath:     keyPath,
			Password:    password,
			Port:        port,
			LocalPath:   uploadLocalPath,
			RemotePath:  uploadRemotePath,
			Mode:        uploadMode,
			Concurrency: forks,
			ShowOutput:  uploadShowOutput,
			LogDir:      uploadLogDir,
			Limit:       uploadLimit,
			Offset:      uploadOffset,
			Backup:      uploadBackup,
			Force:       uploadForce,
		}

		// 执行命令
		resp, err := ctrl.Execute(req)
		if err != nil {
			return err
		}

		// 输出结果
		view.PrintRunResults(resp.Results, resp.TotalDuration, uploadShowOutput, resp.Group, resp.Hosts)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	// 上传相关参数
	uploadCmd.Flags().StringVarP(&uploadLocalPath, "local", "l", "", "本地文件路径（必需）")
	uploadCmd.MarkFlagRequired("local")
	uploadCmd.Flags().StringVarP(&uploadRemotePath, "remote", "r", "", "远程文件路径（必需）")
	uploadCmd.MarkFlagRequired("remote")
	uploadCmd.Flags().StringVar(&uploadMode, "mode", "0644", "文件权限（默认: 0644）")
	uploadCmd.Flags().BoolVar(&uploadShowOutput, "show-output", true, "显示命令输出（默认: true）")
	uploadCmd.Flags().StringVar(&uploadLogDir, "log-dir", "", "日志目录路径（可选，JSON 格式）。会自动生成文件名：upload-时间戳.log")
	uploadCmd.Flags().IntVar(&uploadLimit, "limit", 0, "限制执行的主机数量（0 表示不限制）")
	uploadCmd.Flags().IntVar(&uploadOffset, "offset", 0, "跳过前 N 台主机（默认: 0）")
	uploadCmd.Flags().BoolVar(&uploadBackup, "backup", false, "如果文件已存在，先备份再上传（备份文件名格式: 原文件名.backup.YYYYMMDD-HHMMSS）")
	uploadCmd.Flags().BoolVar(&uploadForce, "force", false, "强制覆盖已存在的文件（默认: false，遇到已存在的文件会跳过）")
}
