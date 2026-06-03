# ShareSerial 调试经验总结

## 问题：用户输入无响应

### 现象描述

| 场景 | 行为 |
|------|------|
| 有 log 输出时敲回车 | ✅ 可以显示 Android console |
| 无 log 输出时敲回车 | ❌ 没有任何反应 |

---

## 根因分析

### 问题 1：PTY 行缓冲模式

**现象**：minicom/picocom 需要敲回车才显示串口 log

**根因**：PTY 默认开启 `ICANON`（行缓冲模式），数据需要换行符才输出

**修复**：使用 `cfmakeraw` 设置 PTY 为 raw 模式
- 禁用 `ICANON`（行缓冲）
- 禁用 `ECHO`（回显）
- 禁用 `OPOST`（输出处理）
- 设置 `VMIN=1, VTIME=0`

**代码位置**：`internal/pty/real_pty_linux.go`

```go
// 设置 raw 模式（参考 cfmakeraw）
termios.Iflag &= uint32(^uint32(unix.IGNBRK | unix.BRKINT | ...))
termios.Oflag &= uint32(^uint32(unix.OPOST))
termios.Lflag &= uint32(^uint32(unix.ECHO | unix.ICANON | unix.ISIG | ...))
```

---

### 问题 2：串口互斥锁阻塞（关键问题）

**现象**：无 log 输出时用户输入卡死

**根因**：`RealSerialPort` 的 Read 和 Write 共享同一个互斥锁

```
┌─────────────────────────────────────────────────────┐
│  readSerialAndBroadcast goroutine                   │
│                                                     │
│  serial.Read()                                      │
│      ↓                                              │
│  获取锁 r.mu                                        │
│      ↓                                              │
│  阻塞等待串口数据（一直持有锁！）                    │
│                                                     │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│  handleClient goroutine                             │
│                                                     │
│  用户输入 → serial.Write()                          │
│      ↓                                              │
│  尝试获取锁 r.mu                                     │
│      ↓                                              │
│  等待锁释放（被阻塞！用户输入卡死）                  │
│                                                     │
└─────────────────────────────────────────────────────┘
```

**为什么有 log 输出时能工作**：
- Read 频繁返回数据，释放锁间隙
- Write 能在间隙获取锁执行

**为什么无 log 输出时不能工作**：
- Read 阻塞等待数据，一直持有锁
- Write 无法获取锁，用户输入被卡住

**修复**：移除串口 Read/Write 的互斥锁

```go
// OLD（有问题）
func (r *RealSerialPort) Read(buf []byte) (int, error) {
    r.mu.Lock()         // 获取锁
    defer r.mu.Unlock() // 释放锁（但 Read 可能阻塞很久！）
    return r.port.Read(buf)
}

func (r *RealSerialPort) Write(buf []byte) (int, error) {
    r.mu.Lock()         // 尝试获取锁（被 Read 阻塞）
    defer r.mu.Unlock()
    return r.port.Write(buf)
}

// NEW（修复）
func (r *RealSerialPort) Read(buf []byte) (int, error) {
    return r.port.Read(buf)  // 无锁，go.bug.st/serial 是线程安全的
}

func (r *RealSerialPort) Write(buf []byte) (int, error) {
    return r.port.Write(buf) // 无锁，可以并发执行
}
```

**代码位置**：`pkg/serial/real_serial_linux.go`

---

## 测试缺失分析

### 为什么测试没发现这些问题

| 问题 | 测试未发现原因 |
|------|----------------|
| PTY 行缓冲 | 测试只验证数据流，没验证实时显示 |
| 串口锁阻塞 | MockSerialPort.Read 立即返回 EOF（不阻塞） |
| 并发读写 | 测试都是顺序执行，没有并发场景 |

### MockSerialPort vs RealSerialPort

| 行为 | MockSerialPort | RealSerialPort |
|------|----------------|----------------|
| 无数据时 Read | 返回 EOF（立即） | **阻塞等待数据** |
| 锁持有时间 | 短（立即返回） | **长（阻塞等待）** |
| Write 能否执行 | ✅ 能（锁很快释放） | ❌ 不能（锁被 Read 持有） |

