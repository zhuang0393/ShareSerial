---
name: architecture-shareserial
description: ShareSerial 系统架构设计
metadata:
  type: project
---

# ARCHITECTURE.md - 系统架构

## 1. 整体架构图（Phase 3 完成）

```mermaid
graph TB
    subgraph "Linux Server Machine"
        SS_L[ShareSerial Server<br>Linux]
        PS_L[Physical Serial<br>/dev/ttyUSB0<br>115200]
        CFG_L[Config File<br>server.yaml]
        
        SS_L --> PS_L
        SS_L --> CFG_L
    end
    
    subgraph "Windows Server Machine"
        SS_W[ShareSerial Server<br>Windows]
        PS_W[Physical Serial<br>COM1<br>115200]
        CFG_W[Config File<br>server-windows.yaml]
        
        SS_W --> PS_W
        SS_W --> CFG_W
    end
    
    subgraph "Linux Client Machine"
        SC_L[ShareSerial Client<br>Linux]
        VS_L[Virtual Serial<br>/dev/vttyShare0<br>PTY]
        CFG_CL[Config File<br>client.yaml]
        TOOL_L[minicom/picocom]
        
        SC_L --> VS_L
        SC_L --> CFG_CL
        TOOL_L --> VS_L
    end
    
    subgraph "Windows Client Machine"
        SC_W[ShareSerial Client<br>Windows]
        LP_W[Local TCP Proxy<br>localhost:8888]
        CFG_CW[Config File<br>client.yaml]
        TOOL_W[Putty/Python]
        
        SC_W --> LP_W
        SC_W --> CFG_CW
        TOOL_W --> LP_W
    end
    
    SS_L -.->|TCP Raw Data| SC_L
    SS_L -.->|TCP Raw Data| SC_W
    SS_W -.->|TCP Raw Data| SC_L
    SS_W -.->|TCP Raw Data| SC_W
```

## 2. 跨平台支持矩阵

| 平台 | 服务端 | 客户端 | 串口类型 | 虚拟串口 |
|------|--------|--------|----------|----------|
| Linux | ✅ | ✅ | /dev/ttyUSB*, /dev/ttyACM* | PTY + symlink |
| Windows | ✅ | ✅ | COM1-COM255 | TCP 端口转发 |

## 3. 服务端架构

### 3.1 模块划分（跨平台）

```mermaid
graph LR
    subgraph "Server Modules (Cross-platform)"
        CMD_L[cmd/server<br>Linux Entry]
        CMD_W[cmd/server-windows<br>Windows Entry]
        
        subgraph "pkg/"
            SERIAL[pkg/serial<br>串口扫描与操作<br>Platform-specific]
            ARB[pkg/arbiter<br>写入仲裁<br>Cross-platform]
        end
        
        subgraph "internal/"
            SRV[internal/server<br>TCP 服务器<br>Cross-platform]
            BC[internal/broadcast<br>多路复用广播<br>Cross-platform]
            CFG[internal/config<br>配置管理<br>Cross-platform]
        end
        
        CMD_L --> SERIAL
        CMD_L --> ARB
        CMD_L --> SRV
        
        CMD_W --> SERIAL
        CMD_W --> ARB
        CMD_W --> SRV
        
        SERIAL --> BC
        ARB --> SERIAL
    end
```

### 3.2 串口模块平台特定实现

| 文件 | 平台 | 说明 |
|------|------|------|
| real_serial_linux.go | Linux | 使用 go.bug.st/serial 操作 /dev/ttyUSB* |
| real_serial_windows.go | Windows | 使用 go.bug.st/serial 操作 COM* |
| scanner_linux.go | Linux | 扫描 /dev/ttyUSB*, /dev/ttyACM*, /dev/ttyS* |
| scanner_windows.go | Windows | 扫描 COM1-COM30 |
| serial.go | All | Port 接口定义、Mock 实现 |

## 4. 客户端架构

### 4.1 Linux 客户端

