# ShareSerial Ubuntu 部署指南

## 场景：同一台机器部署 Server + Client

本指南介绍如何在同一台 Ubuntu 机器上同时部署 Server 和 Client，实现本地串口共享测试。

---

## 步骤 1：安装必要工具

```bash
# 安装 minicom（串口终端工具）
sudo apt update
sudo apt install minicom -y

# 安装串口工具（可选，用于调试）
sudo apt install setserial serialtest -y
```

---

## 步骤 2：配置串口权限

```bash
# 将当前用户加入 dialout 组（串口访问权限）
sudo usermod -a -G dialout $USER

# 重新登录使权限生效，或临时使用 newgrp
newgrp dialout

# 验证权限
groups $USER
```

---

## 步骤 3：检查物理串口

```bash
# 查看可用串口设备
ls -la /dev/ttyUSB* /dev/ttyACM* /dev/ttyS*

# 查看串口详情
dmesg | grep tty

# 测试串口是否可访问
cat /dev/ttyUSB0  # 如果有数据会显示
```

常见串口设备：
- `/dev/ttyUSB0` - USB 转串口设备
- `/dev/ttyACM0` - USB ACM 设备（如 Arduino）
- `/dev/ttyS0` - 传统串口

---

## 步骤 4：启动 Server

```bash
# 进入 release 目录（或 bin 目录）
cd /path/to/shareserial/release

# 解压 Linux 版本（如果使用 release 包）
tar -xzf shareserial-1.0.0-linux-x86_64.tar.gz
cd linux

# 方法 A：使用默认配置
./shareserial-server --serial /dev/ttyUSB0 --port 7700

# 方法 B：使用配置文件
./shareserial-server --config /path/to/server.yaml
```

配置文件示例 (`server.yaml`)：
```yaml
serial:
  path: /dev/ttyUSB0
  baudrate: 115200
  
server:
  port: 7700
  host: 0.0.0.0
  
arbiter:
  timeout: 30  # 写锁超时秒数
```

---

## 步骤 5：启动 Client

```bash
# 在另一个终端窗口启动 Client

# 方法 A：连接本机 Server
./shareserial-client --server 127.0.0.1:7700 --pty /dev/ttyShare0

# 方法 B：使用配置文件
./shareserial-client --config /path/to/client.yaml
```

配置文件示例 (`client.yaml`)：
```yaml
server:
  address: 127.0.0.1:7700
  
pty:
  path: /dev/ttyShare0
  
reconnect:
  enabled: true
  interval: 5
  max_retry: 0  # 0 表示无限重试
```

**注意**：
- Client 会创建 PTY 设备并生成 symlink（如 `/dev/ttyShare0`）
- symlink 实际指向 PTY slave（如 `/dev/pts/4`）
- 多个 Client 可以使用不同的 symlink 名称（如 `/dev/ttyShare1`）

---

## 步骤 6：使用 minicom 访问虚拟串口

```bash
# 方法 A：直接指定设备
sudo minicom -D /dev/ttyShare0

# 方法 B：配置 minicom
sudo minicom -s
```

minicom 配置步骤：
1. 选择 "Serial port setup"
2. 设置：
   - A - Serial Device: `/dev/ttyShare0`
   - E - Bps/Par/Bits: `115200 8N1`（与物理串口一致）
   - F - Hardware Flow Control: `No`
   - G - Software Flow Control: `No`
3. 选择 "Save setup as dfl" 保存默认配置
4. 选择 "Exit" 进入终端

**常用 minicom 命令**：
- `Ctrl+A Z` - 帮助菜单
- `Ctrl+A X` - 退出
- `Ctrl+A C` - 清屏
- `Ctrl+A W` - 自动换行

---

## 步骤 7：验证连接状态

```bash
# 使用 CLI 工具检查状态
./shareserial status --server 127.0.0.1:7700

# 输出示例：
# {
#   "connected": true,
#   "server": "127.0.0.1:7700",
#   "clients": 1,
#   "serial_port": "/dev/ttyUSB0",
#   "write_lock": {
#     "locked": false,
#     "owner": null
#   }
# }
```