---

## 完善测试方案

### 1. 阻塞行为模拟测试

```go
// BlockingMockSerialPort 模拟真实串口阻塞
func (b *BlockingMockSerialPort) Read(buf []byte) (int, error) {
    if len(b.inputBuffer) == 0 {
        // 无数据时阻塞等待（模拟真实行为）
        <-b.blockChan
    }
    // 有数据时读取
    n := copy(buf, b.inputBuffer)
    return n, nil
}
```

### 2. 并发读写测试

```go
// 启动 Read goroutine（阻塞）
go func() {
    buf := make([]byte, 1024)
    port.Read(buf)  // 阻塞等待
}()

// 同时尝试 Write
go func() {
    port.Write([]byte("user input"))  // 应该能执行
}()

// 验证 Write 是否成功
select {
case <-writeDone:
    t.Log("Write succeeded")
case <-time.After(timeout):
    t.Error("Write blocked by Read!")
}
```

---

## 经验教训

### 1. Mock 与真实实现行为差异

**教训**：Mock 应尽可能模拟真实行为，包括阻塞、延迟、错误等

**建议**：
- 创建 BlockingMockSerialPort 模拟阻塞读取
- 创建 SlowMockSerialPort 模拟慢速传输
- 创建 ErrorMockSerialPort 模拟错误场景

### 2. 并发场景测试必要性

**教训**：串口读取和用户输入是并发操作，需要并发测试

**建议**：
- 测试 Read 阻塞时 Write 能否执行
- 测试多个 Client 同时输入
- 测试 Server 广播时 Client 输入

### 3. 锁的使用原则

**教训**：阻塞操作不应该持有锁

**原则**：
- Read/Write 等阻塞操作不加锁
- 只在非阻塞操作（如状态检查）使用锁
- 使用 separate locks 保护不同资源

---

## 数据流架构

```
┌─────────────┐
│   minicom   │
│  (PTY slave)│
└──────┬──────┘
       │ 用户输入
       ↓
┌─────────────┐
│  PTY master │
│   (Client)  │
└──────┬──────┘
       │ TCP
       ↓
┌─────────────┐
│   Server    │
└──────┬──────┘
       │
  ┌────┴────┐
  │         │
  ↓         ↓
┌─────┐   ┌──────────────────┐
│Write│   │ readSerialAnd    │
│(用户)│   │ Broadcast        │
└──┬──┘   └───────┬──────────┘
   │              │ Read（阻塞）
   ↓              ↓
┌─────────────────────┐
│   物理串口           │
│   /dev/ttyUSB0      │
└─────────────────────┘
   │ 设备响应
   ↓
   └─────────────┬─────────────┘
                 │
                 ↓
         Server Broadcast
                 │
                 ↓
              Client
                 │
                 ↓
             minicom 显示
```

**关键点**：
- Write 和 Read 是独立 goroutine
- Read 阻塞不应影响 Write
- 物理串口底层库（go.bug.st/serial）本身支持并发

---

## 代码变更记录

| Commit | 修改 | 说明 |
|--------|------|------|
| `91b9deb` | PTY raw 模式 | 解决行缓冲问题 |
| `377421e` | 移除串口锁 | 解决并发读写阻塞 |
| `ae429b9` | 完善测试 | 添加并发场景测试 |

---

## 快速诊断命令

```bash
# 检查串口锁问题
grep -A 10 "func.*Read" pkg/serial/real_serial_linux.go | grep "Lock"

# 检查 PTY raw 模式
grep -A 10 "termios" internal/pty/real_pty_linux.go

# 运行并发测试
go test -v ./pkg/serial/ -run "TestConcurrent"

# 运行阻塞测试
go test -v ./pkg/serial/ -run "TestBlocking"
```

---

*Created: 2026-06-03*
*Last Updated: 2026-06-03*