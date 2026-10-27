; 无限画布 · 我自己的 AI 工作台 —— Windows 一键安装程序
; 通过 makensis 编译，详见 scripts/build-installer.ps1

!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"

Unicode true

!ifndef APP_VERSION
  !define APP_VERSION "0.7.1"
!endif
!ifndef BUILD_ROOT
  !define BUILD_ROOT ".."
!endif

!define APP_NAME "我自己的 AI 工作台"
!define APP_ID "InfiniteCanvasWorkbench"
!define APP_PUBLISHER "infinite-canvas"
!define APP_HOMEPAGE "https://github.com/tigerowo/infinite-canvas"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}"

Name "${APP_NAME}"
OutFile "${BUILD_ROOT}\dist\我自己的AI工作台_Setup_v${APP_VERSION}_x64.exe"
InstallDir "$LOCALAPPDATA\Programs\InfiniteCanvas"
InstallDirRegKey HKCU "${UNINST_KEY}" "InstallLocation"
RequestExecutionLevel user
SetCompressor /SOLID lzma
ShowInstDetails show
ShowUninstDetails show

VIProductVersion "${APP_VERSION}.0"
VIAddVersionKey "ProductName" "${APP_NAME}"
VIAddVersionKey "FileDescription" "${APP_NAME} 安装程序"
VIAddVersionKey "FileVersion" "${APP_VERSION}"
VIAddVersionKey "ProductVersion" "${APP_VERSION}"
VIAddVersionKey "LegalCopyright" "AGPL-3.0"

!define MUI_ICON "${BUILD_ROOT}\payload\installer.ico"
!define MUI_UNICON "${BUILD_ROOT}\payload\installer.ico"
!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TITLE "欢迎安装 ${APP_NAME}"
!define MUI_WELCOMEPAGE_TEXT "本向导将把完整的无限画布工作台安装到你的电脑。$\r$\n$\r$\n安装包内置后端服务、前端页面和 Node 运行环境，无需另外安装 Docker、Node.js 或其他依赖，安装后即可在本机浏览器中使用。$\r$\n$\r$\n点击「下一步」继续。"
!define MUI_FINISHPAGE_TITLE "安装完成"
!define MUI_FINISHPAGE_TEXT "${APP_NAME} 已安装完成。$\r$\n$\r$\n首次启动会自动生成配置文件并初始化数据库，随后弹出窗口显示网页地址与管理员账号密码，浏览器会自动打开页面。$\r$\n$\r$\n数据与日志都保存在安装目录的 data 文件夹中。"
!define MUI_FINISHPAGE_RUN "$INSTDIR\launcher.exe"
!define MUI_FINISHPAGE_RUN_TEXT "立即启动 ${APP_NAME}"
!define MUI_FINISHPAGE_LINK "查看使用说明"
!define MUI_FINISHPAGE_LINK_LOCATION "$INSTDIR\使用说明.txt"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "${BUILD_ROOT}\payload\stage\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "SimpChinese"

Function .onInit
  ; 升级安装：先停掉正在运行的服务与驻留的启动器，避免文件被占用
  IfFileExists "$INSTDIR\launcher.exe" 0 done
  DetailPrint "正在停止已运行的无限画布服务…"
  ExecWait '"$INSTDIR\launcher.exe" killall'
  Sleep 2000
done:
FunctionEnd

Section "主程序" SecMain
  SetOutPath "$INSTDIR"
  SetOverwrite on
  File /r "${BUILD_ROOT}\payload\stage\*.*"

  CreateDirectory "$INSTDIR\data"

  ; 开始菜单快捷方式
  CreateDirectory "$SMPROGRAMS\${APP_NAME}"
  CreateShortCut "$SMPROGRAMS\${APP_NAME}\启动 ${APP_NAME}.lnk" "$INSTDIR\launcher.exe" "" "$INSTDIR\logo.ico" 0
  CreateShortCut "$SMPROGRAMS\${APP_NAME}\停止服务.lnk" "$INSTDIR\launcher.exe" "stop" "$INSTDIR\logo.ico" 0
  CreateShortCut "$SMPROGRAMS\${APP_NAME}\查看运行状态.lnk" "$INSTDIR\launcher.exe" "status" "$INSTDIR\logo.ico" 0
  CreateShortCut "$SMPROGRAMS\${APP_NAME}\查看日志.lnk" "$INSTDIR\launcher.exe" "logs" "$INSTDIR\logo.ico" 0
  CreateShortCut "$SMPROGRAMS\${APP_NAME}\环境检测.lnk" "$INSTDIR\环境检测工具.exe" "" "$INSTDIR\环境检测工具.exe" 0
  CreateShortCut "$SMPROGRAMS\${APP_NAME}\使用说明.lnk" "$INSTDIR\使用说明.txt"
  CreateShortCut "$SMPROGRAMS\${APP_NAME}\卸载 ${APP_NAME}.lnk" "$INSTDIR\uninstall.exe"

  ; 桌面快捷方式
  CreateShortCut "$DESKTOP\${APP_NAME}.lnk" "$INSTDIR\launcher.exe" "" "$INSTDIR\logo.ico" 0

  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; 卸载信息（当前用户，无需管理员权限）
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayName" "${APP_NAME}"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayVersion" "${APP_VERSION}"
  WriteRegStr HKCU "${UNINST_KEY}" "Publisher" "${APP_PUBLISHER}"
  WriteRegStr HKCU "${UNINST_KEY}" "URLInfoAbout" "${APP_HOMEPAGE}"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\logo.ico"
  WriteRegStr HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "${UNINST_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKCU "${UNINST_KEY}" "QuietUninstallString" '"$INSTDIR\uninstall.exe" /S'
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoRepair" 1
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKCU "${UNINST_KEY}" "EstimatedSize" "$0"
SectionEnd

Section "Uninstall"
  DetailPrint "正在停止无限画布服务…"
  ExecWait '"$INSTDIR\launcher.exe" killall'
  Sleep 2500

  Delete "$SMPROGRAMS\${APP_NAME}\启动 ${APP_NAME}.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\停止服务.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\查看运行状态.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\查看日志.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\环境检测.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\使用说明.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\卸载 ${APP_NAME}.lnk"
  RMDir "$SMPROGRAMS\${APP_NAME}"
  Delete "$DESKTOP\${APP_NAME}.lnk"

  DeleteRegKey HKCU "${UNINST_KEY}"

  ; 数据目录单独询问，默认保留
  IfFileExists "$INSTDIR\data\*.*" 0 noData
  MessageBox MB_YESNO|MB_ICONQUESTION "是否同时删除数据目录？$\r$\n$\r$\n$INSTDIR\data 中保存着数据库、账号密码、日志与提示词缓存。$\r$\n选择「否」将保留数据，方便以后重装继续使用。" /SD IDNO IDNO keepData
  RMDir /r "$INSTDIR\data"
keepData:
noData:

  RMDir /r "$INSTDIR\web"
  RMDir /r "$INSTDIR\node"
  Delete "$INSTDIR\server.exe"
  Delete "$INSTDIR\launcher.exe"
  Delete "$INSTDIR\环境检测工具.exe"
  Delete "$INSTDIR\logo.ico"
  Delete "$INSTDIR\使用说明.txt"
  Delete "$INSTDIR\LICENSE"
  Delete "$INSTDIR\VERSION"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
SectionEnd
