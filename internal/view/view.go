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

// setupTableStyle 设置表格样式
// 配置表格的显示样式，包括分隔符、颜色等
func setupTableStyle(t table.Writer) {
	t.SetStyle(table.StyleDefault)
	t.Style().Options.SeparateRows = false
	t.Style().Options.SeparateColumns = true
	t.Style().Color.Header = []text.Color{text.FgHiWhite}
	t.Style().Color.Row = []text.Color{text.FgWhite}
	t.Style().Color.RowAlternate = []text.Color{text.FgHiBlack}
}

// PrintRunResults 打印 run 命令的执行结果
func PrintRunResults(results []*ssh.Result, totalDuration time.Duration, showOutput bool) {
	stats := collectRunStatistics(results)
	
	printRunResultsTable(results, stats)
	
	if showOutput {
		printRunDetailedOutput(results)
	}
	
	printRunSummary(results, stats, totalDuration)
}

// runStatistics 执行结果统计信息
type runStatistics struct {
	successCount int
	failCount    int
	successHosts []string
	failHosts    []string
}

// collectRunStatistics 收集执行结果统计信息
func collectRunStatistics(results []*ssh.Result) *runStatistics {
	stats := &runStatistics{
		successHosts: make([]string, 0),
		failHosts:    make([]string, 0),
	}

	for _, result := range results {
		if result.Error == nil && result.ExitCode == 0 {
			stats.successCount++
			stats.successHosts = append(stats.successHosts, result.Host)
		} else {
			stats.failCount++
			stats.failHosts = append(stats.failHosts, result.Host)
		}
	}

	return stats
}

// printRunResultsTable 打印执行结果表格
func printRunResultsTable(results []*ssh.Result, stats *runStatistics) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	setupTableStyle(t)
	t.AppendHeader(table.Row{"主机", "状态", "退出码", "耗时", "错误信息"})

	for _, result := range results {
		row := buildResultTableRow(result)
		t.AppendRow(row)
	}

	fmt.Println()
	t.Render()
}

// buildResultTableRow 构建结果表格行
func buildResultTableRow(result *ssh.Result) table.Row {
	var status string
	var exitCode string
	var duration string
	var errorMsg string

	if result.Error == nil && result.ExitCode == 0 {
		status = text.Colors{text.FgGreen}.Sprint("✓ 成功")
	} else {
		status = text.Colors{text.FgRed}.Sprint("✗ 失败")
		if result.ExitCode != 0 {
			exitCode = fmt.Sprintf("%d", result.ExitCode)
		}
		if result.Error != nil {
			errorMsg = truncateError(result.Error.Error(), 50)
		}
	}

	if result.Duration > 0 {
		duration = result.Duration.Round(time.Millisecond).String()
	}

	return table.Row{result.Host, status, exitCode, duration, errorMsg}
}

// truncateError 截断过长的错误信息
func truncateError(errorMsg string, maxLen int) string {
	if len(errorMsg) > maxLen {
		return errorMsg[:maxLen-3] + "..."
	}
	return errorMsg
}

// printRunDetailedOutput 打印详细输出信息
func printRunDetailedOutput(results []*ssh.Result) {
	fmt.Println("\n" + text.Colors{text.FgHiCyan}.Sprint(strings.Repeat("=", 80)))
	fmt.Println(text.Colors{text.FgHiCyan, text.Bold}.Sprint("详细输出"))
	fmt.Println(text.Colors{text.FgHiCyan}.Sprint(strings.Repeat("=", 80)))

	for _, result := range results {
		printHostDetailedOutput(result)
	}
}

// printHostDetailedOutput 打印单个主机的详细输出
func printHostDetailedOutput(result *ssh.Result) {
	isSuccess := result.Error == nil && result.ExitCode == 0
	hostColor := getHostColor(isSuccess)
	fmt.Printf("\n%s\n", hostColor.Sprint("["+result.Host+"]"))

	if result.Stdout != "" {
		fmt.Printf("%s\n%s\n",
			text.Colors{text.FgHiWhite}.Sprint("标准输出:"),
			result.Stdout)
	}

	if result.Stderr != "" {
		fmt.Printf("%s\n%s\n",
			text.Colors{text.FgRed}.Sprint("标准错误:"),
			text.Colors{text.FgRed}.Sprint(result.Stderr))
	}

	if result.Error != nil {
		fmt.Printf("%s %s\n",
			text.Colors{text.FgRed}.Sprint("错误:"),
			text.Colors{text.FgRed}.Sprint(result.Error.Error()))
	}

	fmt.Println(text.Colors{text.FgHiBlack}.Sprint(strings.Repeat("-", 80)))
}

