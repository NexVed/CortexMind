; ============================================================
; installer.nsi — CortexMind Windows installer (NSIS)
;
; Produces a per-user installer (no admin/UAC required) that:
;   - installs CortexMind.exe into %LOCALAPPDATA%\Programs\CortexMind
;   - creates Start Menu + Desktop shortcuts
;   - registers an Add/Remove Programs entry with an uninstaller
;   - installs the Microsoft WebView2 runtime if it isn't already present
;
; Build (from build/build-desktop.ps1, or manually):
;   makensis /DVERSION=0.1.0 build/windows/installer.nsi
; Output: build/dist/CortexMind-Setup-<version>.exe
; ============================================================
Unicode true

!ifndef VERSION
  !define VERSION "0.1.0"
!endif
!define APPNAME "CortexMind"
!define COMPANY "NexVed"
!define EXE "CortexMind.exe"
!define UNINSTKEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}"
!define WV2CLIENT "{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}"

Name "${APPNAME} ${VERSION}"
OutFile "..\dist\CortexMind-Setup-${VERSION}.exe"
RequestExecutionLevel user
InstallDir "$LOCALAPPDATA\Programs\${APPNAME}"
InstallDirRegKey HKCU "Software\${COMPANY}\${APPNAME}" "InstallDir"
SetCompressor /SOLID lzma

!include "MUI2.nsh"
!define MUI_ICON "icon.ico"
!define MUI_UNICON "icon.ico"
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\${EXE}"
!define MUI_FINISHPAGE_RUN_TEXT "Launch CortexMind"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Function .onInit
  ; A running older build keeps its own embedded UI and local auth session alive.
  ; Stop it before files are replaced so this installer can start the new shell.
  nsExec::ExecToLog '"$SYSDIR\taskkill.exe" /F /IM CortexMind.exe'
FunctionEnd
Section "Install"
  SetOutPath "$INSTDIR"
  File "..\dist\${EXE}"

  Call EnsureWebView2

  CreateDirectory "$SMPROGRAMS\${APPNAME}"
  CreateShortcut "$SMPROGRAMS\${APPNAME}\${APPNAME}.lnk" "$INSTDIR\${EXE}" "" "$INSTDIR\${EXE}" 0
  CreateShortcut "$DESKTOP\${APPNAME}.lnk" "$INSTDIR\${EXE}" "" "$INSTDIR\${EXE}" 0

  WriteUninstaller "$INSTDIR\uninstall.exe"
  WriteRegStr HKCU "Software\${COMPANY}\${APPNAME}" "InstallDir" "$INSTDIR"
  WriteRegStr HKCU "${UNINSTKEY}" "DisplayName" "${APPNAME}"
  WriteRegStr HKCU "${UNINSTKEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "${UNINSTKEY}" "Publisher" "${COMPANY}"
  WriteRegStr HKCU "${UNINSTKEY}" "DisplayIcon" "$INSTDIR\${EXE}"
  WriteRegStr HKCU "${UNINSTKEY}" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
  WriteRegDWORD HKCU "${UNINSTKEY}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINSTKEY}" "NoRepair" 1
SectionEnd

; EnsureWebView2 installs Microsoft's Evergreen WebView2 runtime only if no
; runtime is already registered (Windows 11 ships with it). The webview is
; required for the native window; without it the app cannot render.
Function EnsureWebView2
  ReadRegStr $0 HKLM "SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\${WV2CLIENT}" "pv"
  StrCmp $0 "" 0 wv2done
  ReadRegStr $0 HKCU "SOFTWARE\Microsoft\EdgeUpdate\Clients\${WV2CLIENT}" "pv"
  StrCmp $0 "" 0 wv2done

  DetailPrint "Installing the Microsoft WebView2 runtime..."
  InitPluginsDir
  NSISdl::download "https://go.microsoft.com/fwlink/p/?LinkId=2124703" "$PLUGINSDIR\MicrosoftEdgeWebview2Setup.exe"
  Pop $1
  StrCmp $1 "success" 0 wv2fail
  ExecWait '"$PLUGINSDIR\MicrosoftEdgeWebview2Setup.exe" /silent /install'
  Goto wv2done
  wv2fail:
    DetailPrint "WebView2 download failed ($1). CortexMind will offer to install it on first run."
  wv2done:
FunctionEnd

Section "Uninstall"
  Delete "$INSTDIR\${EXE}"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\${APPNAME}\${APPNAME}.lnk"
  RMDir "$SMPROGRAMS\${APPNAME}"
  Delete "$DESKTOP\${APPNAME}.lnk"
  DeleteRegKey HKCU "${UNINSTKEY}"
  DeleteRegKey HKCU "Software\${COMPANY}\${APPNAME}"
SectionEnd
