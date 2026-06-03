#!/bin/bash
# ShareSerial 本地停止脚本

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
echo_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

echo_info "停止 ShareSerial 服务..."

# 方法 1: 从 PID 文件读取
if [ -f /tmp/shareserial-pids.txt ]; then
    PIDS=$(cat /tmp/shareserial-pids.txt)
    echo_info "读取 PID: $PIDS"
    for PID in $PIDS; do
        if ps -p $PID > /dev/null 2>&1; then
            kill $PID 2>/dev/null || true
            echo_info "已终止进程 $PID"
        fi
    done
    rm -f /tmp/shareserial-pids.txt
fi

# 方法 2: 查找并终止所有 ShareSerial 进程
echo_info "查找 ShareSerial 进程..."
pkill -f shareserial-server 2>/dev/null || true
pkill -f shareserial-client 2>/dev/null || true

# 清理 symlink
echo_info "清理虚拟串口 symlink..."
rm -f /tmp/ttyShare* 2>/dev/null || true
sudo rm -f /dev/ttyShare* 2>/dev/null || true

# 检查是否还有进程
if pgrep -f shareserial > /dev/null; then
    echo_warn "仍有 ShareSerial 进程运行"
    ps aux | grep shareserial | grep -v grep
else
    echo_info "所有 ShareSerial 进程已停止"
fi

echo_info "清理完成"