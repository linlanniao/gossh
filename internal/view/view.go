package view

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"gossh/internal/executor"
	"gossh/internal/ssh"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// setupTableStyle 设置表格样式 - 使用简洁优雅的样式
func setupTableStyle(t table.Writer) {
	// 使用预定义的简洁样式，然后自定义颜色
	t.SetStyle(table.StyleDefault)
	t.Style().Options.SeparateRows = false
	t.Style().Options.SeparateColumns = true
	t.Style().Color.Header = []text.Color{text.FgHiWhite}
	t.Style().Color.Row = []text.Color{text.FgWhite}
	t.Style().Color.RowAlternate = []text.Color{text.FgHiBlack}
}

// PrintRunResults 打印 run 命令的执行结果
func PrintRunResults(results []*ssh.Result, totalDuration time.Duration, showOutput bool) {
	successCount := 0
	failCount := 0
	var successHosts []string
	var failHosts []string

	// 创建表格
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	setupTableStyle(t)
	t.AppendHeader(table.Row{"主机", "状态", "退出码", "耗时", "错误信息"})

	for _, result := range results {
		var status string
		var exitCode string
		var duration string
		var errorMsg string

		if result.Error == nil && result.ExitCode == 0 {
			successCount++
			successHosts = append(successHosts, result.Host)
			status = text.Colors{text.FgGreen}.Sprint("✓ 成功")
		} else {
			failCount++
			failHosts = append(failHosts, result.Host)
			status = text.Colors{text.FgRed}.Sprint("✗ 失败")
			if result.ExitCode != 0 {
				exitCode = fmt.Sprintf("%d", result.ExitCode)
			}
			if result.Error != nil {
				// 截断过长的错误信息
				errorMsg = result.Error.Error()
				if len(errorMsg) > 50 {
					errorMsg = errorMsg[:47] + "..."
				}
			}
		}

		// 格式化耗时显示
		if result.Duration > 0 {
			duration = result.Duration.Round(time.Millisecond).String()
		}

		t.AppendRow(table.Row{result.Host, status, exitCode, duration, errorMsg})
	}

	fmt.Println()
	t.Render()

	// 如果需要显示详细输出，在表格后单独显示
	if showOutput {
		fmt.Println("\n" + text.Colors{text.FgHiCyan}.Sprint(strings.Repeat("=", 80)))
		fmt.Println(text.Colors{text.FgHiCyan, text.Bold}.Sprint("详细输出"))
		fmt.Println(text.Colors{text.FgHiCyan}.Sprint(strings.Repeat("=", 80)))
		for _, result := range results {
			// 判断是否成功
			isSuccess := result.Error == nil && result.ExitCode == 0
			
			// 主机名颜色：成功用绿色，失败用红色
			var hostColor text.Colors
			if isSuccess {
				hostColor = text.Colors{text.FgGreen, text.Bold}
			} else {
				hostColor = text.Colors{text.FgRed, text.Bold}
			}
			fmt.Printf("\n%s\n", hostColor.Sprint("["+result.Host+"]"))
			
			// 标准输出（成功时显示）
			if result.Stdout != "" {
				fmt.Printf("%s\n%s\n", 
					text.Colors{text.FgHiWhite}.Sprint("标准输出:"),
					result.Stdout)
			}
			
			// 标准错误（失败时显示，用红色）
			if result.Stderr != "" {
				fmt.Printf("%s\n%s\n", 
					text.Colors{text.FgRed}.Sprint("标准错误:"),
					text.Colors{text.FgRed}.Sprint(result.Stderr))
			}
			
			// 错误信息（失败时显示，用红色）
			if result.Error != nil {
				fmt.Printf("%s %s\n", 
					text.Colors{text.FgRed}.Sprint("错误:"),
					text.Colors{text.FgRed}.Sprint(result.Error.Error()))
			}
			
			// 分隔线用灰色
			fmt.Println(text.Colors{text.FgHiBlack}.Sprint(strings.Repeat("-", 80)))
		}
	}

	// 打印汇总信息
	fmt.Printf("\n总计: %d 台主机 | %s | %s | 总耗时: %s\n", 
		len(results),
		text.Colors{text.FgGreen}.Sprint(fmt.Sprintf("成功: %d", successCount)),
		text.Colors{text.FgRed}.Sprint(fmt.Sprintf("失败: %d", failCount)),
		totalDuration.Round(time.Millisecond).String())
	
	// 打印成功的主机列表
	if len(successHosts) > 0 {
		fmt.Printf("%s: %s\n",
			text.Colors{text.FgGreen, text.Bold}.Sprint("成功主机"),
			text.Colors{text.FgGreen}.Sprint(strings.Join(successHosts, ", ")))
	}
	
	// 打印失败的主机列表
	if len(failHosts) > 0 {
		fmt.Printf("%s: %s\n",
			text.Colors{text.FgRed, text.Bold}.Sprint("失败主机"),
			text.Colors{text.FgRed}.Sprint(strings.Join(failHosts, ", ")))
	}
	
	fmt.Println()
}

