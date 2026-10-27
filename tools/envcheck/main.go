// 无限画布 环境检测工具
//
// 用途：在“点击没反应/接口连接失败/页面打不开”的机器上跑一遍，输出结论并生成报告文件，
// 方便把报告发给开发者定位。不依赖任何第三方运行库（单文件 exe，Go 静态编译）。
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const (
	appName        = "无限画布"
	toolVersion    = "1.0"
	uninstallKey   = `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\InfiniteCanvasWorkbench`
	webPortDefault = 3000
	apiPortDefault = 8080
)

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	user32                 = syscall.NewLazyDLL("user32.dll")
	procSetConsoleOutputCP = kernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleCP       = kernel32.NewProc("SetConsoleCP")
	procGetConsoleCP       = kernel32.NewProc("GetConsoleOutputCP")
	procGetStdHandle       = kernel32.NewProc("GetStdHandle")
	procGetFileType        = kernel32.NewProc("GetFileType")
	procMsgBox             = user32.NewProc("MessageBoxW")

	consoleGBK  = false
	stdinReader = bufio.NewReader(os.Stdin)
	report      strings.Builder
	problems    []string
	warnings    []string
)

type fileSpec struct {
	rel   string
	label string
	minKB int64
}

func main() {
	defer crashGuard()

	// 强制控制台使用 UTF-8（65001），避免 GBK 代码页下中文乱码；
	// 部分老系统（如未打补丁的 Win7）切换可能失败，此时按实际代码页输出 GBK
	procSetConsoleOutputCP.Call(65001)
	procSetConsoleCP.Call(65001)
	consoleGBK = stdoutIsConsole() && getConsoleOutputCP() != 65001

	printlnLine("")
	printlnLine("==================================================")
	printlnLine(fmt.Sprintf("  %s 环境检测工具 v%s", appName, toolVersion))
	printlnLine("  作用：检查安装是否完整、能不能在这台机器上跑起来")
	printlnLine("==================================================")
	printlnLine("")
	section("一、系统环境")

	checkSystem()
	installDir, source := findInstallDir()
	section("二、安装位置")
	if installDir == "" {
		line("FAIL", "没有找到安装目录", "注册表和常见路径里都没有找到本程序，可能没有安装成功")
		problems = append(problems, "没有找到安装目录：程序可能没有安装成功，或被安全软件删除了整个文件夹")
	} else {
		line("OK", "安装目录", installDir+"（来源："+source+"）")
		checkFiles(installDir)
		checkWritable(installDir)
		checkZoneMark(installDir)
	}
	section("三、运行状态")
	checkPorts()
	if installDir != "" {
		checkProcesses(installDir)
		if healthyOurPort() == 0 {
			tryLaunch(installDir)
		}
		checkLogs(installDir)
	}
	section("四、快捷方式与注册信息")
	checkShortcuts(installDir)
	checkUninstallEntry()
	section("五、安全软件与防火墙")
	checkFirewall()
	checkDefender()

	section("结论")
	printVerdict()

	reportPath := saveReport(installDir)
	printlnLine("")
	printlnLine("  报告文件：" + reportPath)
	printlnLine("")
	printlnLine("  按回车键退出...")
	fmt.Scanln()
}

// ---------- 一、系统环境 ----------

func checkSystem() {
	product, display, build, _ := windowsVersion()
	arch := os.Getenv("PROCESSOR_ARCHITECTURE")
	archW6432 := os.Getenv("PROCESSOR_ARCHITEW6432")
	line("OK", "系统版本", fmt.Sprintf("%s %s（Build %s）", product, display, build))
	line("OK", "程序位数", fmt.Sprintf("本工具 %s / 系统 %s", runtime.GOARCH, describeArch(arch, archW6432)))

	buildNum := atoi(build)
	if buildNum > 0 && buildNum < 10240 {
		line("FAIL", "系统版本过低", "本程序需要 Windows 10 及以上（内置的 Node 运行环境不再支持 Win7/Win8）")
		problems = append(problems, "系统版本低于 Windows 10，前后端都无法启动")
		printlnLine("")
		printlnLine("  ------------------------------------------------------------")
		printlnLine("  这台电脑是 " + product + "（内部版本 " + build + "），本程序不支持。")
		printlnLine("  程序内置的 Node 运行环境要求 Windows 10 及以上，")
		printlnLine("  在 Windows 7 / 8 / 8.1 上装上也用不了，双击图标不会有反应。")
		printlnLine("  请升级到 Windows 10 或 Windows 11 后再安装使用。")
		printlnLine("  ------------------------------------------------------------")
	}
	if arch == "x86" && archW6432 == "" {
		line("FAIL", "32 位系统", "安装包是 64 位程序，32 位 Windows 上双击会没有任何反应")
		problems = append(problems, "系统是 32 位，安装包（64 位）无法运行")
	}
}

