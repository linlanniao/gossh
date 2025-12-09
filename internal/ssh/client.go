package ssh

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"golang.org/x/crypto/ssh"
)

// Client 封装 SSH 客户端
type Client struct {
	config *ssh.ClientConfig
	host   string
	port   string
}

// NewClient 创建新的 SSH 客户端
func NewClient(host, port, user, keyPath, password string) (*Client, error) {
	var authMethod ssh.AuthMethod

	// 优先使用 SSH key 认证
	if keyPath != "" {
		key, err := loadPrivateKey(keyPath)
		if err != nil {
			return nil, fmt.Errorf("加载 SSH key 失败: %w", err)
		}
		authMethod = ssh.PublicKeys(key)
	} else if password != "" {
		authMethod = ssh.Password(password)
	} else {
		// 尝试使用默认的 SSH key
		defaultKeyPath := filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
		if key, err := loadPrivateKey(defaultKeyPath); err == nil {
			authMethod = ssh.PublicKeys(key)
		} else {
			return nil, fmt.Errorf("未提供认证方式（key 或 password）")
		}
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应验证 host key
		Timeout:         10 * time.Second,
	}

	return &Client{
		config: config,
		host:   host,
		port:   port,
	}, nil
}

// Execute 执行命令并返回结果
func (c *Client) Execute(command string) (*Result, error) {
	return c.ExecuteWithBecome(command, false, "")
}

// ExecuteWithBecome 执行命令并返回结果，支持 become 模式（类似 sudo）
func (c *Client) ExecuteWithBecome(command string, become bool, becomeUser string) (*Result, error) {
	startTime := time.Now()

	conn, err := c.createSSHConnection()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	session, err := c.createSession(conn)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	stdout, stderr, err := c.setupPipes(session)
	if err != nil {
		return nil, err
	}

	finalCommand := c.buildCommand(command, become, becomeUser)
	if err := session.Start(finalCommand); err != nil {
		return nil, fmt.Errorf("启动命令失败: %w", err)
	}

	output, _ := io.ReadAll(stdout)
	errOutput, _ := io.ReadAll(stderr)

	exitCode := c.waitForCommand(session)
	duration := time.Since(startTime)

	return &Result{
		Host:     c.host,
		Command:  command,
		Stdout:   string(output),
		Stderr:   string(errOutput),
		ExitCode: exitCode,
		Duration: duration,
		Error:    err,
	}, nil
}

// createSSHConnection 创建 SSH 连接
func (c *Client) createSSHConnection() (*ssh.Client, error) {
	address := fmt.Sprintf("%s:%s", c.host, c.port)
	conn, err := ssh.Dial("tcp", address, c.config)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}
	return conn, nil
}

// createSession 创建 SSH 会话
func (c *Client) createSession(conn *ssh.Client) (*ssh.Session, error) {
	session, err := conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}
	return session, nil
}

// setupPipes 设置标准输出和标准错误管道
func (c *Client) setupPipes(session *ssh.Session) (io.Reader, io.Reader, error) {
	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("获取标准输出失败: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("获取标准错误失败: %w", err)
	}

	return stdout, stderr, nil
}

// buildCommand 构建最终执行的命令（支持 become 模式）
func (c *Client) buildCommand(command string, become bool, becomeUser string) string {
	if !become {
		return command
	}

	if becomeUser != "" && becomeUser != "root" {
		return fmt.Sprintf("sudo -u %s %s", becomeUser, command)
	}

	return fmt.Sprintf("sudo %s", command)
}

// waitForCommand 等待命令完成并返回退出码
func (c *Client) waitForCommand(session *ssh.Session) int {
	err := session.Wait()
	if err != nil {
		if exitError, ok := err.(*ssh.ExitError); ok {
			return exitError.ExitStatus()
		}
	}
	return 0
}

// ExecuteScript 执行脚本文件（先上传到临时目录再执行）
func (c *Client) ExecuteScript(scriptPath string) (*Result, error) {
	return c.ExecuteScriptWithBecome(scriptPath, false, "")
}

// ExecuteScriptWithBecome 执行脚本文件（先上传到临时目录再执行），支持 become 模式
func (c *Client) ExecuteScriptWithBecome(scriptPath string, become bool, becomeUser string) (*Result, error) {
	startTime := time.Now()

	// 生成唯一的临时文件名（使用时间戳和随机数）
	tempFileName := fmt.Sprintf("/tmp/gossh_script_%d_%d.sh", time.Now().UnixNano(), os.Getpid())

	// 使用 UploadFile 方法上传脚本文件
	_, err := c.UploadFile(scriptPath, tempFileName, "0755")
	if err != nil {
		return &Result{
			Host:     c.host,
			Command:  scriptPath,
			Stdout:   "",
			Stderr:   fmt.Sprintf("上传脚本失败: %v", err),
			ExitCode: -1,
			Duration: time.Since(startTime),
			Error:    err,
		}, err
	}

	// 获取 SSH 连接用于清理临时文件
	address := fmt.Sprintf("%s:%s", c.host, c.port)
	conn, err := ssh.Dial("tcp", address, c.config)
	if err != nil {
		// 如果连接失败，仍然尝试执行脚本
		conn = nil
	} else {
		defer conn.Close()
	}

	// 执行脚本
	executeCommand := fmt.Sprintf("bash %s", tempFileName)
	result, err := c.ExecuteWithBecome(executeCommand, become, becomeUser)
	if err != nil {
		// 即使执行失败，也尝试清理临时文件
		if conn != nil {
			c.cleanupTempFile(conn, tempFileName)
		}
		return result, err
	}

	// 清理临时文件
	if conn != nil {
		cleanupResult := c.cleanupTempFile(conn, tempFileName)
		if cleanupResult != nil && result.ExitCode == 0 {
			// 如果清理失败但原命令成功，在 stderr 中记录
			result.Stderr += fmt.Sprintf("\n警告: 清理临时文件失败: %v", cleanupResult)
		}
	}

	// 更新总耗时
	result.Duration = time.Since(startTime)

	return result, nil
}