// PrintPingResults 打印 ping 命令的测试结果
func PrintPingResults(results []*ssh.PingResult, totalDuration time.Duration) {
	successCount := 0
	failCount := 0

	// 创建表格
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	setupTableStyle(t)
	t.AppendHeader(table.Row{"主机", "状态", "延迟", "错误信息"})

	for _, result := range results {
		var status string
		var duration string
		var errorMsg string

		if result.Success {
			successCount++
			status = text.Colors{text.FgGreen}.Sprint("✓ 成功")
			duration = result.Duration.Round(time.Millisecond).String()
		} else {
			failCount++
			status = text.Colors{text.FgRed}.Sprint("✗ 失败")
			if result.Duration > 0 {
				duration = result.Duration.Round(time.Millisecond).String()
			}
			if result.Error != nil {
				errorMsg = result.Error.Error()
			}
		}

		t.AppendRow(table.Row{result.Host, status, duration, errorMsg})
	}

	fmt.Println()
	t.Render()

	// 打印汇总信息
	fmt.Printf("\n总计: %d 台主机 | %s | %s | 总耗时: %s\n\n", 
		len(results),
		text.Colors{text.FgGreen}.Sprint(fmt.Sprintf("成功: %d", successCount)),
		text.Colors{text.FgRed}.Sprint(fmt.Sprintf("失败: %d", failCount)),
		totalDuration.Round(time.Millisecond).String())
}

// PrintListResults 打印 list 命令的主机列表
// format: ip（仅IP地址）、full（完整信息）、json（JSON格式）
func PrintListResults(hosts []executor.Host, format string) {
	switch format {
	case "json":
		printListJSON(hosts)
	case "full":
		printListFull(hosts)
	default: // "ip"
		printListIP(hosts)
	}
}

// printListIP 仅打印 IP 地址
func printListIP(hosts []executor.Host) {
	for _, host := range hosts {
		fmt.Println(host.Address)
	}
}

// printListFull 打印完整信息
func printListFull(hosts []executor.Host) {
	// 创建表格
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	setupTableStyle(t)
	t.AppendHeader(table.Row{"IP地址", "端口", "用户", "SSH Key"})

	for _, host := range hosts {
		port := host.Port
		if port == "" {
			port = "22"
		}
		user := host.User
		if user == "" {
			user = "-"
		}
		keyPath := host.KeyPath
		if keyPath == "" {
			keyPath = "-"
		}
		t.AppendRow(table.Row{host.Address, port, user, keyPath})
	}

	fmt.Println()
	t.Render()
	fmt.Printf("\n总计: %d 台主机\n\n", len(hosts))
}

