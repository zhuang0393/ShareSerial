#!/bin/bash
# ShareSerial 串口验证脚本
# 检测串口设备、权限、可用性

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}=== ShareSerial 串口验证脚本 ===${NC}"
echo ""

# 检测串口设备
detect_serial_ports() {
    echo -e "${BLUE}检测串口设备...${NC}"

    USB_PORTS=$(ls /dev/ttyUSB* 2>/dev/null || echo "")
    ACM_PORTS=$(ls /dev/ttyACM* 2>/dev/null || echo "")

    ALL_PORTS=""
    if [ -n "$USB_PORTS" ]; then
        ALL_PORTS="$USB_PORTS"
    fi
    if [ -n "$ACM_PORTS" ]; then
        ALL_PORTS="$ALL_PORTS $ACM_PORTS"
    fi

    if [ -z "$ALL_PORTS" ]; then
        echo -e "${YELLOW}未检测到串口设备${NC}"
        echo ""
        echo "可能原因:"
        echo "  1. 设备未连接"
        echo "  2. USB 驱动未加载"
        echo "  3. 设备被其他程序占用"
        echo ""
        echo "检查方法:"
        echo "  lsusb              # 查看USB设备"
        echo "  dmesg | grep tty   # 查看内核日志"
        echo "  ls -la /dev/tty*   # 查看所有tty设备"
        return 1
    fi

    echo -e "${GREEN}检测到串口设备:${NC}"
    for port in $ALL_PORTS; do
        echo "  $port"
    done
    echo ""
    return 0
}

# 检查串口权限
check_permissions() {
    PORT=$1

    echo -e "${BLUE}检查 $PORT 权限...${NC}"

    # 检查文件存在
    if [ ! -e "$PORT" ]; then
        echo -e "${RED}$PORT 不存在${NC}"
        return 1
    fi

    # 检查权限
    PERMS=$(stat -c "%A %U %G" "$PORT" 2>/dev/null || stat -f "%Sp %Su %Sg" "$PORT" 2>/dev/null)
    echo "权限: $PERMS"

    # 检查用户组
    USER_GROUPS=$(groups)
    if echo "$USER_GROUPS" | grep -q "dialout"; then
        echo -e "${GREEN}用户在 dialout 组中${NC}"
    else
        echo -e "${YELLOW}用户不在 dialout 组${NC}"
        echo "解决方法:"
        echo "  sudo usermod -aG dialout $USER"
        echo "  重新登录后生效"
        echo ""

        # 尝试临时权限
        if [ -w "$PORT" ]; then
            echo -e "${GREEN}当前可写入${NC}"
        else
            echo -e "${YELLOW}尝试临时添加权限...${NC}"
            sudo chmod 666 "$PORT" 2>/dev/null || true
            if [ -w "$PORT" ]; then
                echo -e "${GREEN}临时权限已添加${NC}"
            else
                echo -e "${RED}无法获取写入权限${NC}"
                return 1
            fi
        fi
    fi

    return 0
}

# 测试串口打开
test_open_port() {
    PORT=$1

    echo -e "${BLUE}测试打开 $PORT...${NC}"

    # 使用 stty 测试
    if command -v stty &> /dev/null; then
        stty -F "$PORT" 115200 raw -echo 2>/dev/null || {
            echo -e "${YELLOW}stty 配置失败${NC}"
            return 1
        }
        echo -e "${GREEN}stty 配置成功 (115200, raw)${NC}"
    fi

    # 使用 cat 测试读取
    timeout 1 cat "$PORT" 2>/dev/null || true
    echo -e "${GREEN}串口可正常访问${NC}"

    return 0
}

# 检查串口占用
check_port_usage() {
    PORT=$1

    echo -e "${BLUE}检查 $PORT 是否被占用...${NC}"

    # 查找占用进程
    USERS=$(fuser "$PORT" 2>/dev/null || echo "")

    if [ -n "$USERS" ]; then
        echo -e "${YELLOW}$PORT 被以下进程占用:${NC}"
        for pid in $USERS; do
            CMD=$(ps -p $pid -o comm= 2>/dev/null || echo "unknown")
            echo "  PID $pid: $CMD"
        done
        echo ""
        echo "解决方法:"
        echo "  kill $USERS"
        echo "  或使用: fuser -k $PORT"
        return 1
    else
        echo -e "${GREEN}$PORT 未被占用${NC}"
    fi

    return 0
}

# 获取串口信息
get_port_info() {
    PORT=$1

    echo -e "${BLUE}获取 $PORT 信息...${NC}"

    # USB 设备信息
    if [[ "$PORT" == *USB* ]]; then
        USB_NUM=$(echo "$PORT" | sed 's/.*ttyUSB//')
        DEVICE=$(lsusb 2>/dev/null | grep -i "serial" || echo "未知设备")
        echo "USB 设备: $DEVICE"
    fi

    # 当前配置
    if command -v stty &> /dev/null; then
        SETTINGS=$(stty -F "$PORT" 2>/dev/null || echo "")
        if [ -n "$SETTINGS" ]; then
            echo "当前设置: $SETTINGS"
        fi
    fi

    echo ""
}

# 主流程
main() {
    echo ""

    # 1. 检测串口
    if ! detect_serial_ports; then
        echo ""
        echo -e "${YELLOW}=== 验证结果: 未检测到串口 ===${NC}"
        exit 1
    fi

    # 2. 选择第一个串口进行详细检查
    FIRST_PORT=$(ls /dev/ttyUSB* /dev/ttyACM* 2>/dev/null | head -1)

    if [ -z "$FIRST_PORT" ]; then
        echo -e "${RED}无可用串口${NC}"
        exit 1
    fi

    echo -e "${GREEN}选择 $FIRST_PORT 进行详细验证${NC}"
    echo ""

    # 3. 检查权限
    check_permissions "$FIRST_PORT" || exit 1
    echo ""

    # 4. 检查占用
    check_port_usage "$FIRST_PORT" || true
    echo ""

    # 5. 测试打开
    test_open_port "$FIRST_PORT" || exit 1
    echo ""

    # 6. 获取信息
    get_port_info "$FIRST_PORT"

    # 7. 总结
    echo -e "${GREEN}=== 验证结果: 串口可用 ===${NC}"
    echo ""
    echo "推荐的启动命令:"
    echo "  ./bin/shareserial-server --serial $FIRST_PORT --port 7700"
    echo ""
    echo "或使用配置文件:"
    echo "  sed -i 's|path:.*|path: \"$FIRST_PORT\"|' configs/server.yaml"
    echo "  ./bin/shareserial-server --config configs/server.yaml"
}

# 运行
main