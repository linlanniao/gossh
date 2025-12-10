package executor

import (
	"fmt"
	"sync"
	"time"

	"gossh/internal/ssh"
)

// Executor 批量执行器
// 用于并发执行 SSH 命令、脚本上传和执行、文件上传等操作
type Executor struct {
	hosts    []Host
	user     string
	keyPath  string
	password string
	port     string
}

// Host 主机信息
// 包含主机的地址、端口、用户和 SSH 密钥路径
type Host struct {
	Address string   // 主机地址（IP 或域名）
	Port    string   // SSH 端口
	User    string   // SSH 用户名
	KeyPath string   // SSH 私钥路径
	Groups  []string // 主机所属的分组列表（一个主机可能属于多个分组）
}

// NewExecutor 创建新的执行器
func NewExecutor(hosts []Host, user, keyPath, password, defaultPort string) *Executor {
	// 如果没有指定端口，使用默认端口
	for i := range hosts {
		if hosts[i].Port == "" {
			hosts[i].Port = defaultPort
		}
		if hosts[i].User == "" {
			hosts[i].User = user
		}
		if hosts[i].KeyPath == "" {
			hosts[i].KeyPath = keyPath
		}
	}

	return &Executor{
		hosts:    hosts,
		user:     user,
		keyPath:  keyPath,
		password: password,
		port:     defaultPort,
	}
}

// ProgressTracker 进度跟踪器接口
// 用于统一管理多个主机的进度显示
type ProgressTracker interface {
	AddTracker(host string) interface{}
	UpdateTracker(host string, value int64, message string)
	MarkTrackerDone(host string)
	MarkTrackerErrored(host string, reason string)
}

// taskFunc 任务执行函数类型
// 定义单个主机任务的执行逻辑
type taskFunc func(client *ssh.Client, h Host) (*ssh.Result, error)

// ExecuteCommand 并发执行命令
func (e *Executor) ExecuteCommand(command string, concurrency int, progressTracker ProgressTracker) ([]*ssh.Result, error) {
	return e.ExecuteCommandWithBecome(command, concurrency, false, "", progressTracker)
}

// ExecuteCommandWithBecome 并发执行命令，支持 become 模式
func (e *Executor) ExecuteCommandWithBecome(command string, concurrency int, become bool, becomeUser string, progressTracker ProgressTracker) ([]*ssh.Result, error) {
	task := func(client *ssh.Client, h Host) (*ssh.Result, error) {
		return client.ExecuteWithBecome(command, become, becomeUser)
	}
	return e.executeConcurrent(task, command, concurrency, progressTracker)
}

// ExecuteScript 并发执行脚本（先上传到临时目录再执行）
func (e *Executor) ExecuteScript(scriptPath string, concurrency int, progressTracker ProgressTracker) ([]*ssh.Result, error) {
	return e.ExecuteScriptWithBecome(scriptPath, concurrency, false, "", progressTracker)
}

// ExecuteScriptWithBecome 并发执行脚本（先上传到临时目录再执行），支持 become 模式
func (e *Executor) ExecuteScriptWithBecome(scriptPath string, concurrency int, become bool, becomeUser string, progressTracker ProgressTracker) ([]*ssh.Result, error) {
	task := func(client *ssh.Client, h Host) (*ssh.Result, error) {
		return client.ExecuteScriptWithBecome(scriptPath, become, becomeUser)
	}
	return e.executeConcurrent(task, scriptPath, concurrency, progressTracker)
}

// UploadFile 并发上传文件
func (e *Executor) UploadFile(localPath string, remotePath string, mode string, concurrency int, progressTracker ProgressTracker) ([]*ssh.Result, error) {
	task := func(client *ssh.Client, h Host) (*ssh.Result, error) {
		return client.UploadFile(localPath, remotePath, mode)
	}
	command := fmt.Sprintf("upload %s -> %s", localPath, remotePath)
	return e.executeConcurrent(task, command, concurrency, progressTracker)
}

// executeConcurrent 公共的并发执行逻辑
// 使用信号量控制并发数量，支持进度跟踪和错误处理
func (e *Executor) executeConcurrent(task taskFunc, command string, concurrency int, progressTracker ProgressTracker) ([]*ssh.Result, error) {
	concurrency = normalizeConcurrency(concurrency)
	results := make([]*ssh.Result, len(e.hosts))
	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, host := range e.hosts {
		wg.Add(1)
		go e.executeHostTask(i, host, task, command, results, semaphore, &mu, &wg, progressTracker)
	}

	wg.Wait()
	return results, nil
}

// normalizeConcurrency 规范化并发数
func normalizeConcurrency(concurrency int) int {
	if concurrency <= 0 {
		return 5
	}
	return concurrency
}

