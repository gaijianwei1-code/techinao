// 无限画布 Windows 启动器
// 用法（快捷方式）：无参数 = 启动并打开浏览器；stop = 停止；status = 状态；logs = 打开日志目录
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	createNoWindow   = 0x08000000
	detachedProcess  = 0x00000008
	msgBoxOK         = 0x00000000
	msgBoxIconError  = 0x00000010
	msgBoxIconInfo   = 0x00000040
	defaultWebPort   = 3000
	defaultAPIPort   = 8080
	fallbackAppDir   = "InfiniteCanvas"
	readyTimeoutSec  = 150
	configFileName  = ".env"
	pidFileName     = ".pids"
	portFileName    = ".port"
	credentialsName = "账号密码.txt"
	adminUserName   = "admin"
)

var (
	user32     = syscall.NewLazyDLL("user32.dll")
	procMsgBox = user32.NewProc("MessageBoxW")
	logFile    *os.File
)

func msgBox(title, text string, flags uintptr) {
	t, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	m, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	procMsgBox.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), flags)
}

func logf(format string, args ...any) {
	line := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
	if logFile != nil {
		logFile.WriteString(line)
		logFile.Sync()
	}
	fmt.Print(line)
}

func exeDir() string {
	p, err := os.Executable()
	if err != nil {
		return "."
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		p = resolved
	}
	return filepath.Dir(p)
}

func ensureDir(path string) error { return os.MkdirAll(path, 0o755) }

func writable(dir string) bool {
	if err := ensureDir(dir); err != nil {
		return false
	}
	f, err := os.CreateTemp(dir, ".wtest")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

// 数据目录优先放在安装目录下的 data，不可写时退回 %LOCALAPPDATA%
func resolveDataDir(appDir string) string {
	local := filepath.Join(appDir, "data")
	if writable(local) {
		return local
	}
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		fallback := filepath.Join(la, fallbackAppDir, "data")
		if writable(fallback) {
			return fallback
		}
	}
	return local
}

func randomBytes(n int) []byte {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		fallback := []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
		copy(buf, fallback)
	}
	return buf
}

