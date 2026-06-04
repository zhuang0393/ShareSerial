@echo off
REM ShareSerial Windows Client Startup Script (with Virtual COM Port)

echo ========================================
echo ShareSerial Windows Client
echo ========================================
echo

REM Check parameters
set SERVER_IP=192.168.246.17
set SERVER_PORT=7700
set LOCAL_PORT=8888

if "%1"=="" (
    echo Using default server: %SERVER_IP%:%SERVER_PORT%
) else (
    set SERVER_IP=%1
)

if "%2"=="" (
    echo Using default local port: %LOCAL_PORT%
) else (
    set LOCAL_PORT=%2
)

echo.
echo Configuration:
echo   Server: %SERVER_IP%:%SERVER_PORT%
echo   Local Port: %LOCAL_PORT%
echo.

REM Check if com0com is installed
set COM0COM_PATH=
if exist "C:\Program Files (x86)\com0com\setupc.exe" (
    set COM0COM_PATH=C:\Program Files (x86)\com0com
)
if exist "C:\Program Files\com0com\setupc.exe" (
    set COM0COM_PATH=C:\Program Files\com0com
)

if not "%COM0COM_PATH%"=="" (
    echo [INFO] com0com detected, searching for virtual COM port...

    REM Find configured virtual COM port
    for /f "tokens=1" %%i in ('"%COM0COM_PATH%\setupc.exe" list 2^>nul ^| findstr "Tcp=127.0.0.1:%LOCAL_PORT%"') do (
        set VCOM_PORT=%%i
    )

    if not "%VCOM_PORT%"=="" (
        echo [OK] Virtual COM port configured: %VCOM_PORT%
        echo.
        echo ========================================
        echo MobaXterm Configuration:
        echo   Type: Serial
        echo   Port: %VCOM_PORT%
        echo   Speed: 115200 baud
        echo ========================================
        echo.
    ) else (
        echo [INFO] Virtual COM port not configured, using TCP proxy mode
        echo.
        echo ========================================
        echo MobaXterm Configuration:
        echo   Type: Raw
        echo   Host: localhost
        echo   Port: %LOCAL_PORT%
        echo ========================================
        echo.
    )
) else (
    echo [INFO] com0com not installed, using TCP proxy mode
    echo.
    echo ========================================
    echo MobaXterm Configuration:
    echo   Type: Raw
    echo   Host: localhost
    echo   Port: %LOCAL_PORT%
    echo ========================================
    echo.
)

REM Start Client
echo [INFO] Starting ShareSerial Client...
echo.

shareserial-client-windows.exe --server %SERVER_IP%:%SERVER_PORT% --local-port %LOCAL_PORT%