@echo off
REM ShareSerial Windows Virtual COM Port Uninstall Script
REM Requires Administrator privileges

echo ========================================
echo ShareSerial Phase 2 - Uninstall Virtual COM
echo ========================================
echo

REM Check administrator privileges
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [FAIL] Administrator privileges required!
    echo Please right-click and run as Administrator
    pause
    exit /b 1
)

echo [OK] Administrator privileges confirmed
echo

REM Check if com0com is installed
set COM0COM_PATH=
if exist "C:\Program Files (x86)\com0com\setupc.exe" (
    set COM0COM_PATH=C:\Program Files (x86)\com0com
)
if exist "C:\Program Files\com0com\setupc.exe" (
    set COM0COM_PATH=C:\Program Files\com0com
)

if "%COM0COM_PATH%"=="" (
    echo [WARN] com0com is not installed, nothing to uninstall
    pause
    exit /b 0
)

echo [OK] com0com installed: %COM0COM_PATH%
echo

REM Show current configuration
echo [INFO] Current virtual COM port configuration:
"%COM0COM_PATH%\setupc.exe" list
echo.

REM Remove all ShareSerial-related virtual COM ports
echo [INFO] Removing ShareSerial virtual COM ports...

REM Find and remove ports bridged to TCP 8888
for /f "tokens=1" %%i in ('"%COM0COM_PATH%\setupc.exe" list 2^>nul ^| findstr "Tcp=127.0.0.1:8888"') do (
    echo [INFO] Removing: %%i
    "%COM0COM_PATH%\setupc.exe" remove %%i >nul 2>&1
)

echo.
echo ========================================
echo Uninstall Complete!
echo ========================================
echo.
echo To completely remove com0com:
echo   Control Panel - Programs - Uninstall com0com
echo.
pause