func randomPassword() string {
	const alphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	raw := randomBytes(16)
	out := make([]byte, len(raw))
	for i, b := range raw {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(out)
}

func randomHex(n int) string { return hex.EncodeToString(randomBytes(n)) }

func toSlash(path string) string { return filepath.ToSlash(path) }

func readEnvFile(path string) map[string]string {
	values := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return values
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return values
}

// 首次运行生成配置，返回 (是否新建, 管理员账号, 管理员密码)
func ensureConfig(dataDir string) (bool, string, string) {
	configPath := filepath.Join(dataDir, configFileName)
	if _, err := os.Stat(configPath); err == nil {
		values := readEnvFile(configPath)
		pwd := values["ADMIN_PASSWORD"]
		user := values["ADMIN_USERNAME"]
		if user == "" {
			user = adminUserName
		}
		return false, user, pwd
	}

	password := randomPassword()
	content := strings.Join([]string{
		"# 无限画布运行配置（首次启动自动生成，可自行修改后重启生效）",
		"ADMIN_USERNAME=" + adminUserName,
		"ADMIN_PASSWORD=" + password,
		"JWT_SECRET=" + randomHex(32),
		"JWT_EXPIRE_HOURS=168",
		"STORAGE_DRIVER=sqlite",
		"DATABASE_DSN=" + toSlash(filepath.Join(dataDir, "infinite-canvas.db")),
		"AI_LOG_DIR=" + toSlash(filepath.Join(dataDir, "logs", "ai-calls")),
		"PROMPT_DATA_DIR=" + toSlash(filepath.Join(dataDir, "prompts")),
		"",
		"# 网页端口，默认 3000；被占用时会自动往后顺延",
		"WEB_PORT=" + strconv.Itoa(defaultWebPort),
		"# 只允许本机访问填 127.0.0.1，局域网访问改成 0.0.0.0",
		"WEB_HOST=127.0.0.1",
		"# 后端内部端口，默认 8080",
		"API_PORT=" + strconv.Itoa(defaultAPIPort),
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		msgBox("无限画布", "写入配置文件失败：\n"+configPath+"\n"+err.Error(), msgBoxOK|msgBoxIconError)
		os.Exit(1)
	}
	os.WriteFile(filepath.Join(dataDir, credentialsName),
		[]byte("管理后台账号："+adminUserName+"\n管理后台密码："+password+"\n\n登录后可到「后台 - 用户管理」修改密码。\n"),
		0o644)
	return true, adminUserName, password
}

// 端口占用检测必须按组件实际绑定的地址试听：
// 后端监听 :port（所有网卡），前端默认监听 127.0.0.1:port，
// 否则会出现“检测通过、真正启动时端口被占”的情况。
func portFree(port int, allInterfaces bool) bool {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	if allInterfaces {
		address = fmt.Sprintf(":%d", port)
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

func freePort(preferred int, allInterfaces bool) int {
	for port := preferred; port < preferred+40; port++ {
		if portFree(port, allInterfaces) {
			return port
		}
	}
	return 0
}

// 只认自己的健康接口，避免把占用同端口的其他程序误判成本程序
func appAlive(port int) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/api/health", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
	return resp.StatusCode == http.StatusOK && strings.TrimSpace(string(body)) == "ok"
}

func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func spawn(dir, exe string, args []string, extraEnv []string, logPath string) (*exec.Cmd, error) {
	handle, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = handle
	cmd.Stderr = handle
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow | detachedProcess}
	if err := cmd.Start(); err != nil {
		handle.Close()
		return nil, err
	}
	go func() {
		cmd.Wait()
		handle.Close()
	}()
	return cmd, nil
}

func writePIDs(dataDir string, apiPID, webPID int) {
	os.WriteFile(filepath.Join(dataDir, pidFileName), []byte(fmt.Sprintf("%d\n%d\n", apiPID, webPID)), 0o644)
}

func readPIDs(dataDir string) []string {
	data, err := os.ReadFile(filepath.Join(dataDir, pidFileName))
	if err != nil {
		return nil
	}
	var pids []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line != "" && line != "0" {
			pids = append(pids, line)
		}
	}
	return pids
}

// 记录本次实际使用的网页端口，配置端口被占用时会顺延，供状态查询和停止使用
func writePort(dataDir string, port int) {
	os.WriteFile(filepath.Join(dataDir, portFileName), []byte(strconv.Itoa(port)), 0o644)
}

func readPort(dataDir string) int {
	data, err := os.ReadFile(filepath.Join(dataDir, portFileName))
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return port
}

// 结束所有由本安装目录启动的进程（含仍停在提示框上的启动器），供升级/卸载清理
func killByPath(appDir string) {
	pattern := strings.ReplaceAll(appDir, "'", "''") + "\\*"
	script := fmt.Sprintf(
		"Get-Process -ErrorAction SilentlyContinue | Where-Object { $_.Path -like '%s' -and $_.Id -ne %d } | Stop-Process -Force -ErrorAction SilentlyContinue",
		pattern, os.Getpid())
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
	cmd.Run()
}

func killPIDs(pids []string) int {
	killed := 0
	for _, pid := range pids {
		cmd := exec.Command("taskkill", "/PID", pid, "/T", "/F")
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
		if err := cmd.Run(); err == nil {
			killed++
		}
	}
	return killed
}

func listenerPIDs(port int) []string {
	cmd := exec.Command("netstat", "-ano", "-p", "TCP")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var pids []string
	seen := map[string]bool{}
	suffix := fmt.Sprintf(":%d ", port)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[0] != "TCP" {
			continue
		}
		if !strings.HasSuffix(fields[1], suffix) && fields[1] != fmt.Sprintf("0.0.0.0:%d", port) {
			continue
		}
		if strings.EqualFold(fields[3], "LISTENING") && !seen[fields[4]] {
			seen[fields[4]] = true
			pids = append(pids, fields[4])
		}
	}
	return pids
}

