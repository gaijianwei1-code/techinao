# 编译 NSIS 安装程序。
# 说明：安装脚本源码在项目里（tools\packaging），载荷与 NSIS 工具链放在构建工作区。
# Pure ASCII source: Chinese names are built from code points so the script is not
# affected by PowerShell 5.1 reading BOM-less UTF-8 files as ANSI.
$ErrorActionPreference = 'Stop'
$here = $PSScriptRoot                                      # tools\packaging
$project = Split-Path (Split-Path $here -Parent) -Parent   # 项目根目录
$build = 'c:\Users\11\.trae-cn\work\6aaa7312573ee181ecf93750\ic-build'

$makensis = Join-Path $build 'tools\nsis\nsis-3.11\makensis.exe'
$nsi = Join-Path $here 'installer.nsi'
$version = (Get-Content (Join-Path $project 'VERSION') -Raw).Trim().TrimStart('v')

Write-Output '== 生成图标 =='
node (Join-Path $build 'icon\make-icons.mjs') (Join-Path $project 'web\public\logo.svg') (Join-Path $build 'payload\logo.ico') (Join-Path $build 'payload\installer.ico')
Copy-Item (Join-Path $build 'payload\logo.ico') (Join-Path $build 'payload\stage\logo.ico') -Force

Write-Output '== 修正 .nsi 编码（UTF-8 BOM）=='
$text = [System.IO.File]::ReadAllText($nsi, [System.Text.Encoding]::UTF8)
[System.IO.File]::WriteAllText($nsi, $text, (New-Object System.Text.UTF8Encoding $true))

Write-Output "== 编译安装程序（版本 $version）=="
New-Item -ItemType Directory -Force -Path (Join-Path $build 'dist') | Out-Null
& $makensis "/DAPP_VERSION=$version" "/DBUILD_ROOT=$build" $nsi
if ($LASTEXITCODE -ne 0) { throw "makensis 编译失败，退出码 $LASTEXITCODE" }

Write-Output '== 产物 =='
Get-ChildItem (Join-Path $build 'dist') | Select-Object Name, @{ Name = 'MB'; Expression = { [math]::Round($_.Length / 1MB, 1) } }