// getHostColor 根据执行结果获取主机颜色
func getHostColor(isSuccess bool) text.Colors {
	if isSuccess {
		return text.Colors{text.FgGreen, text.Bold}
	}
	return text.Colors{text.FgRed, text.Bold}
}

// printRunSummary 打印执行结果摘要
func printRunSummary(results []*ssh.Result, stats *runStatistics, totalDuration time.Duration) {
	fmt.Printf("\n总计: %d 台主机 | %s | %s | 总耗时: %s\n",
		len(results),
		text.Colors{text.FgGreen}.Sprint(fmt.Sprintf("成功: %d", stats.successCount)),
		text.Colors{text.FgRed}.Sprint(fmt.Sprintf("失败: %d", stats.failCount)),
		totalDuration.Round(time.Millisecond).String())

	if len(stats.successHosts) > 0 {
		fmt.Printf("%s: %s\n",
			text.Colors{text.FgGreen, text.Bold}.Sprint("成功主机"),
			text.Colors{text.FgGreen}.Sprint(strings.Join(stats.successHosts, ", ")))
	}

	if len(stats.failHosts) > 0 {
		fmt.Printf("%s: %s\n",
			text.Colors{text.FgRed, text.Bold}.Sprint("失败主机"),
			text.Colors{text.FgRed}.Sprint(strings.Join(stats.failHosts, ", ")))
	}

	fmt.Println()
}

