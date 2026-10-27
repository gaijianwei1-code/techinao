@echo off
chcp 65001 >nul
title 无限画布 启动器
setlocal
set "DISTRO=Ubuntu-22.04"
set "APPDIR=/mnt/d/.Hermes/infinite-canvas"
set "PORT=3030"
set "ARGS=%*"
set "URL=http://localhost:%PORT%"

echo.
echo  ==================================================
echo    无限画布 / WSL 启动器
echo    start ^| stop ^| restart ^| rebuild ^| logs ^| status ^| update
echo    不带参数 = start
echo  ==================================================
echo.

wsl.exe -d %DISTRO% -e bash -lc "cd %APPDIR% && ./start.sh %*"
if errorlevel 1 goto failed

if not "%ARGS%"=="" goto done

call :keepalive
call :waitready
if errorlevel 1 goto notready

start "" %URL%
echo.
echo  [i] 已在浏览器打开 %URL%
goto done

:keepalive
rem 留一个隐藏的 WSL 会话，避免 WSL 空闲休眠导致服务掉线
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0keepalive.ps1"
exit /b 0

:waitready
rem 等到后端真正就绪（/api/health 返回 ok）再打开浏览器，最多 180 秒
set /a TRY=0
:waitloop
set /a TRY+=1
set "HEALTH="
for /f "usebackq delims=" %%b in (`curl.exe -s -m 4 "%URL%/api/health" 2^>nul`) do set "HEALTH=%%b"
if "%HEALTH%"=="ok" (
  echo  [i] 后端已就绪（第 %TRY% 次探测）
  exit /b 0
)
if %TRY% GEQ 60 (
  echo  [x] 等满 180 秒后端仍未就绪。
  exit /b 1
)
echo  [.] 等待后端就绪（%TRY%/60）...
timeout /t 3 >nul
goto waitloop

:notready
echo.
echo  [i] 服务还在启动中，稍等几十秒后手动打开：%URL%
echo      或再点一次「无限画布-打开页面」。
echo  [i] 查看日志：wsl -d %DISTRO% -e bash -lc "cd %APPDIR% && ./start.sh logs"
goto done

:done
echo.
echo  [i] 关闭本窗口不会停止服务（要停止请点「无限画布-停止」）。
echo  [i] 局域网访问：http://本机IP:%PORT%（需在防火墙放行该端口）。
echo.
echo  处理完毕，按任意键关闭本窗口...
pause >nul
exit /b 0

:failed
echo.
echo  [x] 启动失败。排查命令：
echo      wsl -d %DISTRO% -e bash -lc "cd %APPDIR% && ./start.sh logs"
echo.
echo  按任意键关闭本窗口...
pause >nul
exit /b 1
