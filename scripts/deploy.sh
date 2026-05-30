#!/bin/bash
# ShareSerial 一键部署脚本
# 支持配置文件和 systemd 服务

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

VERSION="1.0.0"

echo -e "${GREEN}=== ShareSerial v${VERSION} 部署脚本 ===${NC}"
echo ""

# 检测 Go（仅构建时需要）
check_go() {
    if ! command -v go &> /dev/null; then
        if [ -f /tmp/go/bin/go ]; then
            export PATH=/tmp/go/bin:$PATH
            echo -e "${YELLOW}使用 /tmp/go/bin/go${NC}"
        else
            echo -e "${RED}Go 未安装，但可使用已打包的 bin/ 目录${NC}"
            return 1
        fi
    fi
    return 0
}

# 设置 Go 环境（构建时使用）
setup_go_env() {
    go env -w GOPROXY=https://goproxy.cn,direct 2>/dev/null || true
    export GOMODCACHE=/tmp/go-mod
    mkdir -p /tmp/go-mod
}

# 检测角色
ROLE=${1:-"help"}
SERVER_IP=${2:-""}
SERIAL_PORT=${3:-""}
CONFIG_PATH=${4:-""}

case "$ROLE" in
    server)
        echo -e "${BLUE}部署 Server 端${NC}"
        echo ""

        # 检测串口
        if [ -n "$SERIAL_PORT" ]; then
            SERIAL=$SERIAL_PORT
        else
            SERIAL=$(ls /dev/ttyUSB* /dev/ttyACM* 2>/dev/null | head -1)
        fi

        if [ -z "$SERIAL" ]; then
            echo -e "${YELLOW}未检测到物理串口，使用 Mock 模式${NC}"
            SERIAL="/dev/ttyMock0"
        else
            echo -e "${GREEN}检测到串口: $SERIAL${NC}"

            # 检查权限
            if [ ! -r "$SERIAL" ] || [ ! -w "$SERIAL" ]; then
                echo -e "${YELLOW}串口无权限${NC}"
                echo "解决方法:"
                echo "  sudo usermod -aG dialout $USER"
                echo "  重新登录后生效"
                echo ""
                # 尝试临时添加权限
                sudo chmod 666 $SERIAL 2>/dev/null || true
            fi
        fi

        # 检查 bin 目录
        if [ ! -f bin/shareserial-server ]; then
            echo -e "${YELLOW}bin/shareserial-server 不存在，尝试构建...${NC}"
            if check_go; then
                setup_go_env
                make build-server
            else
                echo -e "${RED}无法构建，请确保 bin/ 目录存在${NC}"
                exit 1
            fi
        fi

        # 准备配置文件
        CONFIG_FILE=${CONFIG_PATH:-"./configs/server.yaml"}
        if [ ! -f "$CONFIG_FILE" ]; then
            echo -e "${YELLOW}配置文件不存在，使用默认配置${NC}"
            CONFIG_FILE=""
        fi

        # 启动 Server
        echo -e "${GREEN}启动 Server...${NC}"
        echo "串口: $SERIAL"
        echo "端口: 7700"
        echo "配置: ${CONFIG_FILE:-默认}"
        echo ""

        # 检查端口是否被占用
        if netstat -tuln 2>/dev/null | grep -q ":7700 "; then
            echo -e "${YELLOW}端口 7700 已被占用${NC}"
            echo "请先停止现有服务或使用其他端口"
            exit 1
        fi

        # 启动（优先使用配置文件）
        if [ -n "$CONFIG_FILE" ]; then
            ./bin/shareserial-server --config "$CONFIG_FILE" --serial "$SERIAL"
        else
            ./bin/shareserial-server --serial "$SERIAL" --port 7700
        fi
        ;;

    client)
        echo -e "${BLUE}部署 Client 端${NC}"
        echo ""

        if [ -z "$SERVER_IP" ]; then
            echo -e "${RED}请指定 Server IP${NC}"
            echo "用法: ./scripts/deploy.sh client <SERVER_IP>"
            echo "示例: ./scripts/deploy.sh client 192.168.1.100"
            exit 1
        fi

        # 检查 bin 目录
        if [ ! -f bin/shareserial-client ]; then
            echo -e "${YELLOW}bin/shareserial-client 不存在，尝试构建...${NC}"
            if check_go; then
                setup_go_env
                make build-client
            else
                echo -e "${RED}无法构建，请确保 bin/ 目录存在${NC}"
                exit 1
            fi
        fi

        # 准备配置文件
        CONFIG_FILE=${CONFIG_PATH:-"./configs/client.yaml"}
        if [ ! -f "$CONFIG_FILE" ]; then
            echo -e "${YELLOW}配置文件不存在，使用默认配置${NC}"
            CONFIG_FILE=""
        fi

        # 测试连接
        echo -e "${GREEN}测试连接 $SERVER_IP:7700...${NC}"
        if ! timeout 5 bash -c "echo > /dev/tcp/$SERVER_IP/7700" 2>/dev/null; then
            echo -e "${YELLOW}无法连接到 Server${NC}"
            echo "请检查:"
            echo "  1. Server 是否已启动"
            echo "  2. IP 地址是否正确: $SERVER_IP"
            echo "  3. 网络是否可达"
            echo ""
            read -p "是否仍要尝试启动 Client? (y/n) " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                exit 1
            fi
        else
            echo -e "${GREEN}连接测试成功${NC}"
        fi

        # 启动 Client
        echo -e "${GREEN}启动 Client...${NC}"
        echo "Server: $SERVER_IP:7700"
        echo "虚拟串口: /dev/vttyShare0"
        echo "配置: ${CONFIG_FILE:-默认}"
        echo ""

        # 启动（优先使用配置文件）
        if [ -n "$CONFIG_FILE" ]; then
            # 更新配置文件中的服务器地址
            sed -i "s/address:.*$/address: \"$SERVER_IP\"/" "$CONFIG_FILE" 2>/dev/null || true
            ./bin/shareserial-client --config "$CONFIG_FILE"
        else
            ./bin/shareserial-client --server "$SERVER_IP:7700"
        fi

        echo ""
        echo -e "${GREEN}=== Client 已启动 ===${NC}"
        echo "虚拟串口路径: /dev/vttyShare0"
        echo ""
        echo "使用方法:"
        echo "  minicom -D /dev/vttyShare0"
        echo "  picocom /dev/vttyShare0"
        echo "  cat /dev/vttyShare0"
        ;;

    test)
        echo -e "${BLUE}本地测试模式（Server + Client 同机）${NC}"
        echo ""

        # 检查 bin
        if [ ! -f bin/shareserial-server ] || [ ! -f bin/shareserial-client ]; then
            echo -e "${YELLOW}构建...${NC}"
            if check_go; then
                setup_go_env
                make build
            else
                echo -e "${RED}无法构建${NC}"
                exit 1
            fi
        fi

        # 清理旧进程
        pkill -f shareserial-server 2>/dev/null || true
        pkill -f shareserial-client 2>/dev/null || true
        sleep 1

        echo -e "${GREEN}启动本地 Server (Mock 模式)...${NC}"
        ./bin/shareserial-server --config ./configs/server.yaml --port 7701 &
        SERVER_PID=$!

        sleep 2

        echo -e "${GREEN}启动本地 Client...${NC}"
        ./bin/shareserial-client --server 127.0.0.1:7701 &
        CLIENT_PID=$!

        sleep 2

        echo -e "${GREEN}测试 CLI...${NC}"
        ./bin/shareserial log --server 127.0.0.1:7701 --since 1m

        echo ""
        echo -e "${GREEN}=== 测试完成 ===${NC}"
        echo "Server PID: $SERVER_PID"
        echo "Client PID: $CLIENT_PID"
        echo ""
        echo "停止服务:"
        echo "  kill $SERVER_PID $CLIENT_PID"
        ;;

    build)
        echo -e "${BLUE}构建所有组件${NC}"
        echo ""

        if ! check_go; then
            echo -e "${RED}Go 未安装${NC}"
            exit 1
        fi

        setup_go_env
        make build

        echo -e "${GREEN}=== 构建完成 ===${NC}"
        ls -la bin/
        ;;

    package)
        echo -e "${BLUE}打包发布${NC}"
        echo ""

        ./scripts/package.sh
        ;;

    install)
        echo -e "${BLUE}安装到系统${NC}"
        echo ""

        # 安装二进制文件
        sudo cp bin/shareserial-server /usr/local/bin/
        sudo cp bin/shareserial-client /usr/local/bin/
        sudo cp bin/shareserial /usr/local/bin/

        # 安装配置文件
        sudo mkdir -p /etc/shareserial
        sudo cp configs/server.yaml /etc/shareserial/
        sudo cp configs/client.yaml /etc/shareserial/

        echo -e "${GREEN}=== 安装完成 ===${NC}"
        echo "二进制文件: /usr/local/bin/"
        echo "配置文件: /etc/shareserial/"
        ;;

    uninstall)
        echo -e "${BLUE}卸载${NC}"
        echo ""

        sudo rm -f /usr/local/bin/shareserial-server
        sudo rm -f /usr/local/bin/shareserial-client
        sudo rm -f /usr/local/bin/shareserial
        sudo rm -rf /etc/shareserial

        echo -e "${GREEN}=== 卸载完成 ===${NC}"
        ;;

    help|--help|-h)
        echo "用法: ./scripts/deploy.sh <命令> [参数]"
        echo ""
        echo "命令:"
        echo "  server [串口] [配置]    启动 Server 端"
        echo "  client <IP> [配置]      启动 Client 端"
        echo "  test                    本地测试（Server + Client）"
        echo "  build                   构建所有组件"
        echo "  package                 打包发布"
        echo "  install                 安装到系统"
        echo "  uninstall               卸载"
        echo "  help                    显示帮助"
        echo ""
        echo "示例:"
        echo "  ./scripts/deploy.sh server"
        echo "  ./scripts/deploy.sh server /dev/ttyUSB0"
        echo "  ./scripts/deploy.sh server /dev/ttyUSB0 ./configs/server.yaml"
        echo "  ./scripts/deploy.sh client 192.168.1.100"
        echo "  ./scripts/deploy.sh client 192.168.1.100 ./configs/client.yaml"
        echo "  ./scripts/deploy.sh test"
        echo "  ./scripts/deploy.sh install"
        ;;

    *)
        echo -e "${RED}未知命令: $ROLE${NC}"
        echo "使用 './scripts/deploy.sh help' 查看帮助"
        exit 1
        ;;
esac