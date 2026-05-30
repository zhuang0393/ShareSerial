#!/bin/bash
# ShareSerial 24 小时稳定性测试脚本
# 后台运行，定期记录状态，异常告警

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置
TEST_DURATION=${1:-"24h"}  # 默认 24 小时，可指定如 "1h", "30m"
REPORT_DIR="./stability-reports"
LOG_FILE="$REPORT_DIR/stability.log"
STATUS_FILE="$REPORT_DIR/status.json"

# 解析时间
parse_duration() {
    DURATION=$1
    UNIT=$(echo "$DURATION" | sed 's/[0-9]//g')
    VALUE=$(echo "$DURATION" | sed 's/[a-z]//g')

    case "$UNIT" in
        h) SECONDS=$((VALUE * 3600)) ;;
        m) SECONDS=$((VALUE * 60)) ;;
        s) SECONDS=$VALUE ;;
        *) SECONDS=$((24 * 3600)) ;;  # 默认 24 小时
    esac

    echo $SECONDS
}

DURATION_SECONDS=$(parse_duration "$TEST_DURATION")

echo -e "${GREEN}=== ShareSerial 稳定性测试 ===${NC}"
echo ""
echo "测试时长: $TEST_DURATION ($DURATION_SECONDS 秒)"
echo "报告目录: $REPORT_DIR"
echo ""

# 创建报告目录
mkdir -p "$REPORT_DIR"

# 清理旧进程
cleanup() {
    echo ""
    echo -e "${YELLOW}清理进程...${NC}"
    pkill -f shareserial-server 2>/dev/null || true
    pkill -f shareserial-client 2>/dev/null || true
    pkill -f stability-monitor 2>/dev/null || true

    # 生成最终报告
    generate_final_report

    echo -e "${GREEN}=== 测试结束 ===${NC}"
}

trap cleanup EXIT

# 启动服务
start_services() {
    echo -e "${BLUE}启动 Server...${NC}"
    pkill -f shareserial-server 2>/dev/null || true
    sleep 1

    ./bin/shareserial-server --port 7701 &
    SERVER_PID=$!
    echo "Server PID: $SERVER_PID"

    sleep 2

    echo -e "${BLUE}启动 Client...${NC}"
    pkill -f shareserial-client 2>/dev/null || true
    sleep 1

    ./bin/shareserial-client --server 127.0.0.1:7701 &
    CLIENT_PID=$!
    echo "Client PID: $CLIENT_PID"

    sleep 2

    # 检查进程存活
    if ! kill -0 $SERVER_PID 2>/dev/null; then
        echo -e "${RED}Server 启动失败${NC}"
        exit 1
    fi

    if ! kill -0 $CLIENT_PID 2>/dev/null; then
        echo -e "${RED}Client 启动失败${NC}"
        exit 1
    fi

    echo -e "${GREEN}服务启动成功${NC}"
    echo ""
}

# 监控函数
monitor() {
    echo -e "${BLUE}开始监控...${NC}"
    echo ""

    START_TIME=$(date +%s)
    END_TIME=$((START_TIME + DURATION_SECONDS))

    ITERATION=0

    while [ $(date +%s) -lt $END_TIME ]; do
        ITERATION=$((ITERATION + 1))
        CURRENT_TIME=$(date +%s)
        ELAPSED=$((CURRENT_TIME - START_TIME))
        REMAINING=$((END_TIME - CURRENT_TIME))

        # 检查进程状态
        SERVER_RUNNING=$(kill -0 $SERVER_PID 2>/dev/null && echo "true" || echo "false")
        CLIENT_RUNNING=$(kill -0 $CLIENT_PID 2>/dev/null && echo "true" || echo "false")

        # 获取内存使用
        SERVER_MEM=$(ps -p $SERVER_PID -o rss= 2>/dev/null | awk '{print $1/1024}' || echo "0")
        CLIENT_MEM=$(ps -p $CLIENT_PID -o rss= 2>/dev/null | awk '{print $1/1024}' || echo "0")

        # 记录状态
        TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")

        echo "$TIMESTAMP | 迭代: $ITERATION | 已运行: ${ELAPSED}s | 剩余: ${REMAINING}s | Server: ${SERVER_RUNNING} (${SERVER_MEM}MB) | Client: ${CLIENT_RUNNING} (${CLIENT_MEM}MB)" >> "$LOG_FILE"

        # 每 10 次迭代输出一次状态
        if [ $((ITERATION % 10)) -eq 0 ]; then
            echo -e "${BLUE}[$TIMESTAMP]${NC} 已运行: $(format_time $ELAPSED) | 剩余: $(format_time $REMAINING)"
            echo "  Server: ${SERVER_RUNNING} (${SERVER_MEM}MB) | Client: ${CLIENT_RUNNING} (${CLIENT_MEM}MB)"
        fi

        # 异常检测
        if [ "$SERVER_RUNNING" = "false" ]; then
            echo -e "${RED}Server 进程异常退出!${NC}" >> "$LOG_FILE"
            echo -e "${RED}[$TIMESTAMP] Server 进程异常退出!${NC}"

            # 尝试重启
            echo "尝试重启 Server..."
            ./bin/shareserial-server --port 7701 &
            SERVER_PID=$!
            sleep 2
        fi

        if [ "$CLIENT_RUNNING" = "false" ]; then
            echo -e "${RED}Client 进程异常退出!${NC}" >> "$LOG_FILE"
            echo -e "${RED}[$TIMESTAMP] Client 进程异常退出!${NC}"

            # 尝试重启
            echo "尝试重启 Client..."
            ./bin/shareserial-client --server 127.0.0.1:7701 &
            CLIENT_PID=$!
            sleep 2
        fi

        # 内存告警 (> 512MB)
        if [ "$(echo "$SERVER_MEM > 512" | bc)" = "1" ]; then
            echo -e "${YELLOW}[$TIMESTAMP] Server 内存过高: ${SERVER_MEM}MB${NC}" >> "$LOG_FILE"
        fi

        # 等待
        sleep 60
    done
}