// executeHostTask 执行单个主机的任务
func (e *Executor) executeHostTask(
	idx int,
	h Host,
	task taskFunc,
	command string,
	results []*ssh.Result,
	semaphore chan struct{},
	mu *sync.Mutex,
	wg *sync.WaitGroup,
	progressTracker ProgressTracker,
) {
	startTime := time.Now()
	defer e.handleTaskPanic(idx, h, command, startTime, results, mu, wg, progressTracker)
	defer wg.Done()

	hostAddr := h.Address

	// 为主机创建 tracker
	if progressTracker != nil {
		progressTracker.AddTracker(hostAddr)
		progressTracker.UpdateTracker(hostAddr, 10, fmt.Sprintf("%s (连接中...)", hostAddr))
	}

	// 获取信号量，控制并发数
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	if progressTracker != nil {
		progressTracker.UpdateTracker(hostAddr, 30, fmt.Sprintf("%s (创建客户端...)", hostAddr))
	}
	client, err := e.createSSHClient(h)
	if err != nil {
		e.handleConnectionError(idx, h, command, startTime, err, results, mu, progressTracker)
		return
	}

	if progressTracker != nil {
		progressTracker.UpdateTracker(hostAddr, 60, fmt.Sprintf("%s (执行中...)", hostAddr))
	}
	result, err := task(client, h)
	if err != nil {
		e.handleTaskError(idx, h, command, startTime, err, results, mu, progressTracker)
		return
	}

	e.handleTaskSuccess(idx, result, results, mu, progressTracker)
}

// createSSHClient 创建 SSH 客户端
func (e *Executor) createSSHClient(h Host) (*ssh.Client, error) {
	keyPath := h.KeyPath
	if keyPath == "" {
		keyPath = e.keyPath
	}

	user := h.User
	if user == "" {
		user = e.user
	}

	port := h.Port
	if port == "" {
		port = e.port
	}

	return ssh.NewClient(h.Address, port, user, keyPath, e.password)
}

// handleTaskPanic 处理任务 panic
func (e *Executor) handleTaskPanic(
	idx int,
	h Host,
	command string,
	startTime time.Time,
	results []*ssh.Result,
	mu *sync.Mutex,
	wg *sync.WaitGroup,
	progressTracker ProgressTracker,
) {
	if r := recover(); r != nil {
		duration := time.Since(startTime)
		err := fmt.Errorf("panic: %v", r)

		mu.Lock()
		if results[idx] == nil {
			results[idx] = e.createErrorResult(h.Address, command, duration, err, "panic")
		}
		mu.Unlock()

		if progressTracker != nil {
			progressTracker.MarkTrackerErrored(h.Address, fmt.Sprintf("panic: %v", r))
		}
	}
}

// handleConnectionError 处理连接错误
func (e *Executor) handleConnectionError(
	idx int,
	h Host,
	command string,
	startTime time.Time,
	err error,
	results []*ssh.Result,
	mu *sync.Mutex,
	progressTracker ProgressTracker,
) {
	duration := time.Since(startTime)
	mu.Lock()
	results[idx] = e.createErrorResult(h.Address, command, duration, err, "连接失败")
	mu.Unlock()

	if progressTracker != nil {
		progressTracker.MarkTrackerErrored(h.Address, fmt.Sprintf("连接失败: %v", err))
	}
}

// handleTaskError 处理任务执行错误
func (e *Executor) handleTaskError(
	idx int,
	h Host,
	command string,
	startTime time.Time,
	err error,
	results []*ssh.Result,
	mu *sync.Mutex,
	progressTracker ProgressTracker,
) {
	duration := time.Since(startTime)
	mu.Lock()
	if results[idx] == nil {
		results[idx] = e.createErrorResult(h.Address, command, duration, err, "执行失败")
	}
	mu.Unlock()

	if progressTracker != nil {
		progressTracker.MarkTrackerErrored(h.Address, fmt.Sprintf("执行失败: %v", err))
	}
}

// handleTaskSuccess 处理任务成功
func (e *Executor) handleTaskSuccess(
	idx int,
	result *ssh.Result,
	results []*ssh.Result,
	mu *sync.Mutex,
	progressTracker ProgressTracker,
) {
	mu.Lock()
	results[idx] = result
	mu.Unlock()

	if progressTracker != nil {
		if result.ExitCode == 0 {
			progressTracker.UpdateTracker(result.Host, 100, result.Host)
			progressTracker.MarkTrackerDone(result.Host)
		} else {
			progressTracker.MarkTrackerErrored(result.Host, fmt.Sprintf("失败(退出码:%d)", result.ExitCode))
		}
	}
}

// createErrorResult 创建错误结果对象
func (e *Executor) createErrorResult(host, command string, duration time.Duration, err error, message string) *ssh.Result {
	return &ssh.Result{
		Host:     host,
		Command:  command,
		Stdout:   "",
		Stderr:   fmt.Sprintf("%s: %v", message, err),
		ExitCode: -1,
		Duration: duration,
		Error:    err,
	}
}
