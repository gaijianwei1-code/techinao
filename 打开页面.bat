@echo off
chcp 65001 >nul
title 无限画布 - 打开页面
setlocal
set "DISTRO=Ubuntu-22.04"
set "PORT=3030"
set "WIP="

for /f "usebackq delims=" %%i in (`wsl.exe -d %DISTRO% -e bash -lc "hostname -I | awk '{print $1}'"`) do set "WIP=%%i"

rem 后端就绪（/api/health 返回 ok）才打开，避免页面提示“接口连接失败”
curl.exe -s -m 5 http://localhost:%PORT%/api/health 2>nul | findstr /x /c:"ok" >nul
if not errorlevel 1 goto openlocal

if "%WIP%"=="" goto nolink
curl.exe -s -m 5 http://%WIP%:%PORT%/api/health 2>nul | findstr /x /c:"ok" >nul
if not errorlevel 1 goto openip

echo.
echo  [x] 服务还没就绪（后端未响应）。
echo      先点「无限画布-一键启动」；如果刚启动，等 30~60 秒再试。
echo.
echo  按任意键关闭本窗口...
pause >nul
exit /b 1

:openlocal
start "" http://localhost:%PORT%
echo.
echo  [i] 已打开 http://localhost:%PORT%
goto done

:openip
start "" http://%WIP%:%PORT%
echo.
echo  [i] 已打开 http://%WIP%:%PORT%   （localhost 未被 WSL 转发，改用 WSL 地址）
goto done

:nolink
echo.
echo  [x] 连不上服务，也没拿到 WSL 地址。先点「无限画布-一键启动」。
echo.
echo  按任意键关闭本窗口...
pause >nul
exit /b 1

:done
echo.
echo  按任意键关闭本窗口...
pause >nul
exit /b 0
