#!/bin/bash
# ShareSerial Server 启动脚本

# 默认配置
SERIAL_PORT="/dev/ttyUSB0"
TCP_PORT=7700

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --serial)
            SERIAL_PORT="$2"
            shift 2
            ;;
        --port)
            TCP_PORT="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "Starting ShareSerial Server..."
echo "Serial Port: $SERIAL_PORT"
echo "TCP Port: $TCP_PORT"

# 检查串口是否存在
if [ ! -e "$SERIAL_PORT" ]; then
    echo "Warning: Serial port $SERIAL_PORT not found, using mock mode"
fi

# 启动服务端
./bin/shareserial-server --serial "$SERIAL_PORT" --port "$TCP_PORT"