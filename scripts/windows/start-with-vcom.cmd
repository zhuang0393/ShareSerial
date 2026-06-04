@echo off
REM ShareSerial Windows 一键启动脚本（带虚拟 COM 口）

echo ========================================
echo ShareSerial Windows Client 启动
echo ========================================
echo

REM 检查参数
set SERVER_IP=192.168.246.17
set SERVER_PORT=7700
set LOCAL_PORT=8888

if "%1"=="" (
    echo 使用默认服务器: %SERVER_IP%:%SERVER_PORT%
) else (
    set SERVER_IP=%1
)

if "%2"=="" (
    echo 使用默认本地端口: %LOCAL_PORT%
) else (
    set LOCAL_PORT=%2
)

echo.
echo 配置:
echo   服务器: %SERVER_IP%:%SERVER_PORT%
echo   本地端口: %LOCAL_PORT%
echo.

REM 检查 com0com 是否已安装并配置
set COM0COM_PATH=
if exist "C:\Program Files (x86)\com0com\setupc.exe" (
    set COM0COM_PATH=C:\Program Files (x86)\com0com
)
if exist "C:\Program Files\com0com\setupc.exe" (
    set COM0COM_PATH=C:\Program Files\com0com
)

if not "%COM0COM_PATH%"=="" (
    echo [INFO] 检测到 com0com，查找虚拟 COM 口...

    REM 查找已配置的虚拟 COM 口
    for /f "tokens=2" %%i in ('"%COM0COM_PATH%\setupc.exe" list 2^>nul ^| findstr "Tcp=127.0.0.1:%LOCAL_PORT%"') do (
        set VCOM_PORT=%%i
    )

    if not "%VCOM_PORT%"=="" (
        echo [OK] 虚拟 COM 口已配置: %VCOM_PORT%
        echo.
        echo ========================================
        echo MobaXterm 配置:
        echo   类型: Serial
        echo   端口: %VCOM_PORT%
        echo   波特率: 115200
        echo ========================================
        echo.
    ) else (
        echo [INFO] 虚拟 COM 口未配置，使用 TCP 代理模式
        echo.
        echo ========================================
        echo MobaXterm 配置:
        echo   类型: Raw
        echo   主机: localhost
        echo   端口: %LOCAL_PORT%
        echo ========================================
        echo.
    )
) else (
    echo [INFO] com0com 未安装，使用 TCP 代理模式
    echo.
    echo ========================================
    echo MobaXterm 配置:
    echo   类型: Raw
    echo   主机: localhost
    echo   端口: %LOCAL_PORT%
    echo ========================================
    echo.
)

REM 启动 Client
echo [INFO] 启动 ShareSerial Client...
echo.

shareserial-client-windows.exe --server %SERVER_IP%:%SERVER_PORT% --local-port %LOCAL_PORT%