func windowsVersion() (product, display, build string, err error) {
	product = regQuery(`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "ProductName")
	display = regQuery(`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "DisplayVersion")
	if display == "" {
		display = regQuery(`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "ReleaseId")
	}
	build = regQuery(`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "CurrentBuild")
	return
}

func describeArch(arch, archW6432 string) string {
	switch {
	case archW6432 != "":
		return "64 位"
	case arch == "AMD64":
		return "64 位"
	case arch == "ARM64":
		return "ARM64"
	case arch == "x86":
		return "32 位"
	}
	return arch
}

// ---------- 二、安装位置 ----------

func findInstallDir() (string, string) {
	if v := regQuery(uninstallKey, "InstallLocation"); v != "" {
		return strings.TrimRight(v, `\`), "注册表"
	}
	exe, _ := os.Executable()
	selfDir := filepath.Dir(exe)
	if fileExists(filepath.Join(selfDir, "server.exe")) {
		return selfDir, "检测工具所在目录"
	}
	local := os.Getenv("LOCALAPPDATA")
	candidates := []string{
		filepath.Join(local, "Programs", "InfiniteCanvas"),
		`D:\InfiniteCanvas`,
		`C:\InfiniteCanvas`,
	}
	for _, dir := range candidates {
		if dir != "" && fileExists(filepath.Join(dir, "server.exe")) {
			return dir, "常见安装路径"
		}
	}
	return "", ""
}

func checkFiles(dir string) {
	specs := []fileSpec{
		{"launcher.exe", "启动器", 2000},
		{"server.exe", "后端服务", 10000},
		{"node\\node.exe", "内置 Node 运行环境", 50000},
		{"web\\server.js", "前端服务入口", 5},
		{"web\\.next\\BUILD_ID", "前端构建产物", 0},
		{"logo.ico", "图标", 1},
	}
	missing := 0
	for _, spec := range specs {
		path := filepath.Join(dir, spec.rel)
		info, err := os.Stat(path)
		if err != nil {
			line("FAIL", spec.label+" 缺失", path+"（可能被杀毒软件删除或安装不完整）")
			missing++
			continue
		}
		sizeKB := info.Size() / 1024
		if sizeKB < spec.minKB {
			line("WARN", spec.label+" 大小异常", fmt.Sprintf("%s 只有 %d KB，疑似被杀毒软件改写成占位文件", path, sizeKB))
			warnings = append(warnings, spec.label+" 文件大小异常，可能被杀毒软件处理过")
			continue
		}
		extra := ""
		if strings.HasSuffix(spec.rel, ".exe") {
			if machine := peMachine(path); machine != "" && machine != "0x8664" {
				extra = "（位数：" + machine + "）"
			}
		}
		line("OK", spec.label, fmt.Sprintf("%s  %d KB%s", path, sizeKB, extra))
	}
	if missing > 0 {
		problems = append(problems, fmt.Sprintf("缺少 %d 个关键文件，双击快捷方式不会有任何反应", missing))
	}
}

func checkWritable(dir string) {
	dataDir := filepath.Join(dir, "data")
	writable := true
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		writable = false
		line("FAIL", "数据目录不可写", dataDir+"："+err.Error())
		problems = append(problems, "数据目录不可写（程序需要在该目录写入数据库与日志）")
	} else {
		probe := filepath.Join(dataDir, ".envcheck.tmp")
		if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
			writable = false
			line("FAIL", "数据目录不可写", dataDir+"："+err.Error())
			problems = append(problems, "数据目录不可写（程序需要在该目录写入数据库与日志）")
		} else {
			os.Remove(probe)
		}
	}
	if writable {
		line("OK", "数据目录可写", dataDir)
	}

	if fileExists(filepath.Join(dataDir, ".env")) {
		line("OK", "已有配置文件", filepath.Join(dataDir, ".env"))
	} else if writable {
		line("WARN", "还没有配置文件", "首次启动时会自动生成 data\\.env 与随机管理员密码")
	}
}

// tryLaunch 试着启动一次，用来判断“点击没反应”到底是完全起不来，还是只是启动慢
func tryLaunch(dir string) {
	launcher := filepath.Join(dir, "launcher.exe")
	if !fileExists(launcher) {
		return
	}
	printlnLine("")
	printlnLine("  是否现在尝试启动一次？（用来区分“完全起不来”和“只是启动慢”）(Y/N)")
	if readLine() != "y" {
		return
	}
	cmd := exec.Command(launcher)
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		line("FAIL", "启动器无法运行", err.Error())
		problems = append(problems, "启动器无法运行："+err.Error()+"（多半被安全软件拦截）")
		return
	}
	line("OK", "已尝试启动", launcher+"（PID "+strconv.Itoa(cmd.Process.Pid)+"）")
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)
		if port := healthyOurPort(); port > 0 {
			line("OK", "启动成功", fmt.Sprintf("服务已在 %d 端口响应，可以正常使用", port))
			return
		}
	}
	running := 0
	for _, name := range []string{"launcher.exe", "server.exe", "node.exe"} {
		for _, item := range processPaths(name) {
			if strings.HasPrefix(strings.ToLower(item), strings.ToLower(dir)) {
				running++
			}
		}
	}
	if running == 0 {
		line("FAIL", "启动没有任何反应", "30 秒内没有出现进程，也没有服务响应：基本可以确定是被安全软件拦截或文件被破坏")
		problems = append(problems, "启动器运行后没有产生任何进程，疑似被安全软件拦截，请检查杀毒软件的拦截/隔离记录")
		return
	}
	line("WARN", "进程已起来但服务没响应", "请在任务管理器确认，并把本报告发给开发者")
	warnings = append(warnings, "进程已启动但服务端口没有响应")
}

// healthyOurPort 返回第一个正在响应本程序健康检查的端口
func healthyOurPort() int {
	for _, base := range []int{webPortDefault, apiPortDefault} {
		for port := base; port < base+11; port++ {
			if isOurApp(port) {
				return port
			}
		}
	}
	return 0
}

// 检查文件是否带“网络下载”标记（会被 SmartScreen 拦），并询问是否解除
func checkZoneMark(dir string) {
	targets := []string{"launcher.exe", "server.exe", "node\\node.exe"}
	var marked []string
	for _, rel := range targets {
		path := filepath.Join(dir, rel)
		if _, err := os.Stat(path + ":Zone.Identifier"); err == nil {
			marked = append(marked, rel)
		}
	}
	if len(marked) == 0 {
		line("OK", "文件网络标记", "没有检测到被 Windows 标记为“来自网络”的文件")
		return
	}
	line("WARN", "文件带网络标记", strings.Join(marked, "、")+" 被标记为来自网络，可能被 SmartScreen 拦截")
	warnings = append(warnings, "部分程序文件带网络标记，建议解除后再试")
	printlnLine("")
	printlnLine("  是否解除这些文件的网络标记（等同于右键“解除锁定”）？(Y/N)")
	if readLine() == "y" {
		for _, rel := range marked {
			path := filepath.Join(dir, rel)
			if err := os.Remove(path + ":Zone.Identifier"); err == nil {
				printlnLine("  已解除：" + rel)
			} else {
				printlnLine("  解除失败：" + rel + "（" + err.Error() + "）")
			}
		}
	} else {
		printlnLine("  已跳过。")
	}
	printlnLine("")
}

// ---------- 三、运行状态 ----------

func checkPorts() {
	used := map[int]string{}
	for _, base := range []int{webPortDefault, apiPortDefault} {
		for port := base; port < base+11; port++ {
			if listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port)); err == nil {
				listener.Close()
				continue
			}
			used[port] = portOwner(port)
		}
	}
	if len(used) == 0 {
		line("OK", "端口可用", fmt.Sprintf("%d 与 %d 及后续 10 个端口都空闲", webPortDefault, apiPortDefault))
		return
	}
	keys := []int{}
	for port := range used {
		keys = append(keys, port)
	}
	sort.Ints(keys)
	for _, port := range keys {
		owner := used[port]
		if isOurApp(port) {
			line("OK", fmt.Sprintf("端口 %d 正在被本程序使用", port), owner+"（服务已在运行）")
		} else {
			detail := owner
			if detail == "" {
				detail = "未知程序"
			}
			line("WARN", fmt.Sprintf("端口 %d 被占用", port), detail+"（不是本程序，启动时会自动往后顺延）")
		}
	}
}

func checkProcesses(dir string) {
	running := []string{}
	for _, name := range []string{"launcher.exe", "server.exe", "node.exe"} {
		for _, item := range processPaths(name) {
			if strings.HasPrefix(strings.ToLower(item), strings.ToLower(dir)) {
				running = append(running, name+" ("+item+")")
			}
		}
	}
	if len(running) == 0 {
		line("WARN", "程序没有在运行", "当前没有本程序的进程，属于“未启动”状态（不算故障）")
		return
	}
	for _, item := range running {
		line("OK", "进程运行中", item)
	}
}

func checkLogs(dir string) {
	logDir := filepath.Join(dir, "data", "logs")
	for _, name := range []string{"launcher.log", "server.log", "web.log"} {
		path := filepath.Join(logDir, name)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		line("OK", name, fmt.Sprintf("%d KB，最后修改 %s", info.Size()/1024, info.ModTime().Format("2006-01-02 15:04:05")))
		tail := tailLines(path, 40)
		appendReport("---- " + name + " 末尾 40 行 ----\n" + tail + "\n")
		for _, row := range strings.Split(tail, "\n") {
			low := strings.ToLower(row)
			for _, keyword := range []string{"fail", "error", "panic", "refused", "bind:", "cannot find"} {
				if strings.Contains(low, keyword) {
					printlnLine("     日志提示：" + strings.TrimSpace(row))
					break
				}
			}
		}
	}
}

// ---------- 四、快捷方式与注册信息 ----------

func checkShortcuts(dir string) {
	dirs := []string{
		filepath.Join(os.Getenv("USERPROFILE"), "Desktop"),
		filepath.Join(os.Getenv("PUBLIC"), "Desktop"),
		filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs"),
		filepath.Join(os.Getenv("ProgramData"), "Microsoft", "Windows", "Start Menu", "Programs"),
	}

	// 先用 Windows Shell 解析快捷方式目标（最准），失败再退回字节启发式解析
	resolved := resolveShortcuts(dirs)
	found := 0
	for _, root := range dirs {
		if root == "" {
			continue
		}
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".lnk") {
				return nil
			}
			target := resolved[path]
			if target == "" {
				data, err := os.ReadFile(path)
				if err != nil || !bytes.Contains(data, []byte("launcher.exe")) {
					return nil
				}
				target = linkTarget(data)
			}
			if !isOurShortcut(path, target) {
				return nil
			}
			found++
			if target != "" && !fileExists(target) {
				line("FAIL", "快捷方式指向的文件不存在", path+" -> "+target)
				problems = append(problems, "快捷方式指向的文件不存在（点击不会有反应）："+target)
				return nil
			}
			if target != "" {
				line("OK", "快捷方式", path+" -> "+target)
				return nil
			}
			line("WARN", "快捷方式", path+"（没能解析出目标路径，建议重新安装生成快捷方式）")
			return nil
		})
	}
	if found == 0 {
		line("WARN", "没有找到指向本程序的快捷方式", "桌面与开始菜单里都没有找到，请从安装目录直接运行 launcher.exe")
	}
}

// 判断这个快捷方式是不是指向本程序
func isOurShortcut(lnkPath, target string) bool {
	if target == "" {
		return strings.Contains(strings.ToLower(lnkPath), "infinitecanvas")
	}
	lower := strings.ToLower(target)
	if strings.Contains(lower, "infinitecanvas") {
		return true
	}
	if strings.HasSuffix(lower, "launcher.exe") && fileExists(filepath.Join(filepath.Dir(target), "server.exe")) {
		return true
	}
	return false
}

// resolveShortcuts 用 WScript.Shell 解析快捷方式，返回 “lnk路径 -> 目标路径” 映射
func resolveShortcuts(dirs []string) map[string]string {
	result := map[string]string{}
	quoted := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if dir != "" {
			quoted = append(quoted, "'"+strings.ReplaceAll(dir, "'", "''")+"'")
		}
	}
	script := fmt.Sprintf(
		"$shell = New-Object -ComObject WScript.Shell; foreach ($d in @(%s)) { if (Test-Path $d) { Get-ChildItem $d -Recurse -Filter *.lnk -ErrorAction SilentlyContinue | ForEach-Object { try { $t = $shell.CreateShortcut($_.FullName).TargetPath; if ($t) { $_.FullName + '|' + $t } } catch { } } } }",
		strings.Join(quoted, ","))
	out := runCmd("powershell", "-NoProfile", "-Command", script)
	for _, row := range strings.Split(out, "\n") {
		row = strings.TrimSpace(row)
		parts := strings.SplitN(row, "|", 2)
		if len(parts) != 2 {
			continue
		}
		result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return result
}

func checkUninstallEntry() {
	name := regQuery(uninstallKey, "DisplayName")
	if name == "" {
		line("WARN", "注册表没有安装记录", "“设置-应用”里可能看不到本程序（不影响使用）")
		return
	}
	line("OK", "注册表安装记录", name+" "+regQuery(uninstallKey, "DisplayVersion")+"  位置："+regQuery(uninstallKey, "InstallLocation"))
}

// ---------- 五、安全软件与防火墙 ----------

func checkFirewall() {
	out := runCmd("netsh", "advfirewall", "show", "allprofiles", "state")
	states := []string{}
	for _, row := range strings.Split(out, "\n") {
		row = strings.TrimSpace(row)
		if strings.HasPrefix(row, "State") || strings.Contains(row, "状态") {
			states = append(states, strings.Join(strings.Fields(row), " "))
		}
	}
	if len(states) == 0 {
		line("WARN", "防火墙状态未知", "netsh 没有返回结果（可能是权限限制）")
		return
	}
	line("OK", "防火墙状态", strings.Join(states, "; ")+"（只影响局域网访问，本机访问不受影响）")
}

func checkDefender() {
	out := runCmd("powershell", "-NoProfile", "-Command",
		"try { $s = Get-MpComputerStatus -ErrorAction Stop; 'RTP=' + $s.RealTimeProtectionEnabled } catch { 'NA' }")
	out = strings.TrimSpace(out)
	if out == "" || out == "NA" {
		line("WARN", "杀毒软件状态未知", "无法读取 Defender 状态（可能装了第三方杀毒，属正常）")
		return
	}
	line("OK", "Defender 实时保护", out+"（如果文件缺失，请检查杀毒软件的隔离区）")
}

// ---------- 结论与报告 ----------

func printVerdict() {
	appendReport("==== 结论 ====\n")
	if len(problems) == 0 && len(warnings) == 0 {
		line("OK", "环境正常", "没有发现阻塞性问题，可以直接双击启动器使用")
		appendReport("环境正常：没有发现阻塞性问题。\n")
		return
	}
	for _, item := range problems {
		line("FAIL", "阻塞问题", item)
		appendReport("[阻塞] " + item + "\n")
	}
	for _, item := range warnings {
		line("WARN", "注意事项", item)
		appendReport("[提示] " + item + "\n")
	}
	if len(problems) == 0 {
		line("OK", "可以使用", "只发现了提示项，不影响启动")
	}
	appendReport("\n建议：把本报告文件发给帮你安装的人；报告里包含了系统信息、文件清单与日志末尾，够定位问题。\n")
}

func section(title string) {
	printlnLine("")
	printlnLine("【" + title + "】")
	appendReport("\n==== " + title + " ====\n")
}

func line(level, name, detail string) {
	mark := "  [√] "
	switch level {
	case "WARN":
		mark = "  [!] "
	case "FAIL":
		mark = "  [×] "
	}
	printlnLine(mark + name + "：" + detail)
	appendReport("[" + level + "] " + name + "：" + detail + "\n")
}

func appendReport(text string) {
	report.WriteString(text)
}

func saveReport(installDir string) string {
	fileName := fmt.Sprintf("%s-环境检测报告-%s.txt", appName, time.Now().Format("20060102-150405"))
	candidates := []string{
		filepath.Join(os.Getenv("USERPROFILE"), "Desktop"),
		os.Getenv("USERPROFILE"),
		os.Getenv("TEMP"),
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}
	local := os.Getenv("LOCALAPPDATA")
	if local != "" {
		candidates = append(candidates, filepath.Join(local, "InfiniteCanvas"))
	}

	header := fmt.Sprintf("%s 环境检测报告\n生成时间：%s\n检测工具版本：v%s\n安装目录：%s\n\n",
		appName, time.Now().Format("2006-01-02 15:04:05"), toolVersion, installDir)
	// UTF-8 BOM，保证记事本正确显示中文
	content := append([]byte{0xEF, 0xBB, 0xBF}, []byte(header+report.String())...)

	var lastErr error
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			lastErr = err
			continue
		}
		path := filepath.Join(dir, fileName)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			lastErr = err
			continue
		}
		summary := "没有发现阻塞性问题，可以双击启动器使用。"
		if len(problems) > 0 {
			summary = "发现问题：\n" + strings.Join(problems, "\n")
		} else if len(warnings) > 0 {
			summary = "有提示项（不影响启动）：\n" + strings.Join(warnings, "\n")
		}
		showBox(appName+" 环境检测完成", summary+"\n\n完整报告已保存到：\n"+path)
		return path
	}
	if lastErr != nil {
		printlnLine("  [×] 报告写入失败：" + lastErr.Error())
	}
	return ""
}

// ---------- 工具函数 ----------

func printlnLine(text string) {
	writeOut(text + "\r\n")
}

func writeOut(text string) {
	if consoleGBK {
		if encoded, _, err := transform.String(simplifiedchinese.GBK.NewEncoder(), text); err == nil {
			os.Stdout.WriteString(encoded)
			return
		}
	}
	os.Stdout.WriteString(text)
}

// crashGuard 兜住意外崩溃，弹窗告知原因，避免“双击闪退、什么也看不到”
func crashGuard() {
	if r := recover(); r != nil {
		showBox(appName+" 环境检测工具 运行出错",
			fmt.Sprintf("检测过程中出现意外错误：\n%v\n\n请把这段信息发给帮你安装的人。", r))
	}
}

// stdoutIsConsole 判断标准输出是否连着真正的控制台窗口（被重定向/管道时为 false）
func stdoutIsConsole() bool {
	handle, _, _ := procGetStdHandle.Call(uintptr(0xFFFFFFF5)) // STD_OUTPUT_HANDLE
	if handle == 0 || handle == ^uintptr(0) {                  // NULL / INVALID_HANDLE_VALUE
		return false
	}
	fileType, _, _ := procGetFileType.Call(handle)
	return fileType == 2 // FILE_TYPE_CHAR
}

func getConsoleOutputCP() uint32 {
	value, _, _ := procGetConsoleCP.Call()
	return uint32(value)
}

func readLine() string {
	text, _ := stdinReader.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(text))
}

func showBox(title, text string) {
	t, err1 := syscall.UTF16PtrFromString(title)
	m, err2 := syscall.UTF16PtrFromString(text)
	if err1 != nil || err2 != nil {
		return
	}
	procMsgBox.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), uintptr(0x40))
}

func regQuery(key, name string) string {
	// 32 位进程在 64 位系统上会被重定向到 WOW6432Node，
	// 先按 64 位视图读（/reg:64），32 位系统上该参数会报错，再退回默认视图
	if value := regQueryView(key, name, "/reg:64"); value != "" {
		return value
	}
	return regQueryView(key, name, "")
}

func regQueryView(key, name, view string) string {
	args := []string{"query", key, "/v", name}
	if view != "" {
		args = append(args, view)
	}
	out := runCmd("reg", args...)
	for _, row := range strings.Split(out, "\n") {
		row = strings.TrimSpace(row)
		if !strings.HasPrefix(row, name) {
			continue
		}
		fields := strings.Fields(row)
		if len(fields) >= 3 {
			return strings.TrimSpace(strings.Join(fields[2:], " "))
		}
	}
	return ""
}

func runCmd(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return ""
	}
	return decodeWindowsText(out)
}

func decodeWindowsText(data []byte) string {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if utf8Valid(data) {
		return string(data)
	}
	if decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), data); err == nil {
		return string(decoded)
	}
	return string(data)
}

func utf8Valid(data []byte) bool {
	return strings.ToValidUTF8(string(data), "\uFFFD") == string(data)
}

func atoi(text string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(text))
	return value
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func tailLines(path string, count int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text := decodeWindowsText(data)
	lines := strings.Split(strings.TrimRight(text, "\r\n"), "\n")
	if len(lines) > count {
		lines = lines[len(lines)-count:]
	}
	return strings.Join(lines, "\n")
}

func peMachine(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	header := make([]byte, 0x40)
	if _, err := io.ReadFull(file, header); err != nil {
		return ""
	}
	if header[0] != 'M' || header[1] != 'Z' {
		return ""
	}
	offset := int(header[0x3C]) | int(header[0x3D])<<8 | int(header[0x3E])<<16 | int(header[0x3F])<<24
	if offset <= 0 {
		return ""
	}
	buf := make([]byte, 6)
	if _, err := file.ReadAt(buf, int64(offset)); err != nil {
		return ""
	}
	if buf[0] != 'P' || buf[1] != 'E' {
		return ""
	}
	machine := int(buf[4]) | int(buf[5])<<8
	return fmt.Sprintf("0x%x", machine)
}

func portOwner(port int) string {
	out := runCmd("netstat", "-ano", "-p", "TCP")
	suffix := fmt.Sprintf(":%d ", port)
	pid := ""
	for _, row := range strings.Split(out, "\n") {
		fields := strings.Fields(strings.TrimSpace(row))
		if len(fields) < 5 || fields[0] != "TCP" {
			continue
		}
		if !strings.HasSuffix(fields[1], suffix) || !strings.EqualFold(fields[3], "LISTENING") {
			continue
		}
		pid = fields[4]
		break
	}
	if pid == "" {
		return ""
	}
	names := processPaths(pid)
	if len(names) > 0 {
		return fmt.Sprintf("PID %s（%s）", pid, strings.Join(names, ", "))
	}
	return "PID " + pid
}

func processPaths(nameOrPID string) []string {
	condition := ""
	if pid := atoi(nameOrPID); pid > 0 {
		condition = fmt.Sprintf("ProcessId=%d", pid)
	} else {
		condition = fmt.Sprintf("Name='%s'", nameOrPID)
	}
	out := runCmd("powershell", "-NoProfile", "-Command",
		fmt.Sprintf("Get-CimInstance Win32_Process -Filter \"%s\" -ErrorAction SilentlyContinue | ForEach-Object { $_.ExecutablePath }", condition))
	paths := collectPaths(out)
	if len(paths) > 0 {
		return paths
	}
	// Windows 7 自带的 PowerShell 2.0 没有 Get-CimInstance，退回 wmic 查询
	return collectPaths(runCmd("wmic", "process", "where", condition, "get", "ExecutablePath", "/value"))
}

// collectPaths 从命令输出里挑出真正的可执行文件路径，过滤 wmic 的提示噪音
func collectPaths(out string) []string {
	var paths []string
	for _, row := range strings.Split(out, "\n") {
		row = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(row), "ExecutablePath="))
		if strings.Contains(row, ":\\") || strings.HasPrefix(row, `\\`) {
			paths = append(paths, row)
		}
	}
	return paths
}

func isOurApp(port int) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/api/health", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32))
	return resp.StatusCode == http.StatusOK && strings.TrimSpace(string(body)) == "ok"
}

// linkTarget 从 .lnk 文件里粗提取目标路径（够用即可，不做完整 ShellLink 解析）
func linkTarget(data []byte) string {
	utf16 := make([]byte, 0, len(data)*2)
	for _, b := range data {
		utf16 = append(utf16, b, 0)
	}
	candidates := []string{decodeUTF16(data), string(data), decodeUTF16(utf16)}
	best := ""
	for _, text := range candidates {
		index := strings.Index(strings.ToLower(text), "launcher.exe")
		if index < 0 {
			continue
		}
		head := text[:index]
		for i := len(head) - 1; i >= 0; i-- {
			if !isPathChar(head[i]) {
				path := head[i+1:]
				if looksLikePath(path) && len(path) > len(best) {
					best = path
				}
				break
			}
		}
	}
	if best == "" {
		return ""
	}
	return filepath.FromSlash(strings.TrimRight(best, "\\"))
}

func isPathChar(ch byte) bool {
	return ch == '\\' || ch == '/' || ch == ':' || ch == '.' || ch == '-' || ch == '_' || ch == ' ' ||
		(ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch > 127
}

func looksLikePath(text string) bool {
	if len(text) < 6 {
		return false
	}
	return strings.Contains(text, ":\\") || strings.HasPrefix(text, "\\\\")
}

func decodeUTF16(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		value := rune(data[i]) | rune(data[i+1])<<8
		if value == 0 {
			continue
		}
		runes = append(runes, value)
	}
	return string(runes)
}
