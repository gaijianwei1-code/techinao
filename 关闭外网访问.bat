@echo off
chcp 65001 >nul
title 无限画布 - 关闭外网访问
setlocal
set "DISTRO=Ubuntu-22.04"
set "APPDIR=/mnt/d/.Hermes/infinite-canvas"

echo.
echo  正在关闭 cloudflared 隧道...
wsl.exe -d %DISTRO% -e bash -lc "cd %APPDIR% && ./scripts/tunnel/tunnel.sh stop"

echo.
echo  [i] 隧道关闭后对外地址立即失效，应用本身不受影响。
echo      想再对外，双击「开启外网访问.bat」会拿到一个新地址。
echo.
echo  按任意键关闭本窗口...
pause >nul
exit /b 0