// printListJSON 打印 JSON 格式
func printListJSON(hosts []executor.Host) {
	type HostInfo struct {
		Address string `json:"address"`
		Port    string `json:"port"`
		User    string `json:"user,omitempty"`
		KeyPath string `json:"key_path,omitempty"`
	}

	hostInfos := make([]HostInfo, len(hosts))
	for i, host := range hosts {
		port := host.Port
		if port == "" {
			port = "22"
		}
		hostInfos[i] = HostInfo{
			Address: host.Address,
			Port:    port,
		}
		if host.User != "" {
			hostInfos[i].User = host.User
		}
		if host.KeyPath != "" {
			hostInfos[i].KeyPath = host.KeyPath
		}
	}

	jsonData, err := json.MarshalIndent(hostInfos, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON 序列化失败: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}

// ProgressTracker 进度跟踪器 - 只显示一个总体进度条
type ProgressTracker struct {
	pw          progress.Writer
	tracker     *progress.Tracker
	total       int64
	completed   int64
	failedHosts []string // 记录失败的主机
	mu          sync.Mutex
}

// NewProgressTracker 创建新的进度跟踪器
func NewProgressTracker(total int, title string) *ProgressTracker {
	pw := progress.NewWriter()
	pw.SetAutoStop(true)
	pw.SetTrackerLength(50)
	pw.SetMessageWidth(40)
	pw.SetNumTrackersExpected(1) // 只有一个进度条
	pw.SetStyle(progress.StyleCircle)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 100)

	// 使用更柔和的颜色方案
	pw.Style().Colors.Message = text.Colors{text.FgHiWhite}
	pw.Style().Colors.Stats = text.Colors{text.FgHiWhite}
	pw.Style().Colors.Tracker = text.Colors{text.FgCyan}
	pw.Style().Options.PercentFormat = "%4.1f%%"

	// 创建唯一的总体进度条
	tracker := &progress.Tracker{
		Message: fmt.Sprintf("%-40s", title),
		Total:   int64(total),
		Units:   progress.UnitsDefault,
	}
	pw.AppendTracker(tracker)

	progressTracker := &ProgressTracker{
		pw:          pw,
		tracker:     tracker,
		total:       int64(total),
		completed:   0,
		failedHosts: make([]string, 0),
	}

	go pw.Render()

	return progressTracker
}

// Increment 增加进度（完成一个任务）
func (pt *ProgressTracker) Increment() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.completed++
	if pt.tracker != nil {
		pt.tracker.SetValue(pt.completed)
	}
}

// AddFailedHost 添加失败的主机（立即打印红色错误信息）
func (pt *ProgressTracker) AddFailedHost(host string, reason string) {
	pt.mu.Lock()
	pt.failedHosts = append(pt.failedHosts, host)
	pt.mu.Unlock()

	// 立即打印失败信息（红色）
	fmt.Fprintf(os.Stderr, "%s %s: %s\n",
		text.Colors{text.FgRed}.Sprint("✗"),
		text.Colors{text.FgRed}.Sprint(host),
		text.Colors{text.FgRed}.Sprint(reason))
}

// GetFailedHosts 获取失败的主机列表
func (pt *ProgressTracker) GetFailedHosts() []string {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return pt.failedHosts
}

// Stop 停止进度跟踪器
func (pt *ProgressTracker) Stop() {
	pt.pw.Stop()
}

// PrintPingConfig 打印 ping 命令的配置参数
func PrintPingConfig(hostsFile, hostsDir, hostsString, group, user, keyPath, password, port string, concurrency int) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	setupTableStyle(t)
	t.SetTitle(text.Colors{text.FgHiCyan, text.Bold}.Sprint("当前配置参数"))
	t.AppendHeader(table.Row{"参数", "值"})

	// 主机列表来源
	var hostsSource string
	if hostsDir != "" {
		hostsSource = fmt.Sprintf("目录 (%s)", hostsDir)
	} else if hostsFile != "" {
		hostsSource = fmt.Sprintf("文件 (%s)", hostsFile)
	} else if hostsString != "" {
		hostsSource = fmt.Sprintf("命令行参数 (%s)", hostsString)
	} else {
		hostsSource = text.Colors{text.FgHiYellow}.Sprint("ansible.cfg inventory")
	}
	t.AppendRow(table.Row{"主机列表来源", hostsSource})

	if group != "" {
		t.AppendRow(table.Row{"分组", text.Colors{text.FgCyan}.Sprint(group)})
	}

	// 认证信息
	t.AppendRow(table.Row{"用户名", getValueOrDefault(user, text.Colors{text.FgHiBlack}.Sprint("(未设置)"))})
	t.AppendRow(table.Row{"SSH 密钥", getValueOrDefault(keyPath, text.Colors{text.FgHiBlack}.Sprint("(未设置)"))})
	if password != "" {
		t.AppendRow(table.Row{"密码", text.Colors{text.FgGreen}.Sprint("***已设置***")})
	} else {
		t.AppendRow(table.Row{"密码", text.Colors{text.FgHiBlack}.Sprint("(未设置)")})
	}
	t.AppendRow(table.Row{"SSH 端口", getValueOrDefault(port, "22")})
	t.AppendRow(table.Row{"并发数", text.Colors{text.FgCyan}.Sprint(fmt.Sprintf("%d", concurrency))})

	fmt.Println()
	t.Render()
	fmt.Println()
}

