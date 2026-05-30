#!/bin/bash
# ShareSerial Client 启动脚本

# 默认配置
SERVER_ADDR="127.0.0.1"
SERVER_PORT=7700
PTY_PATH="/dev/vttyShare0"

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --server)
            SERVER_ADDR="$2"
            shift 2
            ;;
        --port)
            SERVER_PORT="$2"
            shift 2
            ;;
        --pty)
            PTY_PATH="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "Starting ShareSerial Client..."
echo "Server: $SERVER_ADDR:$SERVER_PORT"
echo "Virtual Serial: $PTY_PATH"

# 启动客户端
./bin/shareserial-client --server "$SERVER_ADDR:$SERVER_PORT" --pty "$PTY_PATH"

# 提示使用方式
echo ""
echo "Virtual serial port created: $PTY_PATH"
echo "Use minicom or picocom to access:"
echo "  minicom -D $PTY_PATH"
echo "  picocom $PTY_PATH"