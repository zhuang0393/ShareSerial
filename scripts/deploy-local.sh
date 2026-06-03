#!/bin/bash
# ShareSerial 本地部署脚本
# 用法: ./deploy-local.sh [串口设备] [端口] [PTY路径] [--sudo]

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
echo_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
echo_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 默认参数
SERIAL_PORT=${1:-/dev/ttyUSB0}
SERVER_PORT=${2:-7700}
PTY_PATH=${3:-/tmp/ttyShare0}
USE_SUDO=false

# 检查是否有 --sudo 参数
for arg in "$@"; do
    if [ "$arg" = "--sudo" ]; then
        USE_SUDO=true
        PTY_PATH="/dev/ttyShare0"
    fi
done

# 检查工作目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$(dirname "$SCRIPT_DIR")/bin"

# 检查二进制文件
if [ ! -f "$BIN_DIR/shareserial-server" ]; then
    echo_error "未找到 shareserial-server，请先构建"
    echo_info "运行: make build"
    exit 1
fi

if [ ! -f "$BIN_DIR/shareserial-client" ]; then
    echo_error "未找到 shareserial-client，请先构建"
    echo_info "运行: make build"
    exit 1
fi

echo_info "=========================================="
echo_info "ShareSerial 本地部署"
echo_info "=========================================="
echo_info "物理串口: $SERIAL_PORT"
echo_info "Server 端口: $SERVER_PORT"
echo_info "虚拟串口: $PTY_PATH"
if [ "$USE_SUDO" = true ]; then
    echo_info "模式: sudo (兼容烧录工具)"
else
    echo_info "模式: 普通 (兼容 minicom/screen)"
fi
echo_info "=========================================="

# 步骤 1: 检查串口设备
echo_info "步骤 1: 检查串口设备..."
if [ ! -e "$SERIAL_PORT" ]; then
    echo_warn "串口设备 $SERIAL_PORT 不存在"
    echo_info "可用串口设备:"
    ls -la /dev/ttyUSB* /dev/ttyACM* /dev/ttyS* 2>/dev/null || echo_warn "无串口设备"

    read -p "请输入串口设备路径（如 /dev/ttyACM0）: " SERIAL_PORT
    if [ ! -e "$SERIAL_PORT" ]; then
        echo_error "串口设备 $SERIAL_PORT 不存在，退出"
        exit 1
    fi
fi

# 步骤 2: 检查串口权限
echo_info "步骤 2: 检查串口权限..."
if ! groups $USER | grep -q dialout; then
    echo_warn "当前用户不在 dialout 组"
    read -p "是否添加用户到 dialout 组？(y/n): " answer
    if [ "$answer" = "y" ]; then
        sudo usermod -a -G dialout $USER
        echo_info "已添加用户到 dialout 组，请重新登录使权限生效"
        exit 0
    fi
fi

# 检查串口是否被占用
if sudo fuser "$SERIAL_PORT" 2>/dev/null; then
    echo_warn "串口 $SERIAL_PORT 被占用"
    read -p "是否终止占用进程？(y/n): " answer
    if [ "$answer" = "y" ]; then
        sudo fuser -k "$SERIAL_PORT"
        sleep 1
    fi
fi

# 步骤 3: 检查依赖工具
echo_info "步骤 3: 检查依赖工具..."
if ! which minicom > /dev/null; then
    echo_warn "minicom 未安装"
    read -p "是否安装 minicom？(y/n): " answer
    if [ "$answer" = "y" ]; then
        sudo apt update
        sudo apt install minicom -y
    fi
fi

# 步骤 4: 启动 Server
echo_info "步骤 4: 启动 Server..."
$BIN_DIR/shareserial-server --serial "$SERIAL_PORT" --port "$SERVER_PORT" &
SERVER_PID=$!
sleep 2

# 检查 Server 是否启动成功
if ! ps -p $SERVER_PID > /dev/null; then
    echo_error "Server 启动失败"
    exit 1
fi
echo_info "Server 已启动 (PID: $SERVER_PID)"

# 步骤 5: 启动 Client
echo_info "步骤 5: 启动 Client..."
if [ "$USE_SUDO" = true ]; then
    # sudo 模式：在 /dev/ 创建 symlink
    sudo $BIN_DIR/shareserial-client --server "127.0.0.1:$SERVER_PORT" --pty "$PTY_PATH" &
else
    # 普通模式：在 /tmp/ 创建 symlink
    $BIN_DIR/shareserial-client --server "127.0.0.1:$SERVER_PORT" --pty "$PTY_PATH" &
fi
CLIENT_PID=$!
sleep 2

# 检查 Client 是否启动成功
if ! ps -p $CLIENT_PID > /dev/null; then
    echo_error "Client 启动失败"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi
echo_info "Client 已启动 (PID: $CLIENT_PID)"

# 步骤 6: 检查虚拟串口
echo_info "步骤 6: 检查虚拟串口..."
sleep 1
if [ -L "$PTY_PATH" ]; then
    PTY_TARGET=$(readlink "$PTY_PATH")
    echo_info "虚拟串口已创建: $PTY_PATH -> $PTY_TARGET"
elif [ -e "$PTY_PATH" ]; then
    echo_info "虚拟串口已创建: $PTY_PATH"
else
    echo_warn "虚拟串口未创建，请检查 Client 日志"
fi

# 步骤 7: 检查连接状态
echo_info "步骤 7: 检查连接状态..."
if [ -f "$BIN_DIR/shareserial" ]; then
    $BIN_DIR/shareserial status --server "127.0.0.1:$SERVER_PORT"
fi

# 完成
echo_info "=========================================="
echo_info "部署完成!"
echo_info "=========================================="
echo_info ""
echo_info "使用方法:"
if [ "$USE_SUDO" = true ]; then
    echo_info "  minicom -D $PTY_PATH (兼容所有串口软件)"
    echo_info "  烧录工具可直接选择 $PTY_PATH"
else
    echo_info "  minicom -D $PTY_PATH"
    echo_info "  screen $PTY_PATH"
    echo_info "  烧录工具需手动创建 /dev symlink:"
    echo_info "    sudo ln -sf $(readlink $PTY_PATH) /dev/ttyShare0"
fi
echo_info ""
echo_info "停止服务:"
echo_info "  ./scripts/stop-local.sh"
echo_info "  或手动执行: kill $SERVER_PID $CLIENT_PID"
echo_info ""
echo_info "进程信息已保存到: /tmp/shareserial-pids.txt"
echo_info ""

# 保存 PID
echo "$SERVER_PID $CLIENT_PID" > /tmp/shareserial-pids.txt

# 如果是普通模式，提供额外提示
if [ "$USE_SUDO" = false ]; then
    echo_warn "=========================================="
    echo_warn "提示: 烧录工具兼容性"
    echo_warn "=========================================="
    echo_warn "当前虚拟串口位于 /tmp/ttyShare0"
    echo_warn "烧录工具（如 SDToolBox）可能只识别 /dev/tty*"
    echo_warn ""
    echo_warn "如需烧录工具兼容，请使用 --sudo 参数重新部署:"
    echo_warn "  ./scripts/deploy-local.sh /dev/ttyUSB0 --sudo"
    echo_warn ""
fi