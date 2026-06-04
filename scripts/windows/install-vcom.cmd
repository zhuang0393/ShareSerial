@echo off
REM ShareSerial Windows Virtual COM Port Installation Script
REM Requires Administrator privileges

echo ========================================
echo ShareSerial Phase 2 - Virtual COM Port
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
    echo [WARN] com0com is not installed
    echo.
    echo Please install com0com first:
    echo   1. Download: https://sourceforge.net/projects/com0com/
    echo   2. Run setup.exe to install
    echo   3. Re-run this script
    echo.
    pause
    exit /b 1
)

echo [OK] com0com installed: %COM0COM_PATH%
echo

REM Find available COM port number
echo [INFO] Finding available COM port...

REM Check COM4-COM15 availability
set VCOM_PORT=
for %%i in (4 5 6 7 8 9 10 11 12 13 14 15) do (
    mode COM%%i >nul 2>&1
    if %errorlevel% neq 0 (
        set VCOM_PORT=COM%%i
        goto :found
    )
)

:found
if "%VCOM_PORT%"=="" (
    echo [FAIL] No available COM port found
    pause
    exit /b 1
)

echo [OK] Available COM port: %VCOM_PORT%
echo

REM Create virtual COM port with TCP bridge
echo [INFO] Creating virtual COM port %VCOM_PORT% (TCP bridge to localhost:8888)
echo.

"%COM0COM_PATH%\setupc.exe" install PortName=%VCOM_PORT%,Tcp=127.0.0.1:8888 >nul 2>&1
if %errorlevel% neq 0 (
    echo [FAIL] Failed to create virtual COM port
    echo Try rebooting and running again
    pause
    exit /b 1
)

echo [OK] Virtual COM port created
echo.

REM Verify
echo [INFO] Verifying virtual COM port configuration...
"%COM0COM_PATH%\setupc.exe" list

echo.
echo ========================================
echo Installation Complete!
echo ========================================
echo.
echo Virtual COM Port: %VCOM_PORT%
echo TCP Bridge: localhost:8888
echo.
echo Usage:
echo.
echo 1. Start Windows Client:
echo    shareserial-client-windows.exe --server 192.168.246.17:7700 --local-port 8888
echo.
echo 2. MobaXterm Configuration:
echo    Type: Serial
echo    Port: %VCOM_PORT%
echo    Speed: 115200 baud
echo.
echo ========================================
pause