使用 PTY (Pseudo Terminal) 创建虚拟串口：

```go
type PTYDevice struct {
    master     *os.File    // PTY master
    slave      *os.File    // PTY slave
    slavePath  string      // /dev/pts/X
    symlink    string      // /dev/vttyShare0
}
```

### 4.2 Windows 客户端

使用本地 TCP 端口转发：

```go
type LocalProxy struct {
    localPort  int         // 8888
    remoteConn net.Conn    // 连接到远程服务器
    localConns []net.Conn  // 本地连接（Putty 等）
}
```

## 5. 数据流架构

### 5.1 读取数据流（下行）

```mermaid
graph LR
    PS[Physical Serial] --> SH[SerialHandler]
    SH --> BC[Broadcaster]
    BC --> C1[Client1 Queue]
    BC --> C2[Client2 Queue]
    C1 --> TCP1[TCP Conn]
    C2 --> TCP2[TCP Conn]
    
    subgraph "Linux Client"
        TCP1 --> PTY1[PTY Master]
        PTY1 --> SLAVE1[PTY Slave<br>/dev/vttyShare0]
    end
    
    subgraph "Windows Client"
        TCP2 --> PROXY2[Local Proxy<br>localhost:8888]
        PROXY2 --> PUTTY2[Putty/Python]
    end
```

### 5.2 写入数据流（上行）

```mermaid
graph LR
    subgraph "Arbiter Check"
        ARB[Arbiter] -->|Lock Check| LOCK{Has Lock?}
        LOCK -->|Yes| TCP
        LOCK -->|No| DROP[Drop]
    end
    
    subgraph "Linux Client"
        USER1[User Input] --> SLAVE1[PTY Slave]
        SLAVE1 --> MASTER1[PTY Master]
        MASTER1 --> TCP1[TCP Conn]
    end
    
    subgraph "Windows Client"
        USER2[User Input] --> PUTTY2[Putty]
        PUTTY2 --> PROXY2[Local Proxy]
        PROXY2 --> TCP2[TCP Conn]
    end
    
    TCP1 --> ARB
    TCP2 --> ARB
    TCP --> SH[SerialHandler]
    SH --> PS[Physical Serial]
```

## 6. 关键设计决策

### 6.1 为什么简化协议？

| 方案 | 优点 | 缺点 | 决策 |
|------|------|------|------|
| RFC2217 | 标准、兼容现有工具 | 协议复杂 | ❌ 放弃 |
| 纯 TCP Raw | 简单、性能可控、跨平台 | 不兼容 telnet 工具 | ✅ 采用 |

**决策：纯 TCP Raw Data**
- 实现简单，代码量小
- 跨平台一致性（Linux/Windows）
- 性能可控（延迟 < 100µs）

### 6.2 Windows 为什么不用虚拟串口驱动？

| 方案 | 优点 | 缺点 |
|------|------|------|
| com0com 虚拟驱动 | 兼容所有串口工具 | 需要安装驱动、权限问题 |
| TCP 端口转发 | 无需安装、简单 | 需要用 Putty/Python |

**决策：TCP 端口转发**
- 部署简单，无需安装驱动
- 用户可通过 Putty Raw 连接或 Python socket 连接

### 6.3 为什么用 go.bug.st/serial？

| 方案 | Linux | Windows | 决策 |
|------|-------|---------|------|
| go.bug.st/serial | ✅ | ✅ | ✅ 采用 |
| golang.org/x/sys/exec | 需调用 stty | 需调用 mode | ❌ 放弃 |

**决策：go.bug.st/serial**
- 跨平台一致 API
- 纯 Go 实现，无外部依赖
- 支持所有波特率

## 7. 测试架构

### 7.1 测试覆盖