// cleanupTempFile 清理临时文件
func (c *Client) cleanupTempFile(conn *ssh.Client, filePath string) error {
	session, err := conn.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 执行删除命令，忽略错误
	_ = session.Run(fmt.Sprintf("rm -f %s", filePath))
	return nil
}

// UploadFile 上传文件到远程主机
func (c *Client) UploadFile(localPath string, remotePath string, mode string) (*Result, error) {
	startTime := time.Now()
	command := fmt.Sprintf("upload %s -> %s", localPath, remotePath)

	localFile, err := c.openLocalFile(localPath)
	if err != nil {
		return c.createErrorResult(command, startTime, err, "打开本地文件失败"), err
	}
	defer localFile.Close()

	conn, err := c.createSSHConnection()
	if err != nil {
		return c.createErrorResult(command, startTime, err, "连接失败"), err
	}
	defer conn.Close()

	scpClient, err := c.createSCPClient(conn)
	if err != nil {
		return c.createErrorResult(command, startTime, err, "创建 SCP 客户端失败"), err
	}
	defer scpClient.Close()

	mode = c.normalizeFileMode(mode)
	if err := c.copyFile(scpClient, localFile, remotePath, mode); err != nil {
		return c.createErrorResult(command, startTime, err, "上传文件失败"), err
	}

	return &Result{
		Host:     c.host,
		Command:  command,
		Stdout:   fmt.Sprintf("文件已成功上传到 %s", remotePath),
		Stderr:   "",
		ExitCode: 0,
		Duration: time.Since(startTime),
		Error:    nil,
	}, nil
}

// openLocalFile 打开本地文件并重置文件指针
func (c *Client) openLocalFile(localPath string) (*os.File, error) {
	localFile, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("打开本地文件失败: %w", err)
	}

	// 确保文件指针在开头
	if _, err := localFile.Seek(0, 0); err != nil {
		localFile.Close()
		return nil, fmt.Errorf("重置文件指针失败: %w", err)
	}

	return localFile, nil
}

// createSCPClient 创建 SCP 客户端
func (c *Client) createSCPClient(conn *ssh.Client) (scp.Client, error) {
	scpClient, err := scp.NewClientBySSH(conn)
	if err != nil {
		return scpClient, fmt.Errorf("创建 SCP 客户端失败: %w", err)
	}
	return scpClient, nil
}

// normalizeFileMode 规范化文件权限模式
func (c *Client) normalizeFileMode(mode string) string {
	if mode == "" {
		return "0644"
	}
	return mode
}

// copyFile 使用 SCP 客户端复制文件
func (c *Client) copyFile(scpClient scp.Client, localFile *os.File, remotePath, mode string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return scpClient.CopyFromFile(ctx, *localFile, remotePath, mode)
}

// createErrorResult 创建错误结果对象
func (c *Client) createErrorResult(command string, startTime time.Time, err error, message string) *Result {
	return &Result{
		Host:     c.host,
		Command:  command,
		Stdout:   "",
		Stderr:   fmt.Sprintf("%s: %v", message, err),
		ExitCode: -1,
		Duration: time.Since(startTime),
		Error:    err,
	}
}

// Ping 测试 SSH 连接是否成功，返回连接延迟和错误信息
func (c *Client) Ping() (*PingResult, error) {
	startTime := time.Now()
	address := fmt.Sprintf("%s:%s", c.host, c.port)

	conn, err := ssh.Dial("tcp", address, c.config)
	duration := time.Since(startTime)

	if err != nil {
		return &PingResult{
			Host:     c.host,
			Success:  false,
			Duration: duration,
			Error:    err,
		}, err
	}
	defer conn.Close()

	// 尝试创建一个会话来验证连接完全可用
	session, err := conn.NewSession()
	if err != nil {
		return &PingResult{
			Host:     c.host,
			Success:  false,
			Duration: duration,
			Error:    fmt.Errorf("创建会话失败: %w", err),
		}, err
	}
	session.Close()

	return &PingResult{
		Host:     c.host,
		Success:  true,
		Duration: duration,
		Error:    nil,
	}, nil
}

// PingResult ping 测试结果
type PingResult struct {
	Host     string
	Success  bool
	Duration time.Duration
	Error    error
}

// Result 执行结果
type Result struct {
	Host     string
	Command  string
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Error    error
}

// loadPrivateKey 加载私钥文件
func loadPrivateKey(keyPath string) (ssh.Signer, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		// 尝试解析加密的私钥（需要密码）
		return nil, fmt.Errorf("解析私钥失败，可能需要密码: %w", err)
	}

	return signer, nil
}
