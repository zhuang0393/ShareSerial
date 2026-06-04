@echo off
REM ShareSerial Windows 虚拟 COM 口安装脚本
REM 需要管理员权限运行

echo ========================================
echo ShareSerial Phase 2 - 虚拟 COM 口安装
echo ========================================
echo

REM 检查管理员权限
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [FAIL] 需要管理员权限！
    echo 请右键以管理员身份运行此脚本
    pause
    exit /b 1
)

echo [OK] 管理员权限已确认
echo

REM 检查 com0com 是否已安装
set COM0COM_PATH=
if exist "C:\Program Files (x86)\com0com\setupc.exe" (
    set COM0COM_PATH=C:\Program Files (x86)\com0com
)
if exist "C:\Program Files\com0com\setupc.exe" (
    set COM0COM_PATH=C:\Program Files\com0com
)

if "%COM0COM_PATH%"=="" (
    echo [WARN] com0com 未安装
    echo.
    echo 请先安装 com0com:
    echo   1. 下载: https://sourceforge.net/projects/com0com/
    echo   2. 运行 setup.exe 安装
    echo   3. 重新运行此脚本
    echo.
    pause
    exit /b 1
)

echo [OK] com0com 已安装: %COM0COM_PATH%
echo.

REM 查找可用的 COM 口号
echo [INFO] 查找可用 COM 口号...

REM 检查 COM4-COM10 是否可用
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
    echo [FAIL] 未找到可用的 COM 口号
    pause
    exit /b 1
)

echo [OK] 可用 COM 口: %VCOM_PORT%
echo.

REM 创建 TCP 桥接虚拟串口
echo [INFO] 创建虚拟串口 %VCOM_PORT% (TCP 桥接到 localhost:8888)
echo.

"%COM0COM_PATH%\setupc.exe" install PortName=%VCOM_PORT%,Tcp=127.0.0.1:8888 >nul 2>&1
if %errorlevel% neq 0 (
    echo [FAIL] 创建虚拟串口失败
    echo 可能需要重启后重新运行
    pause
    exit /b 1
)

echo [OK] 虚拟串口已创建
echo.

REM 验证
echo [INFO] 验证虚拟串口配置...
"%COM0COM_PATH%\setupc.exe" list

echo.
echo ========================================
echo 安装完成！
echo ========================================
echo.
echo 虚拟串口: %VCOM_PORT%
echo TCP 桥接: localhost:8888
echo.
echo 使用方法:
echo.
echo 1. 启动 Windows Client:
echo    shareserial-client-windows.exe --server 192.168.246.17:7700 --local-port 8888
echo.
echo 2. MobaXterm 配置:
echo    类型: Serial
echo    端口: %VCOM_PORT%
echo    波特率: 115200
echo.
echo ========================================
pause