| 模块 | 单元测试 | 集成测试 | E2E 测试 | 覆盖率 |
|------|----------|----------|----------|--------|
| broadcast | ✅ | ✅ | ✅ | 95.1% |
| arbiter | ✅ | ✅ | ✅ | 94.6% |
| cli | ✅ | ✅ | ✅ | 90.5% |
| server | ✅ | ✅ | ✅ | 82.4% |
| config | ✅ | - | - | 78.3% |
| pty | ✅ | ✅ | ✅ | 75.6% |
| serial | ✅ | - | - | 34.7%* |

*serial 模块覆盖率较低是因为真实串口需要硬件

### 7.2 模拟测试环境

使用 socat 创建虚拟串口对进行仿真测试：

```bash
socat -d -d pty,link=/tmp/ttyVPhysical pty,link=/tmp/ttyVTerminal
```

---

**Why:** 明确模块边界和接口，便于团队协作和测试
**How to apply:** 新增模块需遵循架构图位置，接口变更需评审

## 2. 服务端架构

### 2.1 模块划分（简化版）

```mermaid
graph LR
    subgraph "Server Modules"
        CMD[cmd/server]
        
        subgraph "pkg/"
            SERIAL[pkg/serial<br>串口扫描与操作]
            ARB[pkg/arbiter<br>写入仲裁]
        end
        
        subgraph "internal/"
            BC[internal/broadcast<br>多路复用广播]
            HOT[internal/hotplug<br>热插拔监控]
        end
        
        CMD --> SERIAL
        CMD --> ARB
        
        SERIAL --> BC
        ARB --> SERIAL
        HOT --> SERIAL
    end
```

**已删除模块：**
- ~~pkg/rfc2217~~ → 简化为纯数据转发
- ~~pkg/mdns~~ → 使用配置文件

### 2.2 服务端核心结构

```go
type Server struct {
    config     *Config
    serials    map[string]*SerialHandler  // 串口名称 -> 处理器
    clients    map[string]*ClientConn     // 客户端 ID -> 连接
    arbiter    *Arbiter                    // 写入仲裁器
    mdns       *MDNSService                // mDNS 服务
    hotplug    *HotplugMonitor             // 热插拔监控
    broadcast  *Broadcaster                // 数据广播器
    
    ctx        context.Context
    cancel     context.CancelFunc
    wg         sync.WaitGroup
}

type SerialHandler struct {
    name       string           // /dev/ttyUSB0
    port       *serial.Port     // 物理串口
    config     *SerialConfig    // 当前配置
    clients    map[*Client]bool // 已连接客户端
}

type ClientConn struct {
    id         string
    conn       net.Conn
    rfcHandler *RFC2217Handler
    serial     string          // 连接的串口名
    hasLock    bool            // 是否持有写锁
}
```

## 3. 客户端架构

### 3.1 模块划分

```mermaid
graph LR
    subgraph "Client Modules"
        CMD[cmd/client]
        
        subgraph "pkg/"
            RFC_C[pkg/rfc2217<br>协议处理]
            MDNS_C[pkg/mdns<br>服务发现]
        end
        
        subgraph "internal/"
            PTY[internal/pty<br>虚拟串口]
            RC[internal/reconnect<br>断线重连]
        end
        
        CMD --> RFC_C
        CMD --> MDNS_C
        CMD --> PTY
        
        PTY --> RFC_C
        RC --> RFC_C
    end
```

### 3.2 客户端核心结构

```go
type Client struct {
    config     *Config
    conn       net.Conn            // TCP 连接
    rfcHandler *RFC2217Handler     // RFC2217 处理器
    pty        *PTYDevice          // 虚拟串口
    mdns       *MDNSClient         // mDNS 客户端
    
    reconnect  *ReconnectManager   // 重连管理
    state      ClientState         // 当前状态
    
    ctx        context.Context
    cancel     context.CancelFunc
}

type PTYDevice struct {
    master     *os.File            // PTY master
    slave      *os.File            // PTY slave
    slavePath  string              // /dev/pts/X
    symlink    string              // /dev/vttyShare0
}
```

## 4. 数据流架构

### 4.1 读取数据流（下行）