// PrintPingResults 打印 ping 命令的测试结果
// 显示所有主机的连接测试结果，包括成功/失败状态、延迟和错误信息
func PrintPingResults(results []*ssh.PingResult, totalDuration time.Duration) {
	successCount := 0
	failCount := 0

	validResults := make([]*ssh.PingResult, 0, len(results))
	for _, result := range results {
		if result != nil {
			validResults = append(validResults, result)
		}
	}

	if len(validResults) == 0 {
		fmt.Printf("\n总计: 0 台主机 | 总耗时: %s\n\n",
			totalDuration.Round(time.Millisecond).String())
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	setupTableStyle(t)
	t.AppendHeader(table.Row{"主机", "状态", "延迟", "错误信息"})

	for _, result := range validResults {
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
				if len(errorMsg) > 100 {
					errorMsg = errorMsg[:97] + "..."
				}
			}
		}

		t.AppendRow(table.Row{result.Host, status, duration, errorMsg})
	}

	fmt.Println()
	t.Render()

	fmt.Printf("\n总计: %d 台主机 | %s | %s | 总耗时: %s\n\n",
		len(validResults),
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

func printListIP(hosts []executor.Host) {
	for _, host := range hosts {
		fmt.Println(host.Address)
	}
}

func printListFull(hosts []executor.Host) {
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

// ProgressTracker 进度跟踪器
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
	pw.SetMessageLength(40)
	pw.SetNumTrackersExpected(1)
	pw.SetStyle(progress.StyleDefault)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 200)

	pw.Style().Colors.Message = text.Colors{text.FgHiWhite}
	pw.Style().Colors.Stats = text.Colors{text.FgHiWhite}
	pw.Style().Colors.Tracker = text.Colors{text.FgCyan}
	pw.Style().Options.PercentFormat = "%4.1f%%"
	pw.Style().Options.TimeInProgressPrecision = time.Millisecond * 100
	pw.Style().Options.TimeDonePrecision = time.Millisecond * 100
	pw.Style().Options.TimeOverallPrecision = time.Millisecond * 100

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

func (pt *ProgressTracker) Increment() {
	pt.mu.Lock()
	pt.completed++
	completed := pt.completed
	pt.mu.Unlock()

	if pt.tracker != nil {
		pt.tracker.SetValue(completed)
	}
}

func (pt *ProgressTracker) AddFailedHost(host string, reason string) {
	pt.mu.Lock()
	pt.failedHosts = append(pt.failedHosts, host)
	pt.mu.Unlock()

	fmt.Fprintf(os.Stderr, "%s %s: %s\n",
		text.Colors{text.FgRed}.Sprint("✗"),
		text.Colors{text.FgRed}.Sprint(host),
		text.Colors{text.FgRed}.Sprint(reason))
}

func (pt *ProgressTracker) GetFailedHosts() []string {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return pt.failedHosts
}

func (pt *ProgressTracker) Stop() {
	pt.mu.Lock()
	if pt.tracker != nil && pt.completed < pt.total {
		pt.tracker.SetValue(pt.total)
	}
	pt.mu.Unlock()

	pt.pw.Stop()
	time.Sleep(200 * time.Millisecond)
}

// PrintPingConfig 打印 ping 命令的配置参数
func PrintPingConfig(hostsFile, hostsDir, hostsString, group, user, keyPath, password, port string, concurrency int) {
	t := createConfigTable(false)
	data := &ConfigData{
		HostsFile:   hostsFile,
		HostsDir:    hostsDir,
		HostsString: hostsString,
		Group:       group,
		User:        user,
		KeyPath:     keyPath,
		Password:    password,
		Port:        port,
		Concurrency: concurrency,
	}
	printCommonConfig(t, data)
	renderConfigTable(t)
}

// PrintRunConfig 打印 run 命令的配置参数
func PrintRunConfig(hostsFile, hostsDir, hostsString, group, user, keyPath, password, port, command string, become bool, becomeUser string, concurrency int, showOutput bool) {
	t := createConfigTable(true)
	data := &ConfigData{
		HostsFile:    hostsFile,
		HostsDir:     hostsDir,
		HostsString:  hostsString,
		Group:        group,
		User:         user,
		KeyPath:      keyPath,
		Password:     password,
		Port:         port,
		Concurrency:  concurrency,
		Command:      command,
		Become:       become,
		BecomeUser:   becomeUser,
		ShowOutput:   showOutput,
		NeedWrapText: true,
	}
	printCommonConfig(t, data)

	if command != "" {
		wrapText := func(s string, maxWidth int) string {
			return text.WrapHard(s, maxWidth)
		}
		commandText := text.Colors{text.FgYellow}.Sprint(command)
		t.AppendRow(table.Row{"执行命令", wrapText(commandText, 80)})
	}

	printBecomeConfig(t, become, becomeUser)
	printOutputConfig(t, showOutput)
	renderConfigTable(t)
}

// PrintScriptConfig 打印 script 命令的配置参数
func PrintScriptConfig(hostsFile, hostsDir, hostsString, group, user, keyPath, password, port, scriptPath string, become bool, becomeUser string, concurrency int, showOutput bool) {
	t := createConfigTable(true)
	data := &ConfigData{
		HostsFile:    hostsFile,
		HostsDir:     hostsDir,
		HostsString:  hostsString,
		Group:        group,
		User:         user,
		KeyPath:      keyPath,
		Password:     password,
		Port:         port,
		Concurrency:  concurrency,
		ScriptPath:   scriptPath,
		Become:       become,
		BecomeUser:   becomeUser,
		ShowOutput:   showOutput,
		NeedWrapText: true,
	}
	printCommonConfig(t, data)

	if scriptPath != "" {
		wrapText := func(s string, maxWidth int) string {
			return text.WrapHard(s, maxWidth)
		}
		scriptText := text.Colors{text.FgYellow}.Sprint(scriptPath)
		t.AppendRow(table.Row{"执行脚本", wrapText(scriptText, 80)})
	}

	printBecomeConfig(t, become, becomeUser)
	printOutputConfig(t, showOutput)
	renderConfigTable(t)
}

// PrintUploadConfig 打印 upload 命令的配置参数
func PrintUploadConfig(hostsFile, hostsDir, hostsString, group, user, keyPath, password, port, localPath, remotePath, mode string, concurrency int, showOutput bool) {
	t := createConfigTable(true)
	data := &ConfigData{
		HostsFile:    hostsFile,
		HostsDir:     hostsDir,
		HostsString:  hostsString,
		Group:        group,
		User:         user,
		KeyPath:      keyPath,
		Password:     password,
		Port:         port,
		Concurrency:  concurrency,
		LocalPath:    localPath,
		RemotePath:   remotePath,
		Mode:         mode,
		ShowOutput:   showOutput,
		NeedWrapText: true,
	}
	printCommonConfig(t, data)

	wrapText := func(s string, maxWidth int) string {
		return text.WrapHard(s, maxWidth)
	}

	if localPath != "" {
		localText := text.Colors{text.FgYellow}.Sprint(localPath)
		t.AppendRow(table.Row{"本地路径", wrapText(localText, 80)})
	}
	if remotePath != "" {
		remoteText := text.Colors{text.FgYellow}.Sprint(remotePath)
		t.AppendRow(table.Row{"远程路径", wrapText(remoteText, 80)})
	}
	t.AppendRow(table.Row{"文件权限", text.Colors{text.FgCyan}.Sprint(getValueOrDefault(mode, "0644"))})

	printOutputConfig(t, showOutput)
	renderConfigTable(t)
}

// PrintListConfig 打印 list 命令的配置参数
func PrintListConfig(hostsFile, hostsDir, hostsString, group, format string) {
	t := createConfigTable(false)
	data := &ConfigData{
		HostsFile:   hostsFile,
		HostsDir:    hostsDir,
		HostsString: hostsString,
		Group:       group,
		Format:      format,
	}
	printCommonConfig(t, data)

	formatValue := getValueOrDefault(format, "ip")
	t.AppendRow(table.Row{"输出格式", text.Colors{text.FgCyan}.Sprint(formatValue)})

	renderConfigTable(t)
}

// getValueOrDefault 获取值或默认值
func getValueOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// ConfigData 配置数据结构
type ConfigData struct {
	HostsFile    string
	HostsDir     string
	HostsString  string
	Group        string
	User         string
	KeyPath      string
	Password     string
	Port         string
	Concurrency  int
	Command      string
	ScriptPath   string
	LocalPath    string
	RemotePath   string
	Mode         string
	Become       bool
	BecomeUser   string
	ShowOutput   bool
	Format       string
	NeedWrapText bool
}

// printCommonConfig 打印公共配置部分
func printCommonConfig(t table.Writer, data *ConfigData) {
	wrapText := func(s string, maxWidth int) string {
		if data.NeedWrapText {
			return text.WrapHard(s, maxWidth)
		}
		return s
	}

	var hostsSource string
	if data.HostsDir != "" {
		hostsSource = fmt.Sprintf("目录 (%s)", data.HostsDir)
	} else if data.HostsFile != "" {
		hostsSource = fmt.Sprintf("文件 (%s)", data.HostsFile)
	} else if data.HostsString != "" {
		hostsSource = fmt.Sprintf("命令行参数 (%s)", data.HostsString)
	} else {
		hostsSource = text.Colors{text.FgHiYellow}.Sprint("ansible.cfg inventory")
	}
	t.AppendRow(table.Row{"主机列表来源", wrapText(hostsSource, 80)})

	if data.Group != "" {
		t.AppendRow(table.Row{"分组", text.Colors{text.FgCyan}.Sprint(data.Group)})
	}

	t.AppendRow(table.Row{"用户名", getValueOrDefault(data.User, text.Colors{text.FgHiBlack}.Sprint("(未设置)"))})

	keyPathValue := getValueOrDefault(data.KeyPath, text.Colors{text.FgHiBlack}.Sprint("(未设置)"))
	t.AppendRow(table.Row{"SSH 密钥", wrapText(keyPathValue, 80)})

	if data.Password != "" {
		t.AppendRow(table.Row{"密码", text.Colors{text.FgGreen}.Sprint("***已设置***")})
	} else {
		t.AppendRow(table.Row{"密码", text.Colors{text.FgHiBlack}.Sprint("(未设置)")})
	}

	t.AppendRow(table.Row{"SSH 端口", getValueOrDefault(data.Port, "22")})

	if data.Concurrency > 0 {
		t.AppendRow(table.Row{"并发数", text.Colors{text.FgCyan}.Sprint(fmt.Sprintf("%d", data.Concurrency))})
	}
}

// printBecomeConfig 打印 Become 相关配置
func printBecomeConfig(t table.Writer, become bool, becomeUser string) {
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
}

// printOutputConfig 打印输出相关配置
func printOutputConfig(t table.Writer, showOutput bool) {
	var outputStatus string
	if showOutput {
		outputStatus = text.Colors{text.FgGreen}.Sprint("是")
	} else {
		outputStatus = text.Colors{text.FgHiBlack}.Sprint("否")
	}
	t.AppendRow(table.Row{"显示输出", outputStatus})
}

// createConfigTable 创建配置表格
func createConfigTable(needWrap bool) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	setupTableStyle(t)
	t.SetTitle(text.Colors{text.FgHiCyan, text.Bold}.Sprint("当前配置参数"))
	t.AppendHeader(table.Row{"参数", "值"})
	if needWrap {
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, WidthMax: 15},
			{Number: 2, WidthMax: 80},
		})
	}
	return t
}

// renderConfigTable 渲染配置表格
func renderConfigTable(t table.Writer) {
	fmt.Println()
	t.Render()
	fmt.Println()
}
