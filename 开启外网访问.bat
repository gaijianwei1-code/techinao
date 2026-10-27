@echo off
chcp 65001 >nul
title 无限画布 - 开启外网访问
setlocal
set "DISTRO=Ubuntu-22.04"
set "APPDIR=/mnt/d/.Hermes/infinite-canvas"
set "PORT=3030"
set "URLFILE=D:\.Hermes\infinite-canvas\.run\tunnel-url.txt"

echo.
echo  ==================================================
echo    无限画布 / 开启外网访问（cloudflared 临时地址）
echo  ==================================================
echo.

rem 应用没起也允许起隧道，但先提示一句，避免朋友打开是 502
curl.exe -s -m 5 http://localhost:%PORT%/api/health 2>nul | findstr /x /c:"ok" >nul
if errorlevel 1 (
  echo  [注意] 本机 %PORT% 端口还没就绪，隧道会一直等它起来。
  echo         建议先点「无限画布-一键启动」，等页面能打开再继续。
  echo.
)

echo  正在启动隧道（首次约 10~30 秒）...
wsl.exe -d %DISTRO% -e bash -lc "cd %APPDIR% && ./scripts/tunnel/tunnel.sh cloudflared"
if errorlevel 1 goto failed

if not exist "%URLFILE%" goto failed
set "URL="
set /p URL=<"%URLFILE%"
if "%URL%"=="" goto failed

echo.
echo  --------------------------------------------------
echo   对外地址：%URL%
echo  --------------------------------------------------
<nul set /p="%URL%"| clip
echo  [i] 地址已复制到剪贴板，发给外地朋友即可。
echo  [i] 临时域名：重启或重跑本脚本都会换地址，记得重新发一次。
echo  [i] 关闭：双击「关闭外网访问.bat」。
echo.
echo  用本机验证一下公网地址（本机 DNS 可能还没同步到这个新域名）...
curl.exe -s -m 20 -o nul -w "  /api/health -> HTTP %%{http_code}\n" "%URL%/api/health"
curl.exe -s -m 20 http://localhost:%PORT%/api/health >nul 2>&1
if errorlevel 1 (
  echo  [注意] 本机 %PORT% 端口没响应，朋友打开会看到 502：先点「无限画布-一键启动」。
) else (
  curl.exe -s -m 20 -o nul "%URL%/api/health" 2>nul
  if errorlevel 1 (
    echo  [i] 本机打不开属于正常（本地 DNS 缓存了新域名），朋友那边能打开；本机想打开就把 DNS 换成 119.29.29.29。
  ) else (
    start "" "%URL%"
  )
)
goto done

:failed
echo.
echo  [x] 没拿到对外地址。
echo      日志：D:\.Hermes\infinite-canvas\logs\tunnel-cloudflared.log
echo      常见原因：cloudflared 缺失、连不上 Cloudflare 边缘、应用没启动。
echo.
echo  按任意键关闭本窗口...
pause >nul
exit /b 1

:done
echo  按任意键关闭本窗口...
pause >nul
exit /b 0