```mermaid
graph LR
    PS[Physical Serial] --> SH[SerialHandler]
    SH --> BC[Broadcaster]
    BC --> C1[Client1 Queue]
    BC --> C2[Client2 Queue]
    C1 --> RFC1[RFC2217 Handler]
    C2 --> RFC2[RFC2217 Handler]
    RFC1 --> TCP1[TCP Conn]
    RFC2 --> TCP2[TCP Conn]
    TCP1 --> PTY1[PTY Master]
    TCP2 --> PTY2[PTY Master]
    PTY1 --> SLAVE1[PTY Slave]
    PTY2 --> SLAVE2[PTY Slave]
```

### 4.2 写入数据流（上行）

```mermaid
graph LR
    USER[User Input] --> SLAVE[PTY Slave]
    SLAVE --> MASTER[PTY Master]
    MASTER --> RFC[RFC2217 Handler]
    
    subgraph "Arbiter Check"
        ARB[Arbiter] -->|Lock Check| LOCK{Has Lock?}
        LOCK -->|Yes| TCP
        LOCK -->|No| DROP[Drop]
    end
    
    RFC --> ARB
    TCP[TCP Conn] --> SH[SerialHandler]
    SH --> PS[Physical Serial]
```

## 5. 关键设计决策

### 5.1 为什么用 RFC2217？

| 方案 | 优点 | 缺点 |
|------|------|------|
| RFC2217 | 标准、兼容现有工具、有现成客户端 | 协议复杂 |
| 自定义 TCP | 简单、性能可控 | 不兼容现有工具 |
| WebSocket | Web 支持 | 串口工具不支持 |

**决策：RFC2217**
- 兼容 telnet-based 串口客户端（如 ser2net）
- 支持远程波特率配置

### 5.2 为什么用 PTY 而不是 CUSE？

| 方案 | 优点 | 缺点 |
|------|------|------|
| PTY | 简单、无需内核驱动 | 设备名不可自定义 |
| CUSE | 可自定义设备名 | 需要 FUSE、复杂 |

**决策：PTY + symlink**
- PTY 创建简单，稳定
- 创建 symlink `/dev/vttyShare0` -> `/dev/pts/X`

### 5.3 写入仲裁为什么选独占模式？

| 方案 | 优点 | 缺点 |
|------|------|------|
| 独占模式 | 简单、安全 | 串口利用率低 |
| 轮询模式 | 公平 | 实现复杂 |
| 无仲裁 | 最大并发 | 输入冲突 |

**决策：独占模式**
- 嵌入式调试场景，一人操作为主
- 防止多人同时输入导致设备异常

## 6. 接口定义

### 6.1 服务端 API（内部）

```go
// pkg/serial/scanner.go
type Scanner interface {
    Scan() ([]SerialInfo, error)
    Watch(ctx context.Context) (<-chan SerialEvent, error)
}

// pkg/arbiter/lock.go
type Arbiter interface {
    Acquire(clientID string) (bool, error)
    Release(clientID string) error
    CurrentOwner() string
    IsLocked() bool
}
```

### 6.2 客户端 API（内部）

```go
// internal/pty/create.go
type PTYDevice interface {
    Create(name string) error
    Close() error
    Path() string
    Read(buf []byte) (int, error)
    Write(buf []byte) (int, error)
}

// internal/reconnect/reconnect.go
type ReconnectManager interface {
    Connect() error
    Disconnect() error
    AutoReconnect(ctx context.Context) <-chan error
}
```

## 7. 端口与权限

### 7.1 端口分配

| 服务 | 默认端口 |
|------|----------|
| RFC2217 Server | 7700 |
| mDNS | 5353 (UDP) |

### 7.2 权限要求

| 角色 | 权限 |
|------|------|
| 服务端 | dialout 组（串口访问）、net_bind_service（可选） |
| 客户端 | 普通用户（PTY 创建无需特殊权限） |

---

**Why:** 明确模块边界和接口，便于团队协作和测试
**How to apply:** 新增模块需遵循架构图位置，接口变更需评审