---

## 步骤 8：发送命令（写锁管理）

```bash
# 使用 CLI 工具发送命令（自动获取写锁）
./shareserial send --command "ls" --server 127.0.0.1:7700

# 在 minicom 中观察输出
```

---

## 步骤 9：多 Client 测试（可选）

```bash
# 启动第二个 Client
./shareserial-client --server 127.0.0.1:7700 --pty /dev/ttyShare1

# 第二个终端使用 minicom
sudo minicom -D /dev/ttyShare1

# 两个 Client 都能看到相同的日志输出
# 写锁确保只有一个 Client 能输入
```

---

## 完整部署脚本

```bash
#!/bin/bash
# deploy-local.sh

# 1. 检查串口设备
SERIAL_PORT=${1:-/dev/ttyUSB0}
if [ ! -e "$SERIAL_PORT" ]; then
    echo "错误: 串口设备 $SERIAL_PORT 不存在"
    exit 1
fi

# 2. 检查权限
if ! groups $USER | grep -q dialout; then
    echo "添加串口权限..."
    sudo usermod -a -G dialout $USER
    echo "请重新登录使权限生效"
    exit 0
fi

# 3. 启动 Server
echo "启动 Server..."
./shareserial-server --serial $SERIAL_PORT --port 7700 &
SERVER_PID=$!
sleep 2

# 4. 启动 Client
echo "启动 Client..."
./shareserial-client --server 127.0.0.1:7700 --pty /dev/ttyShare0 &
CLIENT_PID=$!
sleep 2

# 5. 检查状态
echo "检查连接状态..."
./shareserial status --server 127.0.0.1:7700

echo ""
echo "部署完成!"
echo "================"
echo "Server PID: $SERVER_PID"
echo "Client PID: $CLIENT_PID"
echo "虚拟串口: /dev/ttyShare0"
echo ""
echo "使用命令:"
echo "  sudo minicom -D /dev/ttyShare0"
echo ""
echo "停止服务:"
echo "  kill $SERVER_PID $CLIENT_PID"
```

---

## 常见问题

### Q1: 无法访问串口设备
```bash
# 检查权限
ls -la /dev/ttyUSB0

# 添加权限
sudo chmod 666 /dev/ttyUSB0
# 或永久方案
sudo usermod -a -G dialout $USER
```

### Q2: minicom 无法打开虚拟串口
```bash
# 检查 symlink 是否存在
ls -la /dev/ttyShare0

# 如果不存在，检查 Client 日志
./shareserial-client --server 127.0.0.1:7700 --pty /dev/ttyShare0 2>&1 | tee client.log
```

### Q3: Server 启动失败
```bash
# 检查串口是否被占用
sudo fuser /dev/ttyUSB0

# 如果被占用，终止占用进程
sudo fuser -k /dev/ttyUSB0
```

### Q4: 看不到数据输出
```bash
# 检查物理串口波特率
stty -F /dev/ttyUSB0

# 确保 Client 配置与物理串口一致
# 检查 Server 日志
./shareserial-server --serial /dev/ttyUSB0 --port 7700 2>&1 | tee server.log
```

---

## 清理脚本

```bash
#!/bin/bash
# cleanup.sh

# 停止所有 ShareSerial 进程
pkill -f shareserial-server
pkill -f shareserial-client

# 清理 symlink
sudo rm -f /dev/ttyShare*

echo "清理完成"
```

---

## 系统服务部署（可选）

如果需要长期运行，可以部署为 systemd 服务：

```bash
# Server 服务
sudo cp shareserial-server /usr/local/bin/
sudo cat > /etc/systemd/system/shareserial-server.service << EOF
[Unit]
Description=ShareSerial Server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/shareserial-server --serial /dev/ttyUSB0 --port 7700
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable shareserial-server
sudo systemctl start shareserial-server

# 查看状态
sudo systemctl status shareserial-server
```

---

*Created: 2026-06-03*