// PrintRunConfig 打印 run 命令的配置参数
func PrintRunConfig(hostsFile, hostsDir, hostsString, group, user, keyPath, password, port, command string, become bool, becomeUser string, concurrency int, showOutput bool) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	setupTableStyle(t)
	t.SetTitle(text.Colors{text.FgHiCyan, text.Bold}.Sprint("当前配置参数"))
	t.AppendHeader(table.Row{"参数", "值"})
	
	// 设置列配置：第一列（参数）宽度固定，第二列（值）自动换行，最大宽度80
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMax: 15}, // 参数列最大宽度15
		{Number: 2, WidthMax: 80}, // 值列最大宽度80，超出自动换行
	})

	// 辅助函数：包装长文本（处理包含 ANSI 颜色代码的文本）
	wrapText := func(s string, maxWidth int) string {
		// 使用 text.WrapHard 进行硬换行，保留 ANSI 颜色代码
		return text.WrapHard(s, maxWidth)
	}

	// 主机列表来源
	var hostsSource string
	if hostsDir != "" {
		hostsSource = fmt.Sprintf("目录 (%s)", hostsDir)
	} else if hostsFile != "" {
		hostsSource = fmt.Sprintf("文件 (%s)", hostsFile)
	} else if hostsString != "" {
		hostsSource = fmt.Sprintf("命令行参数 (%s)", hostsString)
	} else {
		hostsSource = text.Colors{text.FgHiYellow}.Sprint("ansible.cfg inventory")
	}
	t.AppendRow(table.Row{"主机列表来源", wrapText(hostsSource, 80)})

	if group != "" {
		t.AppendRow(table.Row{"分组", text.Colors{text.FgCyan}.Sprint(group)})
	}

	// 认证信息
	t.AppendRow(table.Row{"用户名", getValueOrDefault(user, text.Colors{text.FgHiBlack}.Sprint("(未设置)"))})
	keyPathValue := getValueOrDefault(keyPath, text.Colors{text.FgHiBlack}.Sprint("(未设置)"))
	t.AppendRow(table.Row{"SSH 密钥", wrapText(keyPathValue, 80)})
	if password != "" {
		t.AppendRow(table.Row{"密码", text.Colors{text.FgGreen}.Sprint("***已设置***")})
	} else {
		t.AppendRow(table.Row{"密码", text.Colors{text.FgHiBlack}.Sprint("(未设置)")})
	}
	t.AppendRow(table.Row{"SSH 端口", getValueOrDefault(port, "22")})
	t.AppendRow(table.Row{"并发数", text.Colors{text.FgCyan}.Sprint(fmt.Sprintf("%d", concurrency))})

	// 执行内容 - 长命令需要换行
	if command != "" {
		commandText := text.Colors{text.FgYellow}.Sprint(command)
		t.AppendRow(table.Row{"执行命令", wrapText(commandText, 80)})
	}

	// Become 模式
	if become {
		becomeUserText := becomeUser
		if becomeUserText == "" {
			becomeUserText = "root"
		}
		t.AppendRow(table.Row{"Become 模式", text.Colors{text.FgGreen}.Sprint("是")})
		t.AppendRow(table.Row{"Become 用户", text.Colors{text.FgCyan}.Sprint(becomeUserText)})
	} else {
		t.AppendRow(table.Row{"Become 模式", text.Colors{text.FgHiBlack}.Sprint("否")})
	}

	var outputStatus string
	if showOutput {
		outputStatus = text.Colors{text.FgGreen}.Sprint("是")
	} else {
		outputStatus = text.Colors{text.FgHiBlack}.Sprint("否")
	}
	t.AppendRow(table.Row{"显示输出", outputStatus})

	fmt.Println()
	t.Render()
	fmt.Println()
}