func configInt(values map[string]string, key string, fallback int) int {
	if raw, ok := values[key]; ok {
		if parsed, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func doStart(appDir, dataDir string) {
	created, user, password := ensureConfig(dataDir)
	values := readEnvFile(filepath.Join(dataDir, configFileName))
	webHost := strings.TrimSpace(values["WEB_HOST"])
	if webHost == "" {
		webHost = "127.0.0.1"
	}
	preferredWeb := configInt(values, "WEB_PORT", defaultWebPort)
	preferredAPI := configInt(values, "API_PORT", defaultAPIPort)

	if appAlive(preferredWeb) {
		logf("服务已在运行，直接打开页面")
		openBrowser(fmt.Sprintf("http://localhost:%d", preferredWeb))
		return
	}

	serverExe := filepath.Join(appDir, "server.exe")
	nodeExe := filepath.Join(appDir, "node", "node.exe")
	webDir := filepath.Join(appDir, "web")
	for _, path := range []string{serverExe, nodeExe, filepath.Join(webDir, "server.js")} {
		if _, err := os.Stat(path); err != nil {
			msgBox("无限画布", "安装文件缺失：\n"+path+"\n请重新安装。", msgBoxOK|msgBoxIconError)
			return
		}
	}

	logDir := filepath.Join(dataDir, "logs")
	ensureDir(logDir)
	ensureDir(filepath.Join(dataDir, "prompts"))

	apiPort := freePort(preferredAPI, true)
	webPort := freePort(preferredWeb, webHost == "0.0.0.0")
	if apiPort == 0 || webPort == 0 {
		msgBox("无限画布", "找不到可用端口，请关闭其他程序后重试。", msgBoxOK|msgBoxIconError)
		return
	}

	logf("启动后端：%s (端口 %d)", serverExe, apiPort)
	apiCmd, err := spawn(dataDir, serverExe, nil, []string{
		"PORT=" + strconv.Itoa(apiPort),
		"DATABASE_DSN=" + toSlash(filepath.Join(dataDir, "infinite-canvas.db")),
	}, filepath.Join(logDir, "server.log"))
	if err != nil {
		msgBox("无限画布", "后端启动失败：\n"+err.Error(), msgBoxOK|msgBoxIconError)
		return
	}

	// 确认后端真的起来了，避免端口被别的程序占用时页面把接口代理到别的服务
	apiReady := false
	for i := 0; i < 30; i++ {
		if appAlive(apiPort) {
			apiReady = true
			break
		}
		time.Sleep(time.Second)
	}
	if !apiReady {
		logf("后端未就绪，端口 %d 可能被占用", apiPort)
		killPIDs([]string{strconv.Itoa(apiCmd.Process.Pid)})
		msgBox("无限画布", fmt.Sprintf("后端启动失败，端口 %d 可能被其他程序占用。\n\n请查看日志确认：\n%s", apiPort, filepath.Join(logDir, "server.log")), msgBoxOK|msgBoxIconError)
		return
	}

	logf("启动网页服务：%s (端口 %d)", nodeExe, webPort)
	webCmd, err := spawn(webDir, nodeExe, []string{"server.js"}, []string{
		"NODE_ENV=production",
		"NEXT_TELEMETRY_DISABLED=1",
		"PORT=" + strconv.Itoa(webPort),
		"HOSTNAME=" + webHost,
		"API_BASE_URL=http://127.0.0.1:" + strconv.Itoa(apiPort),
	}, filepath.Join(logDir, "web.log"))
	if err != nil {
		killPIDs([]string{strconv.Itoa(apiCmd.Process.Pid)})
		msgBox("无限画布", "网页服务启动失败：\n"+err.Error(), msgBoxOK|msgBoxIconError)
		return
	}

	writePIDs(dataDir, apiCmd.Process.Pid, webCmd.Process.Pid)
	writePort(dataDir, webPort)
	url := fmt.Sprintf("http://localhost:%d", webPort)

	for i := 0; i < readyTimeoutSec; i++ {
		if appAlive(webPort) {
			logf("服务已就绪：%s", url)
			openBrowser(url)
			if created {
				msgBox("无限画布已启动",
					fmt.Sprintf("网页地址：%s\n管理账号：%s\n管理密码：%s\n\n账号密码已保存到：\n%s",
						url, user, password, filepath.Join(dataDir, credentialsName)),
					msgBoxOK|msgBoxIconInfo)
			}
			return
		}
		time.Sleep(time.Second)
	}

	logf("等待超时，请查看日志：%s", filepath.Join(logDir, "web.log"))
	msgBox("无限画布",
		fmt.Sprintf("启动超时，服务可能仍在初始化。\n请稍后手动打开：%s\n\n日志目录：\n%s", url, logDir),
		msgBoxOK|msgBoxIconError)
}

func doStop(dataDir string, silent bool) {
	values := readEnvFile(filepath.Join(dataDir, configFileName))
	ports := []int{
		configInt(values, "WEB_PORT", defaultWebPort),
		configInt(values, "API_PORT", defaultAPIPort),
	}
	// 本次实际使用的网页端口（配置端口被占用时会顺延）
	if port := readPort(dataDir); port > 0 {
		ports = append(ports, port)
	}

	pids := readPIDs(dataDir)
	for _, port := range ports {
		pids = append(pids, listenerPIDs(port)...)
	}
	unique := map[string]bool{}
	var ordered []string
	for _, pid := range pids {
		if pid != "" && !unique[pid] {
			unique[pid] = true
			ordered = append(ordered, pid)
		}
	}

	killed := killPIDs(ordered)
	os.Remove(filepath.Join(dataDir, pidFileName))
	os.Remove(filepath.Join(dataDir, portFileName))
	logf("已停止 %d 个进程", killed)
	if !silent {
		msgBox("无限画布", fmt.Sprintf("已停止服务（结束 %d 个进程）。\n数据保留在：\n%s", killed, dataDir), msgBoxOK|msgBoxIconInfo)
	}
}

func doStatus(dataDir string) {
	values := readEnvFile(filepath.Join(dataDir, configFileName))
	webPort := configInt(values, "WEB_PORT", defaultWebPort)
	if port := readPort(dataDir); port > 0 && appAlive(port) {
		msgBox("无限画布", fmt.Sprintf("服务正在运行。\n网页地址：http://localhost:%d", port), msgBoxOK|msgBoxIconInfo)
		return
	}
	// 配置端口被别的程序占用时，服务会自动往后顺延，这里一并探测
	for port := webPort; port < webPort+40; port++ {
		if appAlive(port) {
			msgBox("无限画布", fmt.Sprintf("服务正在运行。\n网页地址：http://localhost:%d", port), msgBoxOK|msgBoxIconInfo)
			return
		}
	}
	msgBox("无限画布", "服务未在运行。", msgBoxOK|msgBoxIconInfo)
}

func doLogs(dataDir string) {
	logDir := filepath.Join(dataDir, "logs")
	ensureDir(logDir)
	exec.Command("explorer", logDir).Start()
}

func main() {
	appDir := exeDir()
	if err := ensureDir(appDir); err != nil {
		os.Exit(1)
	}
	dataDir := resolveDataDir(appDir)
	ensureDir(dataDir)
	if handle, err := os.OpenFile(filepath.Join(dataDir, "launcher.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		logFile = handle
		defer handle.Close()
	}
	logf("启动器工作目录：%s", appDir)
	logf("数据目录：%s", dataDir)

	command := "start"
	if len(os.Args) > 1 {
		command = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(os.Args[1]), "-"))
	}

	switch command {
	case "stop":
		doStop(dataDir, false)
	case "stopsilent":
		doStop(dataDir, true)
	case "killall":
		// 升级/卸载前调用：停服务并清掉仍驻留的启动器进程，避免文件被占用
		doStop(dataDir, true)
		killByPath(appDir)
	case "status":
		doStatus(dataDir)
	case "logs":
		doLogs(dataDir)
	default:
		doStart(appDir, dataDir)
	}
}
