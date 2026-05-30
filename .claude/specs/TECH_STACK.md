---
name: tech-stack-shareserial
description: ShareSerial 技术栈与依赖库
metadata:
  type: project
---

# TECH_STACK.md - 技术栈定义

## 1. 开发环境

### 1.1 Go 版本

- **最低版本**: Go 1.21
- **推荐版本**: Go 1.22+

### 1.2 构建工具

| 工具 | 版本 | 用途 |
|------|------|------|
| Make | 任意 | 构建脚本 |
| Go modules | 1.21+ | 依赖管理 |

## 2. 核心依赖库

### 2.1 串口操作

```go
// go.bug.org/serial - Go 串口库
import "go.bug.org/serial"

// 功能：
// - OpenPort(name, mode) - 打开串口
// - Read(buf) / Write(buf) - 读写数据
// - Close() - 关闭串口

// 波特率：固定 115200
// 数据位：8
// 停止位：1
// 校验：无
```

### 2.2 PTY 虚拟终端

```go
// golang.org/x/sys/unix - POSIX 系统调用
import "golang.org/x/sys/unix"

// 创建 PTY
master, slave, err := unix.Openpty()
// 设置 termios 属性（固定 115200）
unix.IoctlSetTermios(int(master.Fd()), unix.TCSETS, &termios)
```

### 2.3 网络与并发

```go
// 标准库（无额外依赖）
import (
    "net"           // TCP 连接
    "sync"          // 并发控制
    "context"       // 上下文管理
    "time"          // 超时控制
    "os"            // 文件操作
    "io"            // 数据流
)
```

**已删除依赖：**
- ~~github.com/hashicorp/mdns~~ → 配置文件替代
- ~~RFC2217 自定义实现~~ → 简化为纯数据转发

## 3. CLI 命令设计

### 3.1 CLI 子命令

```go
// cmd/cli/main.go
// 使用 cobra 或标准 flag 库

// shareserial log - 获取实时 Log
// shareserial send - 发送命令（需要写锁）
// shareserial status - 查看连接状态
// shareserial connect - 连接服务端（后台模式）
// shareserial disconnect - 断开连接
```

### 3.2 CLI 输出格式

```go
// Text 格式
[17:30:00.123] INFO: System starting...

// JSON 格式（便于 AI 解析）
{"timestamp":"2026-05-28T17:30:00.123Z","level":"INFO","message":"System starting..."}
```

## 4. 项目结构（简化版）

```
/workspace/hengzhuang.jin/ss/
├── cmd/
│   ├── server/
│   │   └── main.go          # 服务端入口
│   ├── client/
│   │   └── main.go          # 客户端入口（PTY 模式）
│   └── cli/
│       └── main.go          # CLI 入口（AI 可调用）
├── pkg/
│   ├── serial/
│   │   ├── scanner.go       # 串口扫描
│   │   ├── handler.go       # 串口处理
│   │   └── config.go        # 配置（固定 115200）
│   └── arbiter/
│       ├── lock.go          # 写锁管理
│       └── timeout.go       # 超时处理
│   └── logparser/
│       ├── parser.go        # Log 解析（过滤、时间范围）
│       └── formatter.go     # 输出格式化（text/json）
├── internal/
│   ├── pty/
│   │   ├── create.go        # PTY 创建
│   │   └── termios.go       # termios 配置
│   ├── broadcast/
│   │   └── broadcaster.go   # 数据广播
│   └── reconnect/
│       └── reconnect.go     # 断线重连
├── tests/
│   ├── unit/
│   │   ├── arbiter_test.go
│   │   ├── serial_test.go
│   │   ├── broadcast_test.go
│   │   └── cli_test.go
│   ├── integration/
│   │   ├── server_test.go
│   │   └── client_test.go
│   └── e2e/
│       └── full_flow_test.go
├── configs/
│   ├── server.yaml          # 服务端配置示例
│   └── client.yaml          # 客户端配置示例
├── .claude/
│   └── skills/
│       ├── shareserial-log.md    # Log 获取 Skill
│       ├── shareserial-send.md   # 命令发送 Skill
│       └── shareserial-status.md # 状态查看 Skill
├── go.mod
├── Makefile
└── CLAUDE.md
```

**已删除目录：**
- ~~pkg/rfc2217~~
- ~~pkg/mdns~~

## 5. 构建与部署

### 5.1 Makefile

```makefile
.PHONY: build build-server build-client test clean

VERSION := 1.0.0
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

build: build-server build-client

build-server:
	go build $(LDFLAGS) -o bin/shareserial-server ./cmd/server

build-client:
	go build $(LDFLAGS) -o bin/shareserial-client ./cmd/client

test:
	go test -v ./...

clean:
	rm -rf bin/
```

### 5.2 Go Modules

```go
module shareserial

go 1.22

require (
    go.bug.org/serial v1.6.1
    golang.org/x/sys v0.20.0
)
```

**已删除依赖：**
- ~~github.com/hashicorp/mdns~~

## 6. 性能优化策略

### 6.1 服务端广播

- 使用 `epoll` 思想（Go 使用 goroutine + channel）
- 单次读取，多 goroutine 并发写入客户端
- 客户端队列隔离，防止慢客户端阻塞

```go
// 设计模式
type Broadcaster struct {
    input   chan []byte       // 串口数据输入
    clients map[*Client]bool  // 已连接客户端
    buffer  []byte            // 数据缓冲
}

func (b *Broadcaster) Run() {
    for data := range b.input {
        for client := range b.clients {
            client.SendQueue <- data  // 每个客户端独立队列
        }
    }
}
```

### 6.2 客户端虚拟串口

- PTY master → slave 直接转发
- 避免额外数据处理开销
- termios 配置通过 RFC2217 透传

---

**Why:** 明确技术栈边界，防止引入不兼容依赖
**How to apply:** 所有依赖需在 TECH_STACK.md 中登记，新增依赖需评审