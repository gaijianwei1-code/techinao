@echo off
chcp 65001 >nul
title 无限画布 停止
setlocal
set "DISTRO=Ubuntu-22.04"
set "APPDIR=/mnt/d/.Hermes/infinite-canvas"

echo.
wsl.exe -d %DISTRO% -e bash -lc "cd %APPDIR% && ./start.sh stop"
set "RC=%ERRORLEVEL%"

rem 释放启动时留下的保活 WSL 会话
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0keepalive.ps1" -Stop

if not "%RC%"=="0" goto failed

echo.
echo  [i] 已停止服务（数据保留在项目的 data 目录）。
echo.
echo  按任意键关闭本窗口...
pause >nul
exit /b 0

:failed
echo.
echo  [x] 停止失败。排查命令：
echo      wsl -d %DISTRO% -e bash -lc "cd %APPDIR% && ./start.sh stop"
echo.
echo  按任意键关闭本窗口...
pause >nul
exit /b 1