// PrintScriptConfig 打印 script 命令的配置参数
func PrintScriptConfig(hostsFile, hostsDir, hostsString, group, user, keyPath, password, port, scriptPath string, become bool, becomeUser string, concurrency int, showOutput bool) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	setupTableStyle(t)
	t.SetTitle(text.Colors{text.FgHiCyan, text.Bold}.Sprint("当前配置参数"))
	t.AppendHeader(table.Row{"参数", "值"})
	
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMax: 15},
		{Number: 2, WidthMax: 80},
	})

	wrapText := func(s string, maxWidth int) string {
		return text.WrapHard(s, maxWidth)
	}

	var hostsSource string
	if hostsDir != "" {
		hostsSource = fmt.Sprintf("目录 (%s)", hostsDir)
	} else if hostsFile != "" {
		hostsSource = fmt.Sprintf("文件 (%s)", hostsFile)
	} else if hostsString != "" {
		hostsSource = fmt.Sprintf("命令行参数 (%s)", hostsString)
	} else {
		hostsSource = text.Colors{text.FgHiYellow}.Sprint("ansible.cfg inventory")
	}
	t.AppendRow(table.Row{"主机列表来源", wrapText(hostsSource, 80)})

	if group != "" {
		t.AppendRow(table.Row{"分组", text.Colors{text.FgCyan}.Sprint(group)})
	}

	t.AppendRow(table.Row{"用户名", getValueOrDefault(user, text.Colors{text.FgHiBlack}.Sprint("(未设置)"))})
	keyPathValue := getValueOrDefault(keyPath, text.Colors{text.FgHiBlack}.Sprint("(未设置)"))
	t.AppendRow(table.Row{"SSH 密钥", wrapText(keyPathValue, 80)})
	if password != "" {
		t.AppendRow(table.Row{"密码", text.Colors{text.FgGreen}.Sprint("***已设置***")})
	} else {
		t.AppendRow(table.Row{"密码", text.Colors{text.FgHiBlack}.Sprint("(未设置)")})
	}
	t.AppendRow(table.Row{"SSH 端口", getValueOrDefault(port, "22")})
	t.AppendRow(table.Row{"并发数", text.Colors{text.FgCyan}.Sprint(fmt.Sprintf("%d", concurrency))})

	if scriptPath != "" {
		scriptText := text.Colors{text.FgYellow}.Sprint(scriptPath)
		t.AppendRow(table.Row{"执行脚本", wrapText(scriptText, 80)})
	}

	if become {
		becomeUserText := becomeUser
		if becomeUserText == "" {
			becomeUserText = "root"
		}
		t.AppendRow(table.Row{"Become 模式", text.Colors{text.FgGreen}.Sprint("是")})
		t.AppendRow(table.Row{"Become 用户", text.Colors{text.FgCyan}.Sprint(becomeUserText)})
	} else {
		t.AppendRow(table.Row{"Become 模式", text.Colors{text.FgHiBlack}.Sprint("否")})
	}

	var outputStatus string
	if showOutput {
		outputStatus = text.Colors{text.FgGreen}.Sprint("是")
	} else {
		outputStatus = text.Colors{text.FgHiBlack}.Sprint("否")
	}
	t.AppendRow(table.Row{"显示输出", outputStatus})

	fmt.Println()
	t.Render()
	fmt.Println()
}

