@echo off
REM ShareSerial Virtual COM Port Installer
REM Run as Administrator

echo ========================================
echo ShareSerial Virtual COM Port Installer
echo ========================================
echo

pause

echo Checking administrator privileges...
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] NOT running as Administrator
    echo Please right-click and "Run as administrator"
    pause
    exit /b 1
)

echo [OK] Running as Administrator
echo.

REM Check com0com - try both possible paths
set "COM0COM_PATH="
if exist "C:\Program Files\com0com\setupc.exe" (
    set "COM0COM_PATH=C:\Program Files\com0com"
    goto :found
)
if exist "C:\Program Files (x86)\com0com\setupc.exe" (
    set "COM0COM_PATH=C:\Program Files (x86)\com0com"
    goto :found
)

echo [ERROR] com0com NOT installed
echo Download: https://sourceforge.net/projects/com0com/
pause
exit /b 1

:found
echo [OK] com0com found: %COM0COM_PATH%
echo.

REM Find free COM port
echo Finding available COM port (COM4-COM10)...
set "VCOM=COM4"
echo Trying COM4...
mode COM4 >nul 2>&1
if errorlevel 1 (
    echo [OK] COM4 is free
    goto :create
)

set "VCOM=COM5"
echo Trying COM5...
mode COM5 >nul 2>&1
if errorlevel 1 (
    echo [OK] COM5 is free
    goto :create
)

set "VCOM=COM6"
echo Trying COM6...
mode COM6 >nul 2>&1
if errorlevel 1 (
    echo [OK] COM6 is free
    goto :create
)

echo [ERROR] No free COM port found
pause
exit /b 1

:create
echo.
echo Creating virtual COM port: %VCOM%
echo Bridging to: localhost:8888
echo.

"%COM0COM_PATH%\setupc.exe" install PortName=%VCOM%,Tcp=127.0.0.1:8888

if errorlevel 1 (
    echo [ERROR] Failed to create virtual COM port
    echo Try rebooting and running again
    pause
    exit /b 1
)

echo.
echo [OK] Virtual COM port created!
echo.

echo Current com0com configuration:
"%COM0COM_PATH%\setupc.exe" list
echo.

echo ========================================
echo SUCCESS! Virtual COM Port Installed
echo ========================================
echo.
echo COM Port: %VCOM%
echo Bridge: localhost:8888
echo.
echo NEXT STEPS:
echo 1. Run: shareserial-client-windows.exe --server 192.168.246.17:7700 --local-port 8888
echo 2. MobaXterm: Serial, Port %VCOM%, Speed 115200
echo.
pause