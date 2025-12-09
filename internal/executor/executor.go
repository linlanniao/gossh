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

// ExecuteCommand 并发执行命令
func (e *Executor) ExecuteCommand(command string, concurrency int, progressCallback ProgressCallback) ([]*ssh.Result, error) {
	if concurrency <= 0 {
		concurrency = 5 // 默认并发数
	}

	results := make([]*ssh.Result, len(e.hosts))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
	var mu sync.Mutex

	for i, host := range e.hosts {
		wg.Add(1)
		go func(idx int, h Host) {
			defer wg.Done()

			hostAddr := h.Address
			startTime := time.Now()

			// 限制并发数
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 使用主机特定的配置，如果没有则使用默认配置
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

			result, err := client.Execute(command)
			if err != nil {
				duration := time.Since(startTime)
				mu.Lock()
				results[idx] = &ssh.Result{
					Host:     h.Address,
					Command:  command,
					Stdout:   "",
					Stderr:   fmt.Sprintf("执行失败: %v", err),
					ExitCode: -1,
					Duration: duration,
					Error:    err,
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

// ExecuteScript 并发执行脚本
func (e *Executor) ExecuteScript(scriptPath string, concurrency int, progressCallback ProgressCallback) ([]*ssh.Result, error) {
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
			defer wg.Done()

			hostAddr := h.Address
			startTime := time.Now()

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
					Command:  scriptPath,
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

			result, err := client.ExecuteScript(scriptPath)
			if err != nil {
				duration := time.Since(startTime)
				mu.Lock()
				results[idx] = &ssh.Result{
					Host:     h.Address,
					Command:  scriptPath,
					Stdout:   "",
					Stderr:   fmt.Sprintf("执行失败: %v", err),
					ExitCode: -1,
					Duration: duration,
					Error:    err,
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
