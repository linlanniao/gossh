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
func PrintRunResults(results []*ssh.Result, totalDuration time.Duration, showOutput bool, group string, hosts []executor.Host) {
	stats := collectRunStatistics(results)

	printRunResultsTable(results, stats, group, hosts)

	if showOutput {
		printRunDetailedOutput(results)
	}

	printRunSummary(results, stats, totalDuration, group)
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
func printRunResultsTable(results []*ssh.Result, stats *runStatistics, group string, hosts []executor.Host) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	setupTableStyle(t)
	t.AppendHeader(table.Row{"主机", "分组", "状态", "退出码", "耗时", "错误信息"})

	for _, result := range results {
		row := buildResultTableRow(result, group, hosts)
		t.AppendRow(row)
	}

	fmt.Println()
	t.Render()
}

// buildResultTableRow 构建结果表格行
func buildResultTableRow(result *ssh.Result, group string, hosts []executor.Host) table.Row {
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

	// 查找主机所属的分组
	groups := "-"
	for _, host := range hosts {
		if host.Address == result.Host {
			if len(host.Groups) > 0 {
				groups = strings.Join(host.Groups, ",")
			}
			break
		}
	}

	return table.Row{result.Host, groups, status, exitCode, duration, errorMsg}
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
func printRunSummary(results []*ssh.Result, stats *runStatistics, totalDuration time.Duration, group string) {
	groupText := group
	if groupText == "" {
		groupText = "-"
	}
	fmt.Printf("\n总计: %d 台主机 | %s | %s | %s | 总耗时: %s\n",
		len(results),
		text.Colors{text.FgCyan}.Sprint(fmt.Sprintf("分组: %s", groupText)),
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
func PrintPingResults(results []*ssh.PingResult, totalDuration time.Duration, group string, hosts []executor.Host) {
	successCount := 0
	failCount := 0

	validResults := make([]*ssh.PingResult, 0, len(results))
	for _, result := range results {
		if result != nil {
			validResults = append(validResults, result)
		}
	}

	// 构建主机地址到分组的映射
	hostGroupsMap := make(map[string]string)
	for _, host := range hosts {
		key := fmt.Sprintf("%s:%s", host.Address, host.Port)
		if len(host.Groups) > 0 {
			hostGroupsMap[key] = strings.Join(host.Groups, ",")
		} else {
			hostGroupsMap[key] = "-"
		}
	}

	if len(validResults) == 0 {
		groupText := group
		if groupText == "" {
			groupText = "-"
		}
		fmt.Printf("\n总计: 0 台主机 | %s | 总耗时: %s\n\n",
			text.Colors{text.FgCyan}.Sprint(fmt.Sprintf("分组: %s", groupText)),
			totalDuration.Round(time.Millisecond).String())
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	setupTableStyle(t)
	t.AppendHeader(table.Row{"主机", "分组", "状态", "延迟", "错误信息"})

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
				errorMsg = truncateError(result.Error.Error(), 50)
			}
		}

		// 查找主机所属的分组
		groups := "-"
		for _, host := range hosts {
			if host.Address == result.Host {
				if len(host.Groups) > 0 {
					groups = strings.Join(host.Groups, ",")
				}
				break
			}
		}
		t.AppendRow(table.Row{result.Host, groups, status, duration, errorMsg})
	}

	fmt.Println()
	t.Render()

	groupText := group
	if groupText == "" {
		groupText = "-"
	}
	fmt.Printf("\n总计: %d 台主机 | %s | %s | %s | 总耗时: %s\n\n",
		len(validResults),
		text.Colors{text.FgCyan}.Sprint(fmt.Sprintf("分组: %s", groupText)),
		text.Colors{text.FgGreen}.Sprint(fmt.Sprintf("成功: %d", successCount)),
		text.Colors{text.FgRed}.Sprint(fmt.Sprintf("失败: %d", failCount)),
		totalDuration.Round(time.Millisecond).String())
}

// PrintListResults 打印 list 命令的主机列表
// format: ip（仅IP地址）、full（完整信息）、json（JSON格式）
// oneLine: 是否一行输出（逗号分隔）
func PrintListResults(hosts []executor.Host, format string, oneLine bool) {
	switch format {
	case "json":
		printListJSON(hosts)
	case "full":
		if oneLine {
			printListFullOneLine(hosts)
		} else {
			printListFull(hosts)
		}
	default: // "ip"
		if oneLine {
			printListIPOneLine(hosts)
		} else {
			printListIP(hosts)
		}
	}
}

func printListIP(hosts []executor.Host) {
	for _, host := range hosts {
		fmt.Println(host.Address)
	}
}

func printListIPOneLine(hosts []executor.Host) {
	if len(hosts) == 0 {
		return
	}
	addresses := make([]string, len(hosts))
	for i, host := range hosts {
		addresses[i] = host.Address
	}
	fmt.Println(strings.Join(addresses, ","))
}

func printListFullOneLine(hosts []executor.Host) {
	if len(hosts) == 0 {
		return
	}
	items := make([]string, len(hosts))
	for i, host := range hosts {
		port := host.Port
		if port == "" {
			port = "22"
		}
		item := host.Address + ":" + port
		if host.User != "" {
			item = host.User + "@" + item
		}
		items[i] = item
	}
	fmt.Println(strings.Join(items, ","))
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

// PrintGroupResults 打印 list-group 命令的组列表
// oneLine: 是否一行输出（逗号分隔）
func PrintGroupResults(groups []string, oneLine bool) {
	if len(groups) == 0 {
		fmt.Println("未找到任何组")
		return
	}

	if oneLine {
		// 一行输出：逗号分隔
		fmt.Println(strings.Join(groups, ","))
	} else {
		// 简单输出：每行一个组名
		for _, group := range groups {
			fmt.Println(group)
		}
	}
}

const (
	// maxIndividualTrackers 最大独立 tracker 数量
	// 超过此数量时，只显示聚合 tracker
	maxIndividualTrackers = 20
)

// ProgressTracker 进度跟踪器
type ProgressTracker struct {
	pw             progress.Writer
	trackers       map[string]*progress.Tracker // 主机地址 -> tracker 的映射
	overallTracker *progress.Tracker            // 总体进度 tracker
	total          int                          // 总主机数
	completed      int                          // 已完成数量
	failed         int                          // 失败数量
	showIndividual bool                         // 是否显示独立 tracker
	allHosts       map[string]bool              // 所有主机地址集合，用于跟踪未完成的主机
	mu             sync.Mutex
}

// NewProgressTracker 创建新的进度跟踪器
func NewProgressTracker(total int, title string) *ProgressTracker {
	pw := progress.NewWriter()
	pw.SetAutoStop(true)
	pw.SetTrackerLength(50)
	pw.SetMessageLength(40)

	// 根据主机数量决定显示策略
	showIndividual := total <= maxIndividualTrackers
	if showIndividual {
		pw.SetNumTrackersExpected(total)
	} else {
		// 只显示一个总体 tracker
		pw.SetNumTrackersExpected(1)
	}

	pw.SetStyle(progress.StyleDefault)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.SetSortBy(progress.SortByPercentDsc)

	pw.Style().Colors.Message = text.Colors{text.FgHiWhite}
	pw.Style().Colors.Stats = text.Colors{text.FgHiWhite}
	pw.Style().Colors.Tracker = text.Colors{text.FgCyan}
	pw.Style().Options.PercentFormat = "%4.1f%%"
	pw.Style().Options.TimeInProgressPrecision = time.Millisecond * 100
	pw.Style().Options.TimeDonePrecision = time.Millisecond * 100
	pw.Style().Options.TimeOverallPrecision = time.Millisecond * 100
	// 隐藏自动添加的状态文本和分隔符，只保留中文状态提示
	pw.Style().Options.DoneString = ""
	pw.Style().Options.ErrorString = ""
	pw.Style().Options.Separator = ""

	progressTracker := &ProgressTracker{
		pw:             pw,
		trackers:       make(map[string]*progress.Tracker),
		total:          total,
		completed:      0,
		failed:         0,
		showIndividual: showIndividual,
		allHosts:       make(map[string]bool),
	}

	// 如果主机数量较多，创建总体 tracker
	if !showIndividual {
		overallTracker := &progress.Tracker{
			Message: fmt.Sprintf("%-40s", title),
			Total:   int64(total),
			Units:   progress.UnitsDefault,
		}
		pw.AppendTracker(overallTracker)
		progressTracker.overallTracker = overallTracker
	}

	// 启动渲染
	go pw.Render()

	return progressTracker
}

// AddTracker 为主机添加一个 tracker
func (pt *ProgressTracker) AddTracker(host string) interface{} {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// 记录所有主机
	pt.allHosts[host] = false // false 表示未完成

	// 如果主机数量较多，不创建独立 tracker
	if !pt.showIndividual {
		return nil
	}

	tracker := &progress.Tracker{
		Message: fmt.Sprintf("%-40s", host),
		Total:   100,
		Units:   progress.UnitsDefault,
	}
	pt.pw.AppendTracker(tracker)
	pt.trackers[host] = tracker

	return tracker
}

// UpdateTracker 更新指定主机的 tracker 进度
func (pt *ProgressTracker) UpdateTracker(host string, value int64, message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.showIndividual {
		// 显示独立 tracker 模式
		tracker, exists := pt.trackers[host]
		if !exists {
			// 如果 tracker 不存在，创建一个
			tracker = &progress.Tracker{
				Message: fmt.Sprintf("%-40s", host),
				Total:   100,
				Units:   progress.UnitsDefault,
			}
			pt.pw.AppendTracker(tracker)
			pt.trackers[host] = tracker
		}

		if value > 0 {
			tracker.SetValue(value)
		}
		if message != "" {
			tracker.UpdateMessage(fmt.Sprintf("%-40s", message))
		}
	} else {
		// 聚合模式：只更新总体 tracker
		if pt.overallTracker != nil {
			// 计算总体进度（已完成数量）
			completed := pt.completed
			pt.overallTracker.SetValue(int64(completed))

			// 更新消息显示统计信息
			statusMsg := fmt.Sprintf("已完成: %d/%d", completed, pt.total)
			if pt.failed > 0 {
				statusMsg += fmt.Sprintf(" | 失败: %d", pt.failed)
			}
			pt.overallTracker.UpdateMessage(fmt.Sprintf("%-40s", statusMsg))
		}
	}
}

// MarkTrackerDone 标记 tracker 为完成
func (pt *ProgressTracker) MarkTrackerDone(host string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// 标记主机为已完成
	if _, exists := pt.allHosts[host]; exists && !pt.allHosts[host] {
		pt.allHosts[host] = true
		pt.completed++
	}

	if pt.showIndividual {
		tracker, exists := pt.trackers[host]
		if exists {
			// 更新消息为带颜色的成功状态
			successMsg := text.Colors{text.FgGreen}.Sprint(fmt.Sprintf("%s (成功)", host))
			tracker.UpdateMessage(fmt.Sprintf("%-40s", successMsg))
			tracker.MarkAsDone()
		}
	} else {
		// 更新总体 tracker
		if pt.overallTracker != nil {
			pt.overallTracker.SetValue(int64(pt.completed))
			statusMsg := fmt.Sprintf("已完成: %d/%d", pt.completed, pt.total)
			if pt.failed > 0 {
				statusMsg += fmt.Sprintf(" | 失败: %d", pt.failed)
			}
			pt.overallTracker.UpdateMessage(fmt.Sprintf("%-40s", statusMsg))
		}
	}
}

// MarkTrackerErrored 标记 tracker 为错误
func (pt *ProgressTracker) MarkTrackerErrored(host string, reason string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// 标记主机为已完成（失败也算完成）
	if _, exists := pt.allHosts[host]; exists && !pt.allHosts[host] {
		pt.allHosts[host] = true
		pt.completed++
		pt.failed++
	}

	if pt.showIndividual {
		tracker, exists := pt.trackers[host]
		if exists {
			// 更新消息为带颜色的失败状态
			failMsg := text.Colors{text.FgRed}.Sprint(fmt.Sprintf("%s (失败)", host))
			tracker.UpdateMessage(fmt.Sprintf("%-40s", failMsg))
			tracker.MarkAsErrored()
		}
	} else {
		// 更新总体 tracker
		if pt.overallTracker != nil {
			pt.overallTracker.SetValue(int64(pt.completed))
			statusMsg := fmt.Sprintf("已完成: %d/%d | 失败: %d", pt.completed, pt.total, pt.failed)
			pt.overallTracker.UpdateMessage(fmt.Sprintf("%-40s", statusMsg))
		}
	}
}

// Stop 停止进度跟踪器
func (pt *ProgressTracker) Stop() {
	pt.mu.Lock()

	// 检查未完成的主机并标记为超时
	timeoutCount := 0
	for host, completed := range pt.allHosts {
		if !completed {
			timeoutCount++
			pt.allHosts[host] = true // 标记为已完成（超时）
			pt.completed++
			pt.failed++

			if pt.showIndividual {
				tracker, exists := pt.trackers[host]
				if exists {
					// 更新消息为带颜色的超时状态
					timeoutMsg := text.Colors{text.FgRed}.Sprint(fmt.Sprintf("%s (超时)", host))
					tracker.UpdateMessage(fmt.Sprintf("%-40s", timeoutMsg))
					tracker.MarkAsErrored()
				}
			}
		}
	}

	// 确保总体 tracker 显示最终状态
	if !pt.showIndividual && pt.overallTracker != nil {
		pt.overallTracker.SetValue(int64(pt.total))
		statusMsg := fmt.Sprintf("已完成: %d/%d", pt.completed, pt.total)
		if pt.failed > 0 {
			statusMsg += fmt.Sprintf(" | 失败: %d", pt.failed)
		}
		if timeoutCount > 0 {
			statusMsg += fmt.Sprintf(" | 超时: %d", timeoutCount)
		}
		pt.overallTracker.UpdateMessage(fmt.Sprintf("%-40s", statusMsg))
		pt.overallTracker.MarkAsDone()
	}

	pt.mu.Unlock()

	pt.pw.Stop()
	time.Sleep(200 * time.Millisecond)
}

// PrintPingConfig 打印 ping 命令的配置参数
func PrintPingConfig(inventory, group, user, keyPath, password, port string, concurrency int, timeout time.Duration) {
	t := createConfigTable(false)
	data := &ConfigData{
		Inventory:   inventory,
		Group:       group,
		User:        user,
		KeyPath:     keyPath,
		Password:    password,
		Port:        port,
		Concurrency: concurrency,
	}
	printCommonConfig(t, data)

	// 显示超时时间
	timeoutValue := timeout
	if timeoutValue <= 0 {
		timeoutValue = 30 * time.Second
	}
	t.AppendRow(table.Row{"连接超时", text.Colors{text.FgCyan}.Sprint(timeoutValue.String())})

	renderConfigTable(t)
}

// PrintRunConfig 打印 run 命令的配置参数
func PrintRunConfig(inventory, group, user, keyPath, password, port, command string, become bool, becomeUser string, concurrency int, showOutput bool) {
	t := createConfigTable(true)
	data := &ConfigData{
		Inventory:    inventory,
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
func PrintScriptConfig(inventory, group, user, keyPath, password, port, scriptPath string, become bool, becomeUser string, concurrency int, showOutput bool) {
	t := createConfigTable(true)
	data := &ConfigData{
		Inventory:    inventory,
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
func PrintUploadConfig(inventory, group, user, keyPath, password, port, localPath, remotePath, mode string, concurrency int, showOutput bool) {
	t := createConfigTable(true)
	data := &ConfigData{
		Inventory:    inventory,
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
func PrintListConfig(inventory, group, format string) {
	t := createConfigTable(false)
	data := &ConfigData{
		Inventory: inventory,
		Group:     group,
		Format:    format,
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
	Inventory    string // 主机列表（文件路径、目录路径或逗号分隔的主机列表）
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
	if data.Inventory != "" {
		hostsSource = fmt.Sprintf("inventory (%s)", data.Inventory)
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
