---
name: system-flow-shareserial
description: ShareSerial 系统调用流程与状态机
metadata:
  type: project
---

# SYSTEM_FLOW.md - 系统调用流程

## 1. 系统交互时序图（简化版）

### 1.1 服务启动流程

```mermaid
sequenceDiagram
    participant S as Server
    participant P as PhysicalSerial
    participant N as Network
    participant C as ConfigFile

    S->>C: Read server.yaml
    C-->>S: Return config (port, baudrate)
    S->>P: Open /dev/ttyUSB0 (115200)
    P-->>S: Serial ready
    S->>N: Listen on port 7700
```

### 1.2 客户端连接流程

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant V as VirtualSerial
    participant U as UserTool/AI
    participant CFG as ConfigFile

    C->>CFG: Read client.yaml
    CFG-->>C: Return server IP/port
    C->>S: Connect TCP
    S-->>C: Connected
    C->>V: Create PTY (/dev/vttyShare0)
    V-->>U: Expose as standard serial
    U->>V: Read Log data
    V->>C: Forward to stdout/file
```

### 1.3 数据流转（读取）

```mermaid
sequenceDiagram
    participant P as PhysicalSerial
    participant S as Server
    participant C1 as Client1
    participant C2 as Client2
    participant V1 as VirtualSerial1
    participant V2 as VirtualSerial2

    P->>S: Data received (epoll)
    S->>S: Broadcast to all clients
    S->>C1: TCP data packet
    S->>C2: TCP data packet
    C1->>V1: Write to PTY master
    C2->>V2: Write to PTY master
    V1-->>U1: User reads from slave
    V2-->>U2: User reads from slave
```

### 1.4 写入仲裁流程（独占模式）

```mermaid
sequenceDiagram
    participant C1 as Client1
    participant C2 as Client2
    participant S as Server
    participant A as Arbiter
    participant P as PhysicalSerial

    C1->>S: Request Write Lock
    S->>A: Check lock status
    A-->>S: Lock free, grant to C1
    S-->>C1: Lock granted
    S->>S: Broadcast lock status
    S->>C2: Notify: C1 has lock
    C2->>C2: Enter read-only mode
    
    C1->>S: Send command data
    S->>P: Write to physical serial
    
    Note over A: 30s timeout
    
    A->>A: Timeout, release lock
    S->>S: Broadcast lock released
    S->>C1: Notify: lock expired
    S->>C2: Notify: lock free
```

## 2. 状态机定义

### 2.1 服务端串口状态

```mermaid
stateDiagram-v2
    [*] --> Idle: Server Start
    Idle --> Connected: Client Connect
    Connected --> Idle: All Clients Disconnect
    Connected --> WriteLocked: Client Acquire Lock
    WriteLocked --> Connected: Lock Release/Timeout
    WriteLocked --> Idle: Lock Owner Disconnect
```

### 2.2 客户端连接状态（简化版）

```mermaid
stateDiagram-v2
    [*] --> Init: Read Config
    Init --> Connecting: Connect to Server
    Connecting --> Connected: TCP OK
    Connecting --> Init: Connection Failed
    Connected --> ReadOnly: Default
    Connected --> WriteLocked: Lock Granted
    ReadOnly --> WriteLocked: Acquire Lock
    WriteLocked --> ReadOnly: Release Lock
    WriteLocked --> ReadOnly: Lock Timeout
    Connected --> Reconnecting: Network Error
    Reconnecting --> Connected: Reconnect OK
    Reconnecting --> Init: Max Retry Exceeded
```

### 2.3 写锁状态

```mermaid
stateDiagram-v2
    [*] --> Free
    Free --> Locked: Client Acquire
    Locked --> Free: Client Release
    Locked --> Free: Timeout (30s)
    Locked --> Free: Client Disconnect
```

## 3. AI 调用 CLI 接口流程

### 3.1 CLI 命令设计

```bash
# 获取实时 Log（输出到 stdout）
shareserial log --server 192.168.1.100:7700

# 过滤关键词
shareserial log --server 192.168.1.100:7700 --filter "ERROR|WARN"

# 时间范围
shareserial log --server 192.168.1.100:7700 --since "5m" --until "2m"

# 输出 JSON 格式（便于程序解析）
shareserial log --server 192.168.1.100:7700 --format json

# 获取写锁并发送命令
shareserial send --server 192.168.1.100:7700 --command "reboot"

# 查看连接状态
shareserial status --server 192.168.1.100:7700
```

### 3.2 AI 调用时序图

```mermaid
sequenceDiagram
    participant AI as Claude Code/AI Agent
    participant CLI as shareserial CLI
    participant S as Server
    participant P as PhysicalSerial

    AI->>CLI: Execute shareserial log --filter ERROR
    CLI->>S: Connect TCP (read-only mode)
    S-->>CLI: Stream Log data
    CLI->>CLI: Filter by "ERROR"
    CLI-->>AI: Return filtered logs to stdout
    AI->>AI: Analyze Log content
    AI->>CLI: Execute shareserial send --command "dmesg"
    CLI->>S: Request write lock
    S-->>CLI: Lock granted
    CLI->>S: Send command "dmesg"
    S->>P: Write to serial
    P-->>S: Response data
    S-->>CLI: Stream response
    CLI-->>AI: Return response to stdout
    CLI->>S: Release lock
```

### 3.3 JSON 输出格式

```json
{
  "timestamp": "2026-05-28T17:30:00Z",
  "level": "ERROR",
  "message": "kernel: Unable to handle kernel NULL pointer dereference",
  "source": "kernel",
  "raw": "[17:30:00] ERROR: kernel: Unable to handle kernel NULL pointer dereference"
}
```

### 3.4 Claude Skill 封装

```yaml
# .claude/skills/shareserial-log.md
name: shareserial-log
description: 获取远程串口 Log 数据，支持过滤和分析
trigger: 用户提到 "查看 Log"、"串口日志"、"分析 Log"

parameters:
  - name: filter
    description: 过滤关键词（正则表达式）
    default: ""
  - name: since
    description: 时间范围起点（如 "5m" 表示最近5分钟）
    default: "1m"
  - name: format
    description: 输出格式（text/json）
    default: "text"

command: shareserial log --server ${SERVER} --filter "${filter}" --since "${since}" --format ${format}
```

**已删除章节：**
- ~~RFC2217 协议交互~~ → 简化为纯数据转发
- ~~mDNS 服务发现~~ → 配置文件替代

## 4. 错误处理流程

### 5.1 网络断开

```mermaid
flowchart TD
    A[Network Error Detected] --> B{Retry Count < Max?}
    B -->|Yes| C[Wait 1s]
    C --> D[Reconnect]
    D --> E{Connected?}
    E -->|Yes| F[Resume Data Flow]
    E -->|No| B
    B -->|No| G[Notify User]
    G --> H[Keep PTY Alive]
```

### 5.2 串口热插拔

```mermaid
flowchart TD
    A[udev Event] --> B{Serial Added?}
    B -->|Yes| C[Scan New Port]
    C --> D[Create Handler]
    D --> E[Broadcast to Clients]
    B -->|No| F{Serial Removed?}
    F -->|Yes| G[Remove Handler]
    G --> H[Notify Clients]
    F -->|No| I[Ignore]
```

---

**Why:** 定义清晰的交互流程和状态机，便于开发和测试
**How to apply:** 开发时遵循时序图，测试用例覆盖所有状态转换