---
name: implementation-shareserial
description: ShareSerial TDD 实施路线图
metadata:
  type: project
---

# IMPLEMENTATION.md - TDD 路线图

## 1. 马斯克五步工作法

### Step 1: 质疑需求（Question Every Requirement）

| 原需求 | 质疑 | 结论 |
|--------|------|------|
| "完美兼容既有全部串口工具" | 是否真的需要全部？ | 核心兼容 minicom、picocom、SecureCRT |
| "自动扫描 /dev/ttyUSB*、/dev/ttyACM*" | 是否需要其他路径？ | 暂时只需 USB 和 ACM |
| "延迟 < 5ms" | 是否有实测依据？ | 改为目标 < 10ms，实测后优化 |
| "支持高波特率 1500000" | 实际场景最高多少？ | 先支持到 921600，1500000 为扩展 |

### Step 2: 删除部分（Delete Parts）

删除 Phase 1 不需要的功能：
- ~~加密传输~~ → Phase 2
- ~~用户认证~~ → Phase 2
- ~~GUI 界面~~ → Phase 2
- ~~轮询仲裁模式~~ → 仅独占模式
- ~~CUSE 虚拟串口~~ → PTY + symlink
- ~~mDNS 服务发现~~ → 配置文件手动配置
- ~~RFC2217 协议~~ → 简化为纯数据转发
- ~~多波特率支持~~ → 固定 115200
- ~~Windows 支持~~ → 仅 Ubuntu

### Step 3: 简化优化（Simplify and Optimize）

简化后的 Phase 1 核心功能：
1. 服务端：串口扫描（固定波特率 115200） + TCP 服务 + 数据广播
2. 客户端：配置文件连接 + PTY 虚拟串口 + 数据转发
3. 仲裁：独占模式写锁

### Step 4: 加速迭代（Accelerate Cycle Time）

采用 TDD 快速迭代：
- 每个功能点独立测试
- 频繁提交，小步快跑
- 使用 Mock 串口加速测试

### Step 5: 自动化（Automate）

- CI/CD 自动测试
- 自动构建发布
- 自动文档生成

## 2. TDD 实施路线图（简化版）

### 阶段一：基础组件（Red → Green）

#### 任务 1.1: 写入仲裁器

| 步骤 | 测试用例 | 状态 |
|------|----------|------|
| Red | `TestArbiterAcquireLock` | ✅ PASS |
| Red | `TestArbiterReleaseLock` | ✅ PASS |
| Red | `TestArbiterLockTimeout` | ✅ PASS |
| Red | `TestArbiterConcurrentAcquire` | ✅ PASS |
| Green | 实现 `pkg/arbiter/lock.go` | ✅ 完成 |

```go
// tests/unit/arbiter_test.go
func TestArbiterLockTimeout(t *testing.T) {
    arbiter := NewArbiter(30 * time.Second)
    
    // Client1 获取锁
    ok, _ := arbiter.Acquire("client1")
    assert.True(t, ok)
    
    // 模拟超时
    time.Sleep(35 * time.Second)
    
    // 检查锁已释放
    assert.False(t, arbiter.IsLocked())
}
```

#### 任务 1.2: 串口处理器（固定波特率）

| 步骤 | 测试用例 | 状态 |
|------|----------|------|
| Red | `TestSerialHandlerOpenClose` | TODO |
| Red | `TestSerialHandlerReadWrite` | TODO |
| Green | 实现 `pkg/serial/handler.go` | TODO |

**已删除任务：**
- ~~RFC2217 协议解析~~ → 简化为纯数据转发
- ~~多波特率支持~~ → 固定 115200

### 阶段二：服务端核心（Red → Green）

#### 任务 2.1: 数据广播器

| 步骤 | 测试用例 | 状态 |
|------|----------|------|
| Red | `TestBroadcasterOneToMany` | TODO |
| Red | `TestBroadcasterClientQueue` | TODO |
| Red | `TestBroadcasterSlowClient` | TODO |
| Green | 实现 `internal/broadcast/broadcaster.go` | TODO |

```go
// tests/unit/broadcast_test.go
func TestBroadcasterSlowClient(t *testing.T) {
    bc := NewBroadcaster()
    
    // 添加快客户端和慢客户端
    fastClient := NewMockClient()
    slowClient := NewMockClientWithDelay(100 * time.Millisecond)
    
    bc.AddClient(fastClient)
    bc.AddClient(slowClient)
    
    // 广播数据
    bc.Broadcast([]byte("test"))
    
    // 快客户端应立即收到
    assertReceived(t, fastClient, "test", 10*time.Millisecond)
    
    // 慢客户端不应阻塞快客户端
    // 使用独立队列实现
}
```

**已删除任务：**
- ~~mDNS 服务端~~ → 配置文件替代

### 阶段三：客户端核心（Red → Green）

#### 任务 3.1: PTY 虚拟串口

| 步骤 | 测试用例 | 状态 |
|------|----------|------|
| Red | `TestPTYCreate` | TODO |
| Red | `TestPTYReadWrite` | TODO |
| Red | `TestPTYSymlink` | TODO |
| Green | 实现 `internal/pty/create.go` | TODO |

