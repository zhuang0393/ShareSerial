# ShareSerial - 远程串口共享工具

跨平台、零配置的串口共享系统。通过网络将物理串口虚拟化到多台远程机器，实现多人同时读取 Log、有序写入命令。

## 核心特性

- **无感透明**：虚拟串口表现为标准 `/dev/ttyXXX`，兼容现有工具
- **多人读取**：服务端 One-to-Many 广播，单客户端卡顿不影响其他
- **写入仲裁**：独占模式写锁，防止多人同时输入乱码
- **低延迟**：测试延迟 < 100µs（目标 < 10ms）
- **CLI 支持**：AI 可调用的命令行接口，支持 JSON 输出
- **配置文件**：YAML 配置支持，便于部署管理
- **systemd 服务**：支持系统服务安装，自动重启
- **跨平台**：支持 Linux 和 Windows

## 快速开始

### 安装

```bash
# 构建
make build

# 或单独构建
make build-server  # 服务端
make build-client  # 客户端
make build-cli     # CLI 工具

# 安装到系统
make install
```

### 服务端

```bash
# 方式 1: 命令行参数
./bin/shareserial-server --serial /dev/ttyUSB0 --port 7700

# 方式 2: 配置文件
./bin/shareserial-server --config configs/server.yaml

# 方式 3: systemd 服务
make install-systemd
sudo systemctl start shareserial-server
```

### 客户端（Linux）

```bash
# 方式 1: 命令行参数
./bin/shareserial-client --server 192.168.1.100:7700

# 方式 2: 配置文件
./bin/shareserial-client --config configs/client.yaml

# 使用虚拟串口
minicom -D /dev/vttyShare0
picocom /dev/vttyShare0
```

### 客户端（Windows）

```bash
# 构建 Windows 版本
make build-client-windows

# 运行
shareserial-client-windows.exe --server 192.168.1.100:7700 --local-port 8888

# 或使用配置文件
shareserial-client-windows.exe --config client-windows.yaml
```

**连接方式**：

Windows 客户端不创建虚拟串口，而是提供本地 TCP 端口转发。用户通过连接 localhost:8888 访问远程串口。

**Putty 连接**：
1. Connection type: **Raw**
2. Host Name: **localhost**
3. Port: **8888**
4. 点击 **Open**

**Python 脚本**：
```python
import socket
s = socket.socket()
s.connect(('localhost', 8888))
while True:
    data = s.recv(1024)
    print(data.decode(), end='')
```

### CLI 工具

```bash
# 获取 Log
./bin/shareserial log --server 192.168.1.100:7700

# 过滤 ERROR
./bin/shareserial log --filter ERROR --since 5m

# JSON 格式（便于 AI 解析）
./bin/shareserial log --format json

# 发送命令
./bin/shareserial send --command reboot

# 查看状态
./bin/shareserial status
```

## 配置文件

### 服务端配置 (configs/server.yaml)

```yaml
serial:
  path: "/dev/ttyUSB0"
  baudrate: 115200  # 固定波特率

server:
  port: 7700
  address: "0.0.0.0"

arbiter:
  timeout: 30  # 写锁超时（秒）
```

### 客户端配置 (configs/client.yaml)

```yaml
server:
  address: "192.168.1.100"
  port: 7700

pty:
  path: "/dev/vttyShare0"

reconnect:
  enabled: true
  interval: 5  # 重连间隔（秒）
  max_retry: 10
```

### 配置优先级

```
命令行参数 > 配置文件 > 默认值
```

## systemd 服务

### 安装

```bash
# 安装服务
make install-systemd

# 启动服务
sudo systemctl start shareserial-server

# 查看状态
sudo systemctl status shareserial-server

# 停止服务
sudo systemctl stop shareserial-server
```

### 配置

systemd 服务使用 `/etc/shareserial/server.yaml` 配置文件。

```bash
# 编辑配置
sudo vim /etc/shareserial/server.yaml

# 重启服务使配置生效
sudo systemctl restart shareserial-server
```

## 部署脚本

```bash
# 一键部署 Server
./scripts/deploy.sh server

# 一键部署 Client
./scripts/deploy.sh client 192.168.1.100

# 本地测试
./scripts/deploy.sh test

# 串口验证
./scripts/verify-serial.sh

# 稳定性测试
./scripts/stability-test.sh          # 24 小时
./scripts/stability-test.sh 1h       # 1 小时
```

## 项目结构

