package executor

import (
	"fmt"
	"sync"
	"time"

	"gossh/internal/ssh"
)

// Executor 批量执行器
type Executor struct {
	hosts    []Host
	user     string
	keyPath  string
	password string
	port     string
}

// Host 主机信息
type Host struct {
	Address string
	Port    string
	User    string
	KeyPath string
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

// ProgressCallback 进度回调函数类型
// stage: 阶段信息，value: 进度值（0-100），isFailed: 是否失败
type ProgressCallback func(host string, stage string, value int64, isFailed bool)

// taskFunc 任务执行函数类型
type taskFunc func(client *ssh.Client, h Host) (*ssh.Result, error)

// ExecuteCommand 并发执行命令
func (e *Executor) ExecuteCommand(command string, concurrency int, progressCallback ProgressCallback) ([]*ssh.Result, error) {
	return e.ExecuteCommandWithBecome(command, concurrency, false, "", progressCallback)
}

// ExecuteCommandWithBecome 并发执行命令，支持 become 模式
func (e *Executor) ExecuteCommandWithBecome(command string, concurrency int, become bool, becomeUser string, progressCallback ProgressCallback) ([]*ssh.Result, error) {
	task := func(client *ssh.Client, h Host) (*ssh.Result, error) {
		return client.ExecuteWithBecome(command, become, becomeUser)
	}
	return e.executeConcurrent(task, command, concurrency, progressCallback)
}

// ExecuteScript 并发执行脚本（先上传到临时目录再执行）
func (e *Executor) ExecuteScript(scriptPath string, concurrency int, progressCallback ProgressCallback) ([]*ssh.Result, error) {
	return e.ExecuteScriptWithBecome(scriptPath, concurrency, false, "", progressCallback)
}

// ExecuteScriptWithBecome 并发执行脚本（先上传到临时目录再执行），支持 become 模式
func (e *Executor) ExecuteScriptWithBecome(scriptPath string, concurrency int, become bool, becomeUser string, progressCallback ProgressCallback) ([]*ssh.Result, error) {
	task := func(client *ssh.Client, h Host) (*ssh.Result, error) {
		return client.ExecuteScriptWithBecome(scriptPath, become, becomeUser)
	}
	return e.executeConcurrent(task, scriptPath, concurrency, progressCallback)
}

// UploadFile 并发上传文件
func (e *Executor) UploadFile(localPath string, remotePath string, mode string, concurrency int, progressCallback ProgressCallback) ([]*ssh.Result, error) {
	task := func(client *ssh.Client, h Host) (*ssh.Result, error) {
		return client.UploadFile(localPath, remotePath, mode)
	}
	command := fmt.Sprintf("upload %s -> %s", localPath, remotePath)
	return e.executeConcurrent(task, command, concurrency, progressCallback)
}

// executeConcurrent 公共的并发执行逻辑
func (e *Executor) executeConcurrent(task taskFunc, command string, concurrency int, progressCallback ProgressCallback) ([]*ssh.Result, error) {
	if concurrency <= 0 {
		concurrency = 5
	}

	results := make([]*ssh.Result, len(e.hosts))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
	var mu sync.Mutex

	for i, host := range e.hosts {
		wg.Add(1)
		go func(idx int, h Host) {
			hostAddr := h.Address
			startTime := time.Now()

			defer func() {
				if r := recover(); r != nil {
					duration := time.Since(startTime)
					mu.Lock()
					if results[idx] == nil {
						results[idx] = &ssh.Result{
							Host:     h.Address,
							Command:  command,
							Stdout:   "",
							Stderr:   fmt.Sprintf("panic: %v", r),
							ExitCode: -1,
							Duration: duration,
							Error:    fmt.Errorf("panic: %v", r),
						}
					}
					mu.Unlock()
					if progressCallback != nil {
						progressCallback(h.Address, fmt.Sprintf("panic: %v", r), 100, true)
					}
				}
				wg.Done()
			}()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

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

			client, err := ssh.NewClient(h.Address, port, user, keyPath, e.password)
			if err != nil {
				duration := time.Since(startTime)
				mu.Lock()
				results[idx] = &ssh.Result{
					Host:     h.Address,
					Command:  command,
					Stdout:   "",
					Stderr:   fmt.Sprintf("连接失败: %v", err),
					ExitCode: -1,
					Duration: duration,
					Error:    err,
				}
				mu.Unlock()
				if progressCallback != nil {
					progressCallback(hostAddr, fmt.Sprintf("连接失败: %v", err), 100, true)
				}
				return
			}

			result, err := task(client, h)
			if err != nil {
				duration := time.Since(startTime)
				mu.Lock()
				if results[idx] == nil {
					results[idx] = &ssh.Result{
						Host:     h.Address,
						Command:  command,
						Stdout:   "",
						Stderr:   fmt.Sprintf("执行失败: %v", err),
						ExitCode: -1,
						Duration: duration,
						Error:    err,
					}
				}
				mu.Unlock()
				if progressCallback != nil {
					progressCallback(hostAddr, fmt.Sprintf("执行失败: %v", err), 100, true)
				}
				return
			}

			mu.Lock()
			results[idx] = result
			mu.Unlock()

			if progressCallback != nil {
				if result.ExitCode == 0 {
					progressCallback(hostAddr, "完成", 100, false)
				} else {
					progressCallback(hostAddr, fmt.Sprintf("失败(退出码:%d)", result.ExitCode), 100, true)
				}
			}
		}(i, host)
	}

	wg.Wait()
	return results, nil
}