```go
// tests/unit/pty_test.go
func TestPTYCreate(t *testing.T) {
    pty, err := CreatePTY("/dev/vttyShare0")
    assert.NoError(t, err)
    
    // 检查 symlink 存在
    assert.FileExists(t, "/dev/vttyShare0")
    
    // 检查可以使用 minicom 打开
    // （需要集成测试）
}
```

#### 任务 3.2: 断线重连

| 步骤 | 测试用例 | 状态 |
|------|----------|------|
| Red | `TestReconnectOnDisconnect` | TODO |
| Red | `TestReconnectMaxRetry` | TODO |
| Red | `TestReconnectResumeData` | TODO |
| Green | 实现 `internal/reconnect/reconnect.go` | TODO |

#### 任务 3.3: CLI 命令接口（AI 可调用）

| 步骤 | 测试用例 | 状态 |
|------|----------|------|
| Red | `TestCLILogCommand` | TODO |
| Red | `TestCLILogFilter` | TODO |
| Red | `TestCLILogJSONFormat` | TODO |
| Red | `TestCLISendCommand` | TODO |
| Red | `TestCLIStatusCommand` | TODO |
| Green | 实现 `cmd/cli/main.go` | TODO |

```go
// tests/unit/cli_test.go
func TestCLILogFilter(t *testing.T) {
    // 测试 CLI 过滤功能
    output := RunCLI("log", "--filter", "ERROR", "--since", "1m")
    // 检查输出只包含 ERROR 行
    assert.Contains(t, output, "ERROR")
    assert.NotContains(t, output, "INFO")
}

func TestCLILogJSONFormat(t *testing.T) {
    // 测试 JSON 格式输出
    output := RunCLI("log", "--format", "json", "--lines", "10")
    // 检查输出是有效 JSON
    var logs []LogEntry
    err := json.Unmarshal(output, &logs)
    assert.NoError(t, err)
}
```

**已删除任务：**
- ~~mDNS 客户端~~ → 配置文件替代

### 阶段四：集成测试（Blue）

#### 任务 4.1: 服务端集成

```go
// tests/integration/server_test.go
func TestServerFullFlow(t *testing.T) {
    // 使用 Mock 串口
    mockSerial := NewMockSerialPort()
    
    server := NewServer(WithSerial(mockSerial))
    server.Start()
    
    // 模拟客户端连接
    client := NewTestClient()
    client.Connect(server.Addr())
    
    // 发送 RFC2217 命令
    client.SetBaudRate(115200)
    
    // 检查串口配置变更
    assert.Equal(t, 115200, mockSerial.BaudRate())
}
```

#### 任务 4.2: 客户端集成

```go
// tests/integration/client_test.go
func TestClientPTYFlow(t *testing.T) {
    // 启动 Mock 服务端
    server := NewMockRFC2217Server()
    server.Start()
    
    // 启动客户端
    client := NewClient()
    client.Connect(server.Addr())
    
    // 检查 PTY 创建
    assert.FileExists(t, client.PTYPath())
    
    // 写入数据
    pty, _ := os.Open(client.PTYPath())
    pty.Write([]byte("test"))
    
    // 检查服务端收到
    assertReceived(t, server, "test")
}
```

### 阶段五：端到端测试（Blue）

```go
// tests/e2e/full_flow_test.go
func TestEndToEndFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("E2E test requires real serial port")
    }
    
    // 使用真实串口（需要测试环境）
    // ...
}
```

## 3. 测试用例清单（简化版）

### 3.1 单元测试

| 模块 | 测试文件 | 测试数量 | 状态 |
|------|----------|----------|------|
| Arbiter | `arbiter_test.go` | 5 | TODO |
| Serial | `serial_test.go` | 2 | TODO |
| PTY | `pty_test.go` | 3 | TODO |
| Broadcast | `broadcast_test.go` | 3 | TODO |
| CLI | `cli_test.go` | 5 | TODO |

**已删除测试：**
- ~~RFC2217~~
- ~~mDNS~~

### 3.2 集成测试

| 模块 | 测试文件 | 测试数量 | 状态 |
|------|----------|----------|------|
| Server | `server_test.go` | 2 | TODO |
| Client | `client_test.go` | 2 | TODO |

### 3.3 端到端测试

| 场景 | 测试用例 | 状态 |
|------|----------|------|
| 多客户端读取 | `TestMultiClientRead` | TODO |
| 写锁仲裁 | `TestWriteLockArbitration` | TODO |
| 断线重连 | `TestDisconnectReconnect` | TODO |

## 4. 实施计划（简化版）

### Week 1: 基础组件

- [ ] 写入仲裁器（4 个测试用例）
- [ ] 串口处理器（2 个测试用例）

### Week 2: 服务端核心

- [ ] 数据广播器（3 个测试用例）
- [ ] TCP 服务器
- [ ] 服务端集成测试

### Week 3: 客户端核心

- [ ] PTY 虚拟串口（3 个测试用例）
- [ ] 断线重连（3 个测试用例）
- [ ] CLI 命令接口（5 个测试用例）
- [ ] 客户端集成测试

### Week 4: 集成与验收

- [ ] 端到端测试
- [ ] 配置文件解析
- [ ] Claude Skill 封装
- [ ] 24 小时稳定性测试

---

**Why:** 明确 TDD 路线图，确保测试先行
**How to apply:** 每个任务先写测试（Red），再实现功能（Green），最后重构（Blue）