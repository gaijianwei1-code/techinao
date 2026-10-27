@echo off
chcp 936 >nul
title 无限画布 - 一键启动全部服务（含内网穿透）
setlocal
set "DISTRO=Ubuntu-22.04"
set "APPDIR=/mnt/d/.Hermes/infinite-canvas"
set "PORT=3030"
set "LOCALURL=http://localhost:%PORT%"
set "URLFILE=%~dp0.run\tunnel-url.txt"

echo.
echo  ==================================================
echo    无限画布 / 一键启动全部服务
echo    1. 启动应用（Docker）    2. WSL 保活
echo    3. 等待后端就绪          4. 内网穿透
echo  ==================================================
echo.

rem ---------- 1/4 启动应用 ----------
echo  [1/4] 启动应用...
echo        首次运行需要构建镜像（约 3~10 分钟），期间请勿关闭本窗口。
echo.
wsl.exe -d %DISTRO% -e bash -lc "cd %APPDIR% && ./start.sh start"
if errorlevel 1 goto appfailed

rem ---------- 2/4 WSL 保活 ----------
echo.
echo  [2/4] 保持 WSL 会话存活（避免 WSL 空闲休眠导致服务掉线）...
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0keepalive.ps1"

rem ---------- 3/4 等后端就绪 ----------
echo.
echo  [3/4] 等待后端就绪...
call :waitready
if errorlevel 1 goto notready

rem ---------- 4/4 内网穿透 ----------
echo.
echo  [4/4] 启动内网穿透（cloudflared 临时域名）...
wsl.exe -d %DISTRO% -e bash -lc "cd %APPDIR% && ./scripts/tunnel/tunnel.sh cloudflared"
if errorlevel 1 goto tunnelfailed
if not exist "%URLFILE%" goto tunnelfailed

set "PUBURL="
set /p PUBURL=<"%URLFILE%"
if "%PUBURL%"=="" goto tunnelfailed

echo|set /p=%PUBURL%|clip

echo.
echo  --------------------------------------------------
echo   本机地址：%LOCALURL%
echo   对外地址：%PUBURL%
echo  --------------------------------------------------
echo   [i] 对外地址已复制到剪贴板，直接发给朋友即可。
echo   [i] 临时域名：重启或重跑本脚本都会换地址，记得重新发一次。
echo.

echo  正在用本机验证公网地址（新域名本机 DNS 可能还没同步）...
set "PHEALTH="
for /f "usebackq delims=" %%b in (`curl.exe -s -m 25 "%PUBURL%/api/health" 2^>nul`) do set "PHEALTH=%%b"
if not "%PHEALTH%"=="ok" goto publicslow

echo   [i] 公网自测通过，朋友那边可以直接打开。
start "" "%PUBURL%"
echo   [i] 已在本机浏览器打开对外地址。
goto done

:publicslow
echo   [注意] 本机打不开对外地址，这多半是本机 DNS 把新域名缓存成了不存在，属于正常现象。
echo          朋友那边能打开；本机想打开就把 DNS 换成 119.29.29.29。
start "" "%LOCALURL%"
echo   [i] 已在本机浏览器打开 %LOCALURL%。
goto done

:waitready
rem 等到后端真正就绪（/api/health 返回 ok）再起隧道，否则朋友打开是 502，最多 180 秒
set /a TRY=0
:waitloop
set /a TRY+=1
set "HEALTH="
for /f "usebackq delims=" %%b in (`curl.exe -s -m 4 "%LOCALURL%/api/health" 2^>nul`) do set "HEALTH=%%b"
if "%HEALTH%"=="ok" (
  echo   [i] 后端已就绪（第 %TRY% 次探测）
  exit /b 0
)
if %TRY% GEQ 60 (
  echo   [x] 等满 180 秒后端仍未就绪。
  exit /b 1
)
echo   [.] 等待后端就绪（%TRY%/60）...
timeout /t 3 >nul
goto waitloop

:appfailed
echo.
echo  [x] 应用启动失败，已跳过内网穿透。
echo      排查：wsl -d %DISTRO% -e bash -lc "cd %APPDIR% && ./start.sh logs"
goto fail

:notready
echo.
echo  [x] 后端 180 秒内没就绪，已跳过内网穿透（否则朋友打开会看到 502）。
echo      排查：wsl -d %DISTRO% -e bash -lc "cd %APPDIR% && ./start.sh logs"
echo      等确认能打开 %LOCALURL% 后，再双击「无限画布-开启外网访问」单独起隧道。
goto fail

:tunnelfailed
echo.
echo  [x] 应用已就绪，但没拿到对外地址。
echo      日志：%~dp0logs\tunnel-cloudflared.log
echo      常见原因：WSL 里没装 cloudflared、连不上 Cloudflare 边缘、WSL 休眠。
echo      本机访问不受影响：%LOCALURL%
goto fail

:done
echo.
echo  --------------------------------------------------
echo   停止全部：双击「无限画布-停止」（应用 + 保活）
echo   只关隧道：双击「无限画布-关闭外网访问」
echo  --------------------------------------------------
echo   [i] 关闭本窗口不会停止任何服务。
echo   [i] 首次给朋友用之前，记得去「后台 - 用户管理」给他建一个托管账号并设置使用次数，
echo       建议同时在「后台 - 系统设置」关掉「是否允许用户注册」。
echo.
echo  处理完毕，按任意键关闭本窗口...
pause >nul
exit /b 0

:fail
echo.
echo  按任意键关闭本窗口...
pause >nul
exit /b 1
