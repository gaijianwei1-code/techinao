# Keep the WSL VM alive while the app is supposed to be running.
# Without this, WSL shuts down when idle and the container (and the web app) stops.
param(
    [switch]$Stop
)

$distro = 'Ubuntu-22.04'
$marker = 'ic-keepalive'
$pidFile = Join-Path $env:TEMP 'infinite-canvas-keepalive.pid'

function Get-KeepAliveProcess {
    if (-not (Test-Path $pidFile)) { return $null }
    $raw = Get-Content $pidFile -Raw -ErrorAction SilentlyContinue
    if (-not $raw -or -not $raw.Trim()) { return $null }
    $proc = Get-Process -Id ([int]$raw.Trim()) -ErrorAction SilentlyContinue
    if ($proc -and $proc.ProcessName -eq 'wsl') { return $proc }
    return $null
}

if ($Stop) {
    $targets = @()
    $proc = Get-KeepAliveProcess
    if ($proc) { $targets += $proc.Id }
    # wsl.exe spawns a child process, so sweep by command-line marker as well
    Get-CimInstance Win32_Process -Filter "Name='wsl.exe'" -ErrorAction SilentlyContinue |
        Where-Object { $_.CommandLine -like "*$marker*" } |
        ForEach-Object { $targets += $_.ProcessId }
    $targets = @($targets | Select-Object -Unique)
    foreach ($procId in $targets) {
        Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
    }
    Remove-Item $pidFile -Force -ErrorAction SilentlyContinue
    Write-Output ("[keepalive] released " + $targets.Count + " session(s)")
    exit 0
}

$existing = Get-KeepAliveProcess
if ($existing) {
    Write-Output ("[keepalive] already running pid " + $existing.Id)
    exit 0
}

# PowerShell 5.1 does not quote -ArgumentList items, so the command string must
# carry its own quotes; otherwise wsl splits it into several arguments and the
# session exits right away.
$bashCommand = "exec sleep 2147483  # $marker"
try {
    $p = Start-Process -WindowStyle Hidden -FilePath 'wsl.exe' `
        -ArgumentList @('-d', $distro, '-e', 'bash', '-lc', ('"' + $bashCommand + '"')) `
        -PassThru -ErrorAction Stop
} catch {
    Write-Output ("[keepalive] start failed: " + $_.Exception.Message)
    exit 1
}

Start-Sleep -Seconds 2
$p.Refresh()
if ($p.HasExited) {
    Write-Output '[keepalive] session exited immediately'
    Remove-Item $pidFile -Force -ErrorAction SilentlyContinue
    exit 1
}

Set-Content -Path $pidFile -Value $p.Id -Encoding ASCII
Write-Output ("[keepalive] session started pid " + $p.Id)
exit 0
