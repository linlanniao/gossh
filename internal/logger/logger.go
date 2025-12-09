package logger

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Logger 日志记录器
type Logger struct {
	logger *slog.Logger
	file   *os.File
}

// NewLogger 创建新的日志记录器
// 如果 logDir 为空，则不记录日志
// logDir 必须是目录路径，会自动生成文件名：命令名-时间戳.log
func NewLogger(logDir string, command string) (*Logger, error) {
	if logDir == "" {
		return &Logger{}, nil
	}

	// 确保目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	// 自动生成文件名：命令名-时间戳.log
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	fileName := command + "-" + timestamp + ".log"
	actualLogFile := filepath.Join(logDir, fileName)

	// 打开或创建日志文件（追加模式）
	file, err := os.OpenFile(actualLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	// 创建 JSON handler（结构化日志）
	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(handler)

	return &Logger{
		logger: logger,
		file:   file,
	}, nil
}

// Close 关闭日志文件
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// IsEnabled 检查日志是否启用
func (l *Logger) IsEnabled() bool {
	return l.logger != nil
}

// LogCommandStart 记录命令开始
func (l *Logger) LogCommandStart(command string, args map[string]interface{}) {
	if !l.IsEnabled() {
		return
	}

	// 构建 key-value 对
	kv := []any{
		"event", "command_start",
		"command", command,
		"timestamp", time.Now().Format(time.RFC3339),
	}

	for k, v := range args {
		kv = append(kv, k, v)
	}

	l.logger.Info("命令开始执行", kv...)
}

// LogCommandEnd 记录命令结束
func (l *Logger) LogCommandEnd(command string, duration time.Duration, success bool, err error) {
	if !l.IsEnabled() {
		return
	}

	kv := []any{
		"event", "command_end",
		"command", command,
		"timestamp", time.Now().Format(time.RFC3339),
		"duration", duration,
		"success", success,
	}

	if err != nil {
		kv = append(kv, "error", err.Error())
	}

	l.logger.Info("命令执行完成", kv...)
}

// LogHosts 记录主机列表
func (l *Logger) LogHosts(hosts []string) {
	if !l.IsEnabled() {
		return
	}

	l.logger.Info("主机列表",
		"event", "hosts_loaded",
		"count", len(hosts),
		"hosts", hosts,
	)
}

// LogHostResult 记录单个主机的执行结果
func (l *Logger) LogHostResult(host string, command string, exitCode int, duration time.Duration, success bool, stdout string, stderr string, err error) {
	if !l.IsEnabled() {
		return
	}

	kv := []any{
		"event", "host_result",
		"host", host,
		"command", command,
		"exit_code", exitCode,
		"duration", duration,
		"success", success,
	}

	if stdout != "" {
		kv = append(kv, "stdout", stdout)
	}

	if stderr != "" {
		kv = append(kv, "stderr", stderr)
	}

	if err != nil {
		kv = append(kv, "error", err.Error())
	}

	if success {
		l.logger.Info("主机执行结果", kv...)
	} else {
		l.logger.Error("主机执行结果", kv...)
	}
}

// LogInfo 记录一般信息
func (l *Logger) LogInfo(msg string, args ...any) {
	if !l.IsEnabled() {
		return
	}
	l.logger.Info(msg, args...)
}

// LogError 记录错误信息
func (l *Logger) LogError(msg string, err error, args ...any) {
	if !l.IsEnabled() {
		return
	}
	kv := append(args, "error", err.Error())
	l.logger.Error(msg, kv...)
}