# 格式化时间
format_time() {
    SECONDS=$1
    HOURS=$((SECONDS / 3600))
    MINUTES=$(( (SECONDS % 3600) / 60 ))
    SECS=$((SECONDS % 60))

    printf "%02d:%02d:%02d" $HOURS $MINUTES $SECS
}

# 生成最终报告
generate_final_report() {
    echo ""
    echo -e "${GREEN}生成最终报告...${NC}"

    END_TIME=$(date +"%Y-%m-%d %H:%M:%S")

    # 统计数据
    TOTAL_ITERATIONS=$(wc -l < "$LOG_FILE" 2>/dev/null || echo "0")
    ERROR_COUNT=$(grep -c "异常退出" "$LOG_FILE" 2>/dev/null || echo "0")

    # 计算平均内存
    AVG_SERVER_MEM=$(awk '{sum+=$7; count++} END {if(count>0) print sum/count; else print 0}' "$LOG_FILE" 2>/dev/null || echo "0")
    AVG_CLIENT_MEM=$(awk '{sum+=$9; count++} END {if(count>0) print sum/count; else print 0}' "$LOG_FILE" 2>/dev/null || echo "0")

    # 写入报告
    cat > "$REPORT_DIR/report.txt" << EOF
ShareSerial 稳定性测试报告
==========================

测试时长: $TEST_DURATION
开始时间: $(head -1 "$LOG_FILE" | cut -d'|' -f1)
结束时间: $END_TIME

统计数据:
- 总迭代次数: $TOTAL_ITERATIONS
- 异常退出次数: $ERROR_COUNT
- 平均 Server 内存: ${AVG_SERVER_MEM}MB
- 平均 Client 内存: ${AVG_CLIENT_MEM}MB

日志文件: $LOG_FILE

状态:
EOF

    if [ "$ERROR_COUNT" -eq 0 ]; then
        echo "测试通过 - 无异常" >> "$REPORT_DIR/report.txt"
    else
        echo "测试失败 - 发现 $ERROR_COUNT 次异常" >> "$REPORT_DIR/report.txt"
    fi

    echo ""
    echo -e "${GREEN}报告已保存到: $REPORT_DIR/report.txt${NC}"
    cat "$REPORT_DIR/report.txt"
}

# 检查依赖
check_dependencies() {
    echo -e "${BLUE}检查依赖...${NC}"

    if [ ! -f "./bin/shareserial-server" ]; then
        echo -e "${YELLOW}bin/shareserial-server 不存在，构建...${NC}"
        make build-server
    fi

    if [ ! -f "./bin/shareserial-client" ]; then
        echo -e "${YELLOW}bin/shareserial-client 不存在，构建...${NC}"
        make build-client
    fi

    # 检查 bc（用于内存计算）
    if ! command -v bc &> /dev/null; then
        echo -e "${YELLOW}bc 未安装，内存告警功能将受限${NC}"
    fi

    echo ""
}

# 主流程
check_dependencies
start_services
monitor

echo ""
echo -e "${GREEN}=== 测试完成 ===${NC}"
echo "查看报告: cat $REPORT_DIR/report.txt"
echo "查看日志: cat $LOG_FILE"