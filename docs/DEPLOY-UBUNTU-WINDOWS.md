# Ubuntu Server + Windows Client 部署教程

## 场景描述

- **Ubuntu 机器**：连接物理串口 `/dev/ttyUSB0`，部署 Server
- **Windows PC**：部署 Client，使用 MobaXterm 连接

---

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    Ubuntu 机器                               │
│                                                             │
│  物理串口 /dev/ttyUSB0                                       │
│       │                                                     │
│       ↓                                                     │
│  ShareSerial Server                                         │
│  监听端口 7700                                               │
│       │                                                     │
│       ↓ TCP                                                 │
│  网络连接                                                    │
└───────┬─────────────────────────────────────────────────────┘
        │
        │ 跨网络 TCP 连接
        │
┌───────┴─────────────────────────────────────────────────────┐
│                    Windows PC                                │
│                                                             │
│  ShareSerial Client                                         │
│  本地代理端口 8888                                           │
│       │                                                     │
│       ↓ TCP                                                 │
│  MobaXterm 连接 localhost:8888                              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 第一部分：Ubuntu Server 部署

### 1.1 准备环境

```bash
# 检查物理串口
ls -la /dev/ttyUSB*

# 添加串口权限
sudo usermod -a -G dialout $USER
# 重新登录使权限生效
```

### 1.2 获取软件

**方式 A：从 GitHub 下载**
```bash
# 下载 release 包
wget https://github.com/zhuang0393/ShareSerial/releases/download/v1.0.0/shareserial-1.0.0-linux-x86_64.tar.gz

# 解压
tar -xzf shareserial-1.0.0-linux-x86_64.tar.gz
cd linux/
```

**方式 B：从源码构建**
```bash
# 克隆仓库
git clone https://github.com/zhuang0393/ShareSerial.git
cd ShareSerial

# 构建
make build-server
```

### 1.3 启动 Server

```bash
# 直接启动
./shareserial-server --serial /dev/ttyUSB0 --port 7700

# 或使用配置文件
./shareserial-server --config configs/server.yaml
```

### 1.4 检查 Server 状态

```bash
# 查看 Server 日志
# 应显示：
#   ShareSerial Server started on [::]:7700
#   Serial port: /dev/ttyUSB0 (115200 baud)

# 测试端口监听
netstat -tlnp | grep 7700

# 查看 Ubuntu IP 地址
ip addr show | grep "inet " | grep -v 127.0.0.1
# 记录 IP 地址，如：192.168.1.100
```

### 1.5 防火墙配置（如果需要）

```bash
# Ubuntu 防火墙开放端口
sudo ufw allow 7700/tcp
sudo ufw status
```

---

## 第二部分：Windows Client 部署

### 2.1 获取软件

**从 GitHub 下载**
```
https://github.com/zhuang0393/ShareSerial/releases/download/v1.0.0/shareserial-1.0.0-windows-x86_64.zip
```

解压后得到：
- `shareserial-server-windows.exe`（Windows 服务端，本次不使用）
- `shareserial-client-windows.exe`（Windows 客户端）
- `shareserial-cli-windows.exe`（CLI 工具）

### 2.2 启动 Client

**方式 A：命令行启动**
```cmd
# 连接 Ubuntu Server（假设 Ubuntu IP 为 192.168.1.100）
shareserial-client-windows.exe --server 192.168.1.100:7700 --local-port 8888
```

**方式 B：使用配置文件**
```cmd
# 创建配置文件 client.yaml
server:
  address: "192.168.1.100"
  port: 7700

reconnect:
  enabled: true
  interval: 5
  max_retry: 0

# 启动
shareserial-client-windows.exe --config client.yaml --local-port 8888
```

### 2.3 检查 Client 状态

Client 启动后应显示：
```
ShareSerial Windows Client v1.0.0
Server address: 192.168.1.100:7700
Local TCP port: 8888
Connected to server: 192.168.1.100:7700
Local TCP proxy started: localhost:8888

=== Connect with Putty ===
  Connection type: Raw
  Host Name: localhost
  Port: 8888
```

---

## 第三部分：MobaXterm 连接配置

### 3.1 打开 MobaXterm

启动 MobaXterm，点击 **Session** → **New session**

### 3.2 配置连接

选择 **SSH** 或 **Telnet** 都不行，需要使用 **Raw** 连接：

| 设置项 | 值 |
|--------|-----|
| Session type | **Raw**（不是 SSH/Telnet） |
| Remote host | `localhost` 或 `127.0.0.1` |
| Port | `8888` |
| Username | 不填 |

### 3.3 MobaXterm 配置截图说明

```
┌────────────────────────────────────────┐
│  Session settings                      │
│                                        │
│  Session type: [Raw           ▼]      │  ← 选择 Raw
│                                        │
│  Remote host:  [localhost        ]    │  ← 输入 localhost
│  Port:         [8888              ]    │  ← 输入 8888
│                                        │
│  [OK]    [Cancel]                      │
└────────────────────────────────────────┘
```