```
shareserial/
├── cmd/
│   ├── server/          # 服务端入口
│   ├── client/          # 客户端入口（Linux）
│   ├── client-windows/  # 客户端入口（Windows）
│   └── cli/             # CLI 工具
├── pkg/
│   ├── arbiter/         # 写锁仲裁
│   ├── serial/          # 串口操作
│   └── logparser/       # Log 解析
├── internal/
│   ├── server/          # TCP 服务器
│   ├── broadcast/       # 数据广播
│   ├── pty/             # PTY 虚拟串口（Linux）
│   ├── localproxy/      # TCP 端口转发（Windows）
│   ├── reconnect/       # 断线重连
│   └── config/          # 配置解析
├── tests/
│   ├── e2e/             # 端到端测试
│   └── simulation/      # 仿真测试
├── configs/             # 配置示例
├── scripts/             # 部署脚本
└── bin/                 # 可执行文件
```

## 测试

```bash
# 运行所有测试
make test

# 端到端测试
go test -v ./tests/e2e/...

# 性能测试
go test -v ./tests/e2e/... -run TestE2EPerformance
```

### 测试统计

| 模块 | 测试数量 | 状态 |
|------|----------|------|
| Config | 12 | PASS |
| CLI | 8 | PASS |
| Server | 7 | PASS |
| Broadcast | 7 | PASS |
| PTY | 5 | PASS |
| Reconnect | 4 | PASS |
| Arbiter | 8 | PASS |
| LogParser | 9 | PASS |
| Serial | 7 | PASS |
| E2E | 6 | PASS |
| **总计** | **64** | **PASS** |

## AI 调用支持

ShareSerial 提供 CLI 和 Skill 封装，便于 AI Agent 调用：

### CLI 命令

```bash
shareserial log [--filter] [--since] [--format json]
shareserial send --command [--timeout]
shareserial status
```

### Skill 文件

- `.claude/skills/shareserial-log.md` - Log 获取
- `.claude/skills/shareserial-send.md` - 命令发送
- `.claude/skills/shareserial-status.md` - 状态查看

### JSON 输出示例

```json
[
  {
    "timestamp": "17:30:00",
    "level": "INFO",
    "message": "System starting",
    "raw": "[17:30:00] INFO: System starting"
  }
]
```

## 性能指标

| 指标 | 目标 | 实测 |
|------|------|------|
| 网络延迟 | < 10ms | < 100µs |
| 波特率 | 115200 | 固定 |
| 并发客户端 | ≥ 5 | 已测试 5 |
| 测试覆盖 | 高 | 64 个测试 |

## 技术栈

- **语言**: Go 1.18+
- **协议**: TCP Raw Data（简化版）
- **虚拟串口**: PTY + symlink（Linux）
- **配置**: YAML (gopkg.in/yaml.v3)
- **服务管理**: systemd
- **仲裁模式**: 独占模式写锁

## Phase 功能清单

### Phase 1 (Linux)

- ✅ TCP 服务器
- ✅ 串口处理器（Mock + 真实）
- ✅ 写入仲裁器
- ✅ 数据广播器
- ✅ PTY 虚拟串口
- ✅ Log 解析器
- ✅ CLI 命令接口
- ✅ 断线重连
- ✅ 配置文件解析
- ✅ systemd 服务支持
- ✅ 部署脚本
- ✅ 端到端测试
- ✅ 仿真测试环境

### Phase 2 (Windows Client)

- ✅ 本地 TCP 端口转发
- ✅ Windows 客户端 CLI
- ✅ 自动重连
- ✅ 跨平台编译
- 🔄 系统托盘（可选）
- 🔄 com0com 虚拟串口（可选）

### Phase 3 (Windows Server)

- ✅ Windows 串口实现（go.bug.st/serial）
- ✅ COM 端口扫描器
- ✅ Windows 服务端入口
- ✅ 配置文件支持
- ✅ 跨平台编译
- ✅ CI/CD 支持
- 🔄 Windows 服务管理（可选）

## Windows 服务端使用说明

### 安装

```bash
# 构建 Windows 服务端
make build-server-windows

# 或构建所有 Windows 版本
make build-all-windows
```

### 使用

```cmd
# 扫描可用串口
shareserial-server-windows.exe --scan

# 启动服务端
shareserial-server-windows.exe --serial COM1 --port 7700

# 使用配置文件
shareserial-server-windows.exe --config configs/server-windows.yaml
```

### 注意事项

1. **权限**：访问 COM 端口可能需要管理员权限
2. **驱动**：确保已安装正确的串口驱动（如 FTDI、CH340）
3. **端口名称**：Windows 串口为 `COM1`, `COM2` 等，而非 `/dev/ttyUSB0`

## Makefile 命令

```bash
make build              # 构建所有 Linux 版本
make build-client-windows  # 构建 Windows 客户端
make test               # 运行测试
make package            # 打包发布
make package-windows    # 打包 Windows 版本
make release            # 完整发布（构建+测试+打包）
make install            # 安装到系统
make install-systemd    # 安装 systemd 服务
make uninstall          # 卸载
make stability-test     # 运行稳定性测试
make simulation-test    # 运行仿真测试
```

## License

MIT License

---

*Built with TDD + Musk's 5-Step Workflow*

*Last updated: 2026-05-30*