---
name: coding-standards-shareserial
description: ShareSerial 编码规范
metadata:
  type: project
---

# CODING_STANDARDS.md - 编码规范

## 1. Go 语言规范

遵循 [Effective Go](https://golang.org/doc/effective_go) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)。

### 1.1 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 包名 | 小写单词，无下划线 | `serial`, `rfc2217` |
| 导出函数/类型 | 首字母大写 | `NewHandler`, `SerialPort` |
| 内部函数/类型 | 首字母小写 | `parseCommand`, `lockState` |
| 常量 | 大写首字母或驼峰 | `IAC`, `SetBaudRate` |
| 接口 | 动词或名词+er | `Reader`, `Handler`, `Broadcaster` |

### 1.2 错误处理

```go
// 正确：立即处理错误
func OpenSerial(name string) (*serial.Port, error) {
    port, err := serial.Open(name)
    if err != nil {
        return nil, fmt.Errorf("open serial %s: %w", name, err)
    }
    return port, nil
}

// 错误：忽略错误
port, _ := serial.Open(name)  // 禁止
```

### 1.3 Context 使用

```go
// 正确：Context 作为第一个参数
func (s *Server) Run(ctx context.Context) error {
    // ...
}

// 错误：Context 不是第一个参数
func (s *Server) Run(port int, ctx context.Context)  // 禁止
```

## 2. 并发安全

### 2.1 Mutex 规范

```go
// 正确：Mutex 在结构体最前面，嵌入锁类型
type Arbiter struct {
    mu     sync.Mutex
    state  lockState
    owner  *Client
}

// 错误：Mutex 不在最前面
type Arbiter struct {
    state  lockState
    mu     sync.Mutex  // 禁止
}
```

### 2.2 Channel 规范

```go
// 正确：明确 channel 方向
func ReadData(input <-chan []byte, output chan<- []byte)

// 正确：channel 由发送方关闭
// 错误：接收方关闭 channel（可能导致 panic）
```

### 2.3 Goroutine 规范

```go
// 正确：使用 context 控制 goroutine 生命周期
func (b *Broadcaster) Run(ctx context.Context) {
    go func() {
        select {
        case <-ctx.Done():
            return
        case data := <-b.input:
            // process
        }
    }()
}

// 错误：goroutine 无退出机制
go func() {
    for {
        // 无限循环，无法停止  // 禁止
    }
}()
```

## 3. 网络编程规范

### 3.1 TCP 连接处理

```go
// 正确：设置读写超时
conn.SetReadDeadline(time.Now().Add(5 * time.Second))

// 正确：处理连接关闭
_, err := conn.Read(buf)
if err == io.EOF {
    // 客户端正常关闭
}
if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
    // 超时处理
}
```

### 3.2 数据包边界

```go
// 正确：使用固定长度头或分隔符
// RFC2217 使用 Telnet 协议格式（IAC 前缀）

// 错误：假设 TCP 是消息边界
// TCP 是流式协议，Read 可能返回部分数据
```

## 4. 串口操作规范

### 4.1 波特率设置

```go
// 支持的波特率列表
var SupportedBaudRates = []int{
    9600, 19200, 38400, 57600, 115200,
    230400, 460800, 921600, 1500000,
}

// 验证波特率
func isValidBaudRate(rate int) bool {
    for _, r := range SupportedBaudRates {
        if r == rate {
            return true
        }
    }
    return false
}
```

### 4.2 热插拔处理

```go
// 使用 udev 规则或定期扫描
// 不要依赖串口路径不变

// 错误：假设串口路径不变
// /dev/ttyUSB0 可能变成 /dev/ttyUSB1（重新插拔）
```

## 5. 测试规范

### 5.1 单元测试

```go
// 测试文件命名：xxx_test.go
// 测试函数命名：TestXxx

func TestRFC2217ParseSetBaudRate(t *testing.T) {
    input := []byte{0xFF, 0xFA, 0x44, 0x01, 0x00, 0x01, 0xC2, 0x00, 0xFF, 0xF0}
    expected := 115200
    
    rate, err := parseSetBaudRate(input)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if rate != expected {
        t.Errorf("got %d, want %d", rate, expected)
    }
}
```

### 5.2 表驱动测试

```go
func TestParseBaudRate(t *testing.T) {
    tests := []struct {
        name     string
        input    []byte
        expected int
        wantErr  bool
    }{
        {"valid_115200", []byte{...}, 115200, false},
        {"valid_921600", []byte{...}, 921600, false},
        {"invalid_short", []byte{0xFF}, 0, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            rate, err := parseSetBaudRate(tt.input)
            if tt.wantErr && err == nil {
                t.Errorf("expected error, got nil")
            }
            if !tt.wantErr && rate != tt.expected {
                t.Errorf("got %d, want %d", rate, tt.expected)
            }
        })
    }
}
```

### 5.3 集成测试

```go
// 使用 mock 或 fake 串口
type MockSerialPort struct {
    data []byte
}

func (m *MockSerialPort) Read(buf []byte) (int, error) {
    // 返回模拟数据
}
```

## 6. 日志规范

### 6.1 日志级别

| 级别 | 用途 | 示例 |
|------|------|------|
| ERROR | 错误需要人工关注 | `Failed to open serial: permission denied` |
| WARN | 异常但可恢复 | `Client connection timeout, reconnecting` |
| INFO | 关键状态变更 | `New client connected from 192.168.1.10` |
| DEBUG | 调试信息 | `Received RFC2217 SET_BAUDRATE command` |

### 6.2 日志格式

```go
// 正确：结构化日志
log.Printf("[INFO] client connected: ip=%s port=%d", ip, port)

// 错误：无上下文的日志
log.Println("connected")  // 禁止
```

## 7. 注释规范

### 7.1 包注释

```go
// Package rfc2217 implements RFC 2217 (Telnet Remote Serial Port) protocol.
// It provides both server and client side handlers for serial port
// control commands like SET_BAUDRATE, SET_DATASIZE, etc.
package rfc2217
```

### 7.2 函数注释

```go
// NewHandler creates a new RFC2217 handler for the given serial port.
// The handler manages Telnet negotiation and serial port configuration.
func NewHandler(port *serial.Port) *Handler
```

### 7.3 为什么注释（只写 WHY）

```go
// Wait 1 second between retries to avoid overwhelming the server
// during network instability.
time.Sleep(1 * time.Second)
```

---

**Why:** 统一编码风格，提高代码可维护性
**How to apply:** 所有代码需通过 `go fmt`、`go vet`、`golint` 检查