### 3.4 连接测试

点击 **OK** 后，MobaXterm 会连接到 `localhost:8888`

此时：
- 你输入的任何字符会发送到 Ubuntu 物理串口
- 串口设备的输出会显示在 MobaXterm

### 3.5 验证输入输出

**测试步骤**：
1. 在 MobaXterm 中敲回车
2. 应立即显示 Android console 或设备响应
3. 输入命令（如 `ls`）
4. 应显示命令输出

---

## 第四部分：常见问题

### Q1: MobaXterm 无法连接

**检查步骤**：
```cmd
# Windows 检查端口监听
netstat -an | findstr 8888

# 应显示：
#   TCP    127.0.0.1:8888    0.0.0.0:0    LISTENING
```

如果端口未监听，说明 Client 未正常启动。

### Q2: Client 连接 Server 失败

**检查步骤**：
1. 确认 Ubuntu Server 正在运行
2. 确认 Ubuntu IP 地址正确
3. 确认防火墙已开放 7700 端口
4. 确认网络连通（ping 测试）

```cmd
# Windows ping Ubuntu
ping 192.168.1.100

# 测试 TCP 连接
telnet 192.168.1.100 7700
```

### Q3: 无法输入或无响应

**原因**：需要确保 Server 和 Client 都已应用最新修复

**检查**：
- Server 版本 >= v1.0.0（包含串口锁修复）
- Client 版本 >= v1.0.0

### Q4: Windows Client 没有 MobaXterm 显示

**确认**：
- MobaXterm 使用 **Raw** 连接（不是 SSH）
- 连接 `localhost:8888`（不是 Ubuntu IP）

---

## 第五部分：网络配置

### 5.1 局域网直连

最简单场景：Ubuntu 和 Windows 在同一局域网

```
Ubuntu IP: 192.168.1.100
Windows IP: 192.168.1.200

Windows Client 连接: 192.168.1.100:7700
```

### 5.2 跨网段连接

如果 Ubuntu 和 Windows 不在同一网段：
1. 确认路由配置
2. 确认防火墙规则
3. 考虑使用 VPN 或 SSH 隧道

### 5.3 SSH 隧道（可选）

如果网络受限，可以通过 SSH 隧道：

```cmd
# Windows 上建立 SSH 隧道
ssh -L 7700:localhost:7700 user@ubuntu-ip

# 然后 Client 连接本地隧道
shareserial-client-windows.exe --server localhost:7700 --local-port 8888
```

---

## 第六部分：配置文件模板

### 6.1 Ubuntu Server 配置

`server.yaml`:
```yaml
# Server configuration
server:
  host: "0.0.0.0"  # 监听所有网络接口
  port: 7700

# Serial port configuration
serial:
  path: "/dev/ttyUSB0"
  baudrate: 115200

# Arbiter configuration
arbiter:
  timeout: 30  # seconds
```

### 6.2 Windows Client 配置

`client.yaml`:
```yaml
# Server connection
server:
  address: "192.168.1.100"  # Ubuntu IP
  port: 7700

# Reconnection settings
reconnect:
  enabled: true
  interval: 5  # seconds
  max_retry: 0  # 0 = infinite retry
```

---

## 第七部分：部署脚本

### 7.1 Ubuntu Server 启动脚本

```bash
#!/bin/bash
# start-server.sh

SERIAL_PORT=${1:-/dev/ttyUSB0}
SERVER_PORT=${2:-7700}

./shareserial-server --serial "$SERIAL_PORT" --port "$SERVER_PORT"
```

### 7.2 Windows Client 启动脚本

```cmd
@echo off
REM start-client.bat

set SERVER_IP=192.168.1.100
set SERVER_PORT=7700
set LOCAL_PORT=8888

shareserial-client-windows.exe --server %SERVER_IP%:%SERVER_PORT% --local-port %LOCAL_PORT%
```

---

## 第八部分：验证清单

### Ubuntu Server 验证

| 检查项 | 命令 | 期望结果 |
|--------|------|----------|
| 串口设备 | `ls /dev/ttyUSB*` | 存在设备 |
| 权限 | `groups $USER` | 包含 dialout |
| Server 运行 | `ps aux | grep shareserial` | 进程存在 |
| 端口监听 | `netstat -tlnp | grep 7700` | LISTEN |
| IP 地址 | `ip addr show` | 记录 IP |

### Windows Client 验证

| 检查项 | 命令/操作 | 期望结果 |
|--------|----------|----------|
| Client 运行 | 查看窗口 | 显示 Connected |
| 本地端口 | `netstat -an | findstr 8888` | LISTENING |
| 网络连通 | `ping ubuntu-ip` | 正常 |
| MobaXterm | Session → Raw | 连接成功 |

---

*Created: 2026-06-03*