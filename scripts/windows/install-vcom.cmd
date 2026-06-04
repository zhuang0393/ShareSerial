@echo off
REM ShareSerial Virtual COM Port Installer
REM Run as Administrator

echo ========================================
echo ShareSerial Virtual COM Port Installer
echo ========================================
echo

REM Pause at start to see output
echo Press any key to continue...
pause >nul

echo Checking administrator privileges...
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo.
    echo [ERROR] NOT running as Administrator!
    echo.
    echo Please right-click this file and select:
    echo "Run as administrator"
    echo.
    pause
    exit /b 1
)

echo [OK] Running as Administrator
echo.

REM Set com0com path
set COM0COM_PATH=C:\Program Files (x86)\com0com

REM Check if com0com exists
if not exist "%COM0COM_PATH%\setupc.exe" (
    set COM0COM_PATH=C:\Program Files\com0com
)

if not exist "%COM0COM_PATH%\setupc.exe" (
    echo.
    echo [ERROR] com0com is NOT installed!
    echo.
    echo Please download and install com0com:
    echo https://sourceforge.net/projects/com0com/
    echo.
    echo After installing com0com, run this script again.
    echo.
    pause
    exit /b 1
)

echo [OK] com0com found at: %COM0COM_PATH%
echo.

REM Find free COM port (try COM4 to COM10)
echo Finding available COM port...
set VCOM=

REM Check each COM port
for %%p in (4 5 6 7 8 9 10) do (
    mode COM%%p >nul 2>&1
    if errorlevel 1 (
        set VCOM=COM%%p
        goto :port_found
    )
)

:port_found
if "%VCOM%"=="" (
    echo.
    echo [ERROR] No free COM port found (COM4-COM10 all in use)
    echo.
    pause
    exit /b 1
)

echo [OK] Free COM port found: %VCOM%
echo.

REM Create virtual COM port
echo Creating virtual COM port %VCOM% bridged to localhost:8888...
echo.

"%COM0COM_PATH%\setupc.exe" install PortName=%VCOM%,Tcp=127.0.0.1:8888

if errorlevel 1 (
    echo.
    echo [ERROR] Failed to create virtual COM port
    echo.
    echo Possible solutions:
    echo 1. Reboot Windows and try again
    echo 2. Check if COM port is already in use
    echo.
    pause
    exit /b 1
)

echo.
echo [OK] Virtual COM port created successfully!
echo.

REM List current configuration
echo Current configuration:
"%COM0COM_PATH%\setupc.exe" list

echo.
echo ========================================
echo Installation Complete!
echo ========================================
echo.
echo Virtual COM Port: %VCOM%
echo Bridges to: localhost:8888
echo.
echo ========================================
echo NEXT STEPS:
echo ========================================
echo.
echo 1. Start ShareSerial Client:
echo    shareserial-client-windows.exe --server 192.168.246.17:7700 --local-port 8888
echo.
echo 2. Open MobaXterm, create new Session:
echo    Type: Serial
echo    Port: %VCOM%
echo    Speed: 115200
echo.
echo ========================================
echo.
pause