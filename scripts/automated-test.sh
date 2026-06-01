#!/bin/bash
# ShareSerial 全自动化测试脚本
# 一键运行所有测试并生成报告

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REPORT_DIR="$PROJECT_ROOT/test-reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$REPORT_DIR/test_report_$TIMESTAMP.md"
LOG_FILE="$REPORT_DIR/test_log_$TIMESTAMP.log"

# 测试结果
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# 初始化报告目录
init_report_dir() {
    mkdir -p "$REPORT_DIR"
    echo "Test Report Directory: $REPORT_DIR"
}

# 打印步骤
print_step() {
    echo -e "${BLUE}==> $1${NC}"
}

# 打印成功
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# 打印失败
print_fail() {
    echo -e "${RED}✗ $1${NC}"
}

# 打印警告
print_warn() {
    echo -e "${YELLOW}! $1${NC}"
}

# 检查环境依赖
check_dependencies() {
    print_step "Checking dependencies..."

    local missing_deps=()

    # 检查 Go
    if ! command -v go &> /dev/null; then
        missing_deps+=("go")
    else
        print_success "Go $(go version | awk '{print $3}')"
    fi

    # 检查 socat (用于虚拟串口测试)
    if ! command -v socat &> /dev/null; then
        print_warn "socat not found (required for simulation tests)"
        missing_deps+=("socat")
    else
        print_success "socat $(socat -V | head -1)"
    fi

    # 检查 Git
    if ! command -v git &> /dev/null; then
        missing_deps+=("git")
    else
        print_success "git $(git --version | awk '{print $3}')"
    fi

    # 如果有缺失依赖，提示安装
    if [ ${#missing_deps[@]} -gt 0 ]; then
        print_warn "Missing dependencies: ${missing_deps[*]}"
        echo "Install with: sudo apt-get install ${missing_deps[*]}"

        # 如果缺少 socat，跳过模拟测试
        if [[ " ${missing_deps[*]} " =~ " socat " ]]; then
            SKIP_SIMULATION=true
        fi
    fi

    return 0
}

# 设置 Go 环境
setup_go_env() {
    print_step "Setting up Go environment..."

    # 设置 GOPROXY（国内环境）
    go env -w GOPROXY=https://goproxy.cn,direct

    # 设置 GOMODCACHE（避免磁盘空间问题）
    if [ -d "/tmp" ]; then
        mkdir -p /tmp/go-mod
        go env -w GOMODCACHE=/tmp/go-mod
    fi

    print_success "Go environment configured"
}

# 清理旧的构建产物
clean_build() {
    print_step "Cleaning old build artifacts..."

    cd "$PROJECT_ROOT"
    make clean || true

    # 清理测试残留文件
    rm -f /tmp/ttyVPhysical /tmp/ttyVTerminal
    rm -f /tmp/vttyTest*
    rm -f /tmp/go-mod/shareserial-* 2>/dev/null || true

    print_success "Cleaned"
}

# 构建所有组件
build_all() {
    print_step "Building all components..."

    cd "$PROJECT_ROOT"

    # 构建服务端
    echo "Building server..."
    if make build-server; then
        print_success "Server built"
    else
        print_fail "Server build failed"
        return 1
    fi

    # 构建客户端
    echo "Building client..."
    if make build-client; then
        print_success "Client built"
    else
        print_fail "Client build failed"
        return 1
    fi

    # 构建 CLI
    echo "Building CLI..."
    if make build-cli; then
        print_success "CLI built"
    else
        print_fail "CLI build failed"
        return 1
    fi

    print_success "All components built successfully"
    return 0
}

# 运行单元测试
run_unit_tests() {
    print_step "Running unit tests..."

    cd "$PROJECT_ROOT"

    local output_file="$REPORT_DIR/unit_test_output.log"

    # 运行测试并捕获输出
    if go test -v -count=1 ./pkg/... ./internal/... > "$output_file" 2>&1; then
        print_success "Unit tests passed"

        # 解析测试结果
        local passed=$(grep -c "PASS" "$output_file" || echo "0")
        local failed=$(grep -c "FAIL" "$output_file" || echo "0")

        echo "  Passed: $passed tests"
        echo "  Failed: $failed tests"

        PASSED_TESTS=$((PASSED_TESTS + passed))
        FAILED_TESTS=$((FAILED_TESTS + failed))
        TOTAL_TESTS=$((TOTAL_TESTS + passed + failed))

        return 0
    else
        print_fail "Unit tests failed"
        cat "$output_file"
        return 1
    fi
}

# 运行 E2E 测试
run_e2e_tests() {
    print_step "Running E2E tests..."

    cd "$PROJECT_ROOT"

    local output_file="$REPORT_DIR/e2e_test_output.log"

    if go test -v -count=1 ./tests/e2e/... > "$output_file" 2>&1; then
        print_success "E2E tests passed"

        local passed=$(grep -c "PASS" "$output_file" || echo "0")
        local failed=$(grep -c "FAIL" "$output_file" || echo "0")

        echo "  Passed: $passed tests"
        echo "  Failed: $failed tests"

        PASSED_TESTS=$((PASSED_TESTS + passed))
        FAILED_TESTS=$((FAILED_TESTS + failed))
        TOTAL_TESTS=$((TOTAL_TESTS + passed + failed))

        return 0
    else
        print_fail "E2E tests failed"
        cat "$output_file"
        return 1
    fi
}

# 运行模拟测试
run_simulation_tests() {
    print_step "Running simulation tests..."

    # 如果缺少 socat，跳过
    if [ "$SKIP_SIMULATION" = true ]; then
        print_warn "Skipping simulation tests (socat not available)"
        SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
        return 0
    fi

    cd "$PROJECT_ROOT"

    local output_file="$REPORT_DIR/simulation_test_output.log"

    # 检查构建产物
    if [ ! -f "$PROJECT_ROOT/bin/shareserial-server" ]; then
        print_fail "Server binary not found"
        return 1
    fi

    if [ ! -f "$PROJECT_ROOT/bin/shareserial-client" ]; then
        print_fail "Client binary not found"
        return 1
    fi

    # 运行模拟测试（短模式，快速验证）
    if go test -v -short -count=1 ./tests/simulation/... > "$output_file" 2>&1; then
        print_success "Simulation tests passed"

        local passed=$(grep -c "PASS" "$output_file" || echo "0")
        local failed=$(grep -c "FAIL" "$output_file" || echo "0")
        local skipped=$(grep -c "SKIP" "$output_file" || echo "0")

        echo "  Passed: $passed tests"
        echo "  Failed: $failed tests"
        echo "  Skipped: $skipped tests"

        PASSED_TESTS=$((PASSED_TESTS + passed))
        FAILED_TESTS=$((FAILED_TESTS + failed))
        SKIPPED_TESTS=$((SKIPPED_TESTS + skipped))
        TOTAL_TESTS=$((TOTAL_TESTS + passed + failed + skipped))

        return 0
    else
        print_fail "Simulation tests failed"
        cat "$output_file"
        return 1
    fi
}

# 运行完整模拟测试（包含长时间运行测试）
run_full_simulation_tests() {
    print_step "Running full simulation tests (with long-run tests)..."

    if [ "$SKIP_SIMULATION" = true ]; then
        print_warn "Skipping full simulation tests (socat not available)"
        return 0
    fi

    cd "$PROJECT_ROOT"

    local output_file="$REPORT_DIR/simulation_full_test_output.log"

    # 运行完整模拟测试（包括长时间运行测试）
    if go test -v -count=1 ./tests/simulation/... > "$output_file" 2>&1; then
        print_success "Full simulation tests passed"

        local passed=$(grep -c "PASS" "$output_file" || echo "0")
        local failed=$(grep -c "FAIL" "$output_file" || echo "0")

        PASSED_TESTS=$((PASSED_TESTS + passed))
        FAILED_TESTS=$((FAILED_TESTS + failed))
        TOTAL_TESTS=$((TOTAL_TESTS + passed + failed))

        return 0
    else
        print_fail "Full simulation tests failed"
        cat "$output_file"
        return 1
    fi
}

# 生成测试报告
generate_report() {
    print_step "Generating test report..."

    cd "$PROJECT_ROOT"

    # 创建报告文件
    cat > "$REPORT_FILE" << EOF
# ShareSerial 自动化测试报告

**测试时间:** $(date '+%Y-%m-%d %H:%M:%S')

**Git 版本:** $(git rev-parse --short HEAD)

**Go 版本:** $(go version | awk '{print $3}')

---

## 测试摘要

| 指标 | 数值 |
|------|------|
| 总测试数 | $TOTAL_TESTS |
| 通过数 | $PASSED_TESTS |
| 失败数 | $FAILED_TESTS |
| 跳过数 | $SKIPPED_TESTS |
| 通过率 | $(awk "BEGIN {printf \"%.2f%%\", ($PASSED_TESTS/$TOTAL_TESTS)*100}") |

---

## 测试详情

### 单元测试

\`\`\`
$(cat "$REPORT_DIR/unit_test_output.log" 2>/dev/null | grep -E "(PASS|FAIL|=== RUN)" | head -50 || echo "无输出")
\`\`\`

### E2E 测试

\`\`\`
$(cat "$REPORT_DIR/e2e_test_output.log" 2>/dev/null | grep -E "(PASS|FAIL|=== RUN)" | head -30 || echo "无输出")
\`\`\`

### 模拟测试

\`\`\`
$(cat "$REPORT_DIR/simulation_test_output.log" 2>/dev/null | grep -E "(PASS|FAIL|=== RUN|SKIP)" | head -30 || echo "无输出")
\`\`\`

---

## 构建产物

\`\`\`
$(ls -la "$PROJECT_ROOT/bin/" 2>/dev/null || echo "无构建产物")
\`\`\`

---

## 测试环境

- **操作系统:** $(uname -s) $(uname -r)
- **架构:** $(uname -m)
- **socat:** $(command -v socat &> /dev/null && echo "已安装" || echo "未安装")

---

*报告生成时间: $(date '+%Y-%m-%d %H:%M:%S')*
EOF

    print_success "Report generated: $REPORT_FILE"

    # 打印报告摘要
    echo ""
    echo -e "${GREEN}=== 测试报告摘要 ===${NC}"
    echo ""
    cat "$REPORT_FILE" | head -30
}

# 清理测试环境
cleanup() {
    print_step "Cleaning up test environment..."

    # 清理虚拟串口
    rm -f /tmp/ttyVPhysical /tmp/ttyVTerminal
    rm -f /tmp/vttyTest*

    # 终止残留进程
    pkill -f "shareserial-server" 2>/dev/null || true
    pkill -f "shareserial-client" 2>/dev/null || true
    pkill -f "socat.*ttyV" 2>/dev/null || true

    print_success "Cleanup completed"
}

# 主测试流程
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}   ShareSerial 全自动化测试${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    # 初始化
    init_report_dir

    # 重定向所有输出到日志
    exec > >(tee -a "$LOG_FILE") 2>&1

    # 检查依赖
    check_dependencies

    # 设置 Go 环境
    setup_go_env

    # 清理
    clean_build

    # 构建
    if ! build_all; then
        print_fail "Build failed, aborting tests"
        cleanup
        generate_report
        exit 1
    fi

    # 运行测试
    local test_failed=0

    # 单元测试
    if ! run_unit_tests; then
        test_failed=1
    fi

    # E2E 测试
    if ! run_e2e_tests; then
        test_failed=1
    fi

    # 模拟测试
    if ! run_simulation_tests; then
        test_failed=1
    fi

    # 清理
    cleanup

    # 生成报告
    generate_report

    # 最终结果
    echo ""
    echo -e "${BLUE}========================================${NC}"
    if [ $test_failed -eq 0 ]; then
        echo -e "${GREEN}   ✓ 所有测试通过${NC}"
    else
        echo -e "${RED}   ✗ 测试失败${NC}"
    fi
    echo -e "${BLUE}========================================${NC}"

    exit $test_failed
}

# 参数解析
case "$1" in
    --help|-h)
        echo "ShareSerial 全自动化测试脚本"
        echo ""
        echo "用法: $0 [选项]"
        echo ""
        echo "选项:"
        echo "  --quick      快速测试（仅单元测试和 E2E）"
        echo "  --full       完整测试（包含长时间运行测试）"
        echo "  --simulation 仅运行模拟测试"
        echo "  --report     仅生成报告"
        echo "  --help       显示帮助"
        echo ""
        exit 0
        ;;
    --quick)
        init_report_dir
        check_dependencies
        setup_go_env
        clean_build
        build_all
        run_unit_tests
        run_e2e_tests
        cleanup
        generate_report
        ;;
    --simulation)
        init_report_dir
        check_dependencies
        build_all || exit 1
        run_simulation_tests
        cleanup
        generate_report
        ;;
    --full)
        init_report_dir
        check_dependencies
        setup_go_env
        clean_build
        build_all || exit 1
        run_unit_tests
        run_e2e_tests
        run_full_simulation_tests
        cleanup
        generate_report
        ;;
    *)
        main
        ;;
esac