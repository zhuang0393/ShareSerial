@echo off
REM ShareSerial Windows 虚拟 COM 口卸载脚本
REM 需要管理员权限运行

echo ========================================
echo ShareSerial Phase 2 - 虚拟 COM 口卸载
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
    echo [WARN] com0com 未安装，无需卸载
    pause
    exit /b 0
)

echo [OK] com0com 已安装: %COM0COM_PATH%
echo.

REM 显示当前配置
echo [INFO] 当前虚拟串口配置:
"%COM0COM_PATH%\setupc.exe" list
echo.

REM 删除所有 ShareSerial 相关的虚拟串口
echo [INFO] 删除 ShareSerial 相关虚拟串口...

REM 查找并删除 TCP 桥接到 8888 的端口
for /f "tokens=1" %%i in ('"%COM0COM_PATH%\setupc.exe" list 2^>nul ^| findstr "Tcp=127.0.0.1:8888"') do (
    echo [INFO] 删除: %%i
    "%COM0COM_PATH%\setupc.exe" remove %%i >nul 2>&1
)

echo.
echo ========================================
echo 卸载完成！
echo ========================================
echo.
echo 如需完全卸载 com0com:
echo   控制面板 → 程序和功能 → 卸载 com0com
echo.
pause