// PrintUploadConfig 打印 upload 命令的配置参数
func PrintUploadConfig(hostsFile, hostsDir, hostsString, group, user, keyPath, password, port, localPath, remotePath, mode string, concurrency int, showOutput bool) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	setupTableStyle(t)
	t.SetTitle(text.Colors{text.FgHiCyan, text.Bold}.Sprint("当前配置参数"))
	t.AppendHeader(table.Row{"参数", "值"})
	
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMax: 15},
		{Number: 2, WidthMax: 80},
	})

	wrapText := func(s string, maxWidth int) string {
		return text.WrapHard(s, maxWidth)
	}

	var hostsSource string
	if hostsDir != "" {
		hostsSource = fmt.Sprintf("目录 (%s)", hostsDir)
	} else if hostsFile != "" {
		hostsSource = fmt.Sprintf("文件 (%s)", hostsFile)
	} else if hostsString != "" {
		hostsSource = fmt.Sprintf("命令行参数 (%s)", hostsString)
	} else {
		hostsSource = text.Colors{text.FgHiYellow}.Sprint("ansible.cfg inventory")
	}
	t.AppendRow(table.Row{"主机列表来源", wrapText(hostsSource, 80)})

	if group != "" {
		t.AppendRow(table.Row{"分组", text.Colors{text.FgCyan}.Sprint(group)})
	}

	t.AppendRow(table.Row{"用户名", getValueOrDefault(user, text.Colors{text.FgHiBlack}.Sprint("(未设置)"))})
	keyPathValue := getValueOrDefault(keyPath, text.Colors{text.FgHiBlack}.Sprint("(未设置)"))
	t.AppendRow(table.Row{"SSH 密钥", wrapText(keyPathValue, 80)})
	if password != "" {
		t.AppendRow(table.Row{"密码", text.Colors{text.FgGreen}.Sprint("***已设置***")})
	} else {
		t.AppendRow(table.Row{"密码", text.Colors{text.FgHiBlack}.Sprint("(未设置)")})
	}
	t.AppendRow(table.Row{"SSH 端口", getValueOrDefault(port, "22")})
	t.AppendRow(table.Row{"并发数", text.Colors{text.FgCyan}.Sprint(fmt.Sprintf("%d", concurrency))})

	if localPath != "" {
		localText := text.Colors{text.FgYellow}.Sprint(localPath)
		t.AppendRow(table.Row{"本地路径", wrapText(localText, 80)})
	}
	if remotePath != "" {
		remoteText := text.Colors{text.FgYellow}.Sprint(remotePath)
		t.AppendRow(table.Row{"远程路径", wrapText(remoteText, 80)})
	}
	t.AppendRow(table.Row{"文件权限", text.Colors{text.FgCyan}.Sprint(getValueOrDefault(mode, "0644"))})

	var outputStatus string
	if showOutput {
		outputStatus = text.Colors{text.FgGreen}.Sprint("是")
	} else {
		outputStatus = text.Colors{text.FgHiBlack}.Sprint("否")
	}
	t.AppendRow(table.Row{"显示输出", outputStatus})

	fmt.Println()
	t.Render()
	fmt.Println()
}

// PrintListConfig 打印 list 命令的配置参数
func PrintListConfig(hostsFile, hostsDir, hostsString, group, format string) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	setupTableStyle(t)
	t.SetTitle(text.Colors{text.FgHiCyan, text.Bold}.Sprint("当前配置参数"))
	t.AppendHeader(table.Row{"参数", "值"})

	// 主机列表来源
	var hostsSource string
	if hostsDir != "" {
		hostsSource = fmt.Sprintf("目录 (%s)", hostsDir)
	} else if hostsFile != "" {
		hostsSource = fmt.Sprintf("文件 (%s)", hostsFile)
	} else if hostsString != "" {
		hostsSource = fmt.Sprintf("命令行参数 (%s)", hostsString)
	} else {
		hostsSource = text.Colors{text.FgHiYellow}.Sprint("ansible.cfg inventory")
	}
	t.AppendRow(table.Row{"主机列表来源", hostsSource})

	if group != "" {
		t.AppendRow(table.Row{"分组", text.Colors{text.FgCyan}.Sprint(group)})
	}

	formatValue := getValueOrDefault(format, "ip")
	t.AppendRow(table.Row{"输出格式", text.Colors{text.FgCyan}.Sprint(formatValue)})

	fmt.Println()
	t.Render()
	fmt.Println()
}

// getValueOrDefault 获取值或默认值
func getValueOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
