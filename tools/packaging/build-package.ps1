# 组装安装包：把环境检测工具放进载荷 -> 编译 NSIS 安装程序 -> 复制产物到项目 dist。
# Pure ASCII source: Chinese names are built from code points so the script is not
# affected by PowerShell 5.1 reading BOM-less UTF-8 files as ANSI.
$ErrorActionPreference = 'Stop'
$here = $PSScriptRoot                                      # tools\packaging
$project = Split-Path (Split-Path $here -Parent) -Parent   # 项目根目录
$build = 'c:\Users\11\.trae-cn\work\6aaa7312573ee181ecf93750\ic-build'

# 环境检测工具.exe
$toolName = [string]::Join('', [char]0x73AF, [char]0x5883, [char]0x68C0, [char]0x6D4B, [char]0x5DE5, [char]0x5177) + '.exe'
# 我自己的AI工作台_Setup_v<版本>_x64.exe
$version = (Get-Content (Join-Path $project 'VERSION') -Raw).Trim().TrimStart('v')
$setupName = [string]::Join('', [char]0x6211, [char]0x81EA, [char]0x5DF1, [char]0x7684) + 'AI' +
    [string]::Join('', [char]0x5DE5, [char]0x4F5C, [char]0x53F0) + "_Setup_v${version}_x64.exe"

Write-Output '== 1. 环境检测工具放进载荷 =='
# 用 32 位版：32 位程序在 32/64 位 Windows 上都能运行，Win7 老机器也能跑
Copy-Item (Join-Path $build 'dist\ic-env-check-x86.exe') (Join-Path $build ('payload\stage\' + $toolName)) -Force
Get-ChildItem (Join-Path $build 'payload\stage') | Select-Object Name, @{ n = 'MB'; e = { [math]::Round($_.Length / 1MB, 2) } }

Write-Output '== 2. 编译安装程序 =='
& powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $here 'build-installer.ps1') | Select-Object -Last 4

Write-Output '== 3. 复制产物到项目 dist =='
$out = Join-Path $project 'dist'
New-Item -ItemType Directory -Force -Path $out | Out-Null
Copy-Item (Join-Path $build ('dist\' + $setupName)) (Join-Path $out $setupName) -Force
Copy-Item (Join-Path $build 'dist\ic-env-check-x86.exe') (Join-Path $out $toolName) -Force
Get-ChildItem $out | Select-Object Name, @{ n = 'MB'; e = { [math]::Round($_.Length / 1MB, 2) } }, LastWriteTime
Write-Output '== sha256 =='
(Get-FileHash (Join-Path $out $setupName) -Algorithm SHA256).Hash
(Get-FileHash (Join-Path $out $toolName) -Algorithm SHA256).Hash
