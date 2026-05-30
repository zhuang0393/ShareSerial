# ShareSerial 部署指南

## 目录

1. [快速部署](#快速部署)
2. [配置文件部署](#配置文件部署)
3. [systemd 服务部署](#systemd-服务部署)
4. [一键部署脚本](#一键部署脚本)
5. [测试验证](#测试验证)

---

## 快速部署

### Server 端（Ubuntu + 物理串口）

```bash
# 1. 构建
make build-server

# 2. 启动
./bin/shareserial-server --serial /dev/ttyUSB0 --port 7700
```

### Client 端（远程机器）

```bash
# 1. 构建
make build-client

# 2. 连接
./bin/shareserial-client --server 192.168.1.100:7700

# 3. 使用虚拟串口
minicom -D /dev/vttyShare0
```

---

## 配置文件部署

### 准备配置文件

```bash
# Server 配置
cat > configs/server.yaml << EOF
serial:
  path: "/dev/ttyUSB0"
  baudrate: 115200

server:
  port: 7700
  address: "0.0.0.0"

arbiter:
  timeout: 30
EOF

# Client 配置
cat > configs/client.yaml << EOF
server:
  address: "192.168.1.100"
  port: 7700

pty:
  path: "/dev/vttyShare0"

reconnect:
  enabled: true
  interval: 5
  max_retry: 10
EOF
```

### 使用配置文件启动

```bash
# Server
./bin/shareserial-server --config configs/server.yaml

# Client
./bin/shareserial-client --config configs/client.yaml
```

### 配置优先级

```
命令行参数 > 配置文件 > 默认值
```

示例：
```bash
# 配置文件 + 参数覆盖
./bin/shareserial-server --config configs/server.yaml --port 8000
```

---

## systemd 服务部署

### 安装

```bash
# 1. 安装到系统
make install-systemd

# 2. 编辑配置（可选）
sudo vim /etc/shareserial/server.yaml

# 3. 启动服务
sudo systemctl start shareserial-server

# 4. 查看状态
sudo systemctl status shareserial-server

# 5. 启用开机启动
sudo systemctl enable shareserial-server
```

### 管理

```bash
# 停止
sudo systemctl stop shareserial-server

# 重启
sudo systemctl restart shareserial-server

# 查看日志
sudo journalctl -u shareserial-server -f
```

### 卸载

```bash
make uninstall-systemd
```

### 服务文件说明

服务文件位于 `/etc/systemd/system/shareserial-server.service`：

```ini
[Unit]
Description=ShareSerial Server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/shareserial-server --config /etc/shareserial/server.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

---

## 一键部署脚本

### deploy.sh 命令

```bash
# 查看帮助
./scripts/deploy.sh help

# Server 端部署
./scripts/deploy.sh server                     # 自动检测串口
./scripts/deploy.sh server /dev/ttyUSB0        # 指定串口
./scripts/deploy.sh server /dev/ttyUSB0 ./configs/server.yaml

# Client 端部署
./scripts/deploy.sh client 192.168.1.100
./scripts/deploy.sh client 192.168.1.100 ./configs/client.yaml

# 本地测试
./scripts/deploy.sh test

# 构建
./scripts/deploy.sh build

# 打包
./scripts/deploy.sh package

# 安装/卸载
./scripts/deploy.sh install
./scripts/deploy.sh uninstall
```

---

## 测试验证

### 串口验证

```bash
# 检测串口设备
./scripts/verify-serial.sh

# 输出示例：
# 检测到串口设备: /dev/ttyUSB0
# 权限: crw-rw-rw- root dialout
# 用户在 dialout 组中
# 串口可正常访问
```

### 连接测试

```bash
# 本地测试（Server + Client 同机）
./scripts/deploy.sh test
```

### 稳定性测试

```bash
# 24 小时稳定性测试
./scripts/stability-test.sh

# 自定义时长
./scripts/stability-test.sh 1h    # 1 小时
./scripts/stability-test.sh 30m   # 30 分钟

# 查看报告
cat stability-reports/report.txt
```

---

## 权限问题

### 串口权限

```bash
# 检查用户组
groups

# 添加到 dialout 组
sudo usermod -aG dialout $USER

# 重新登录生效
logout

# 或临时权限
sudo chmod 666 /dev/ttyUSB0
```

### PTY 权限

虚拟串口 `/dev/vttyShare0` 由客户端创建，当前用户有权限。

---

## 网络配置

### 端口

默认端口 `7700`，可配置：

```yaml
server:
  port: 7700
```

### 防火墙

```bash
# Ubuntu
sudo ufw allow 7700/tcp

# CentOS
sudo firewall-cmd --add-port=7700/tcp --permanent
sudo firewall-cmd --reload
```

---

## 打包发布

```bash
# 打包
make package

# 或完整发布（构建+测试+打包）
make release

# 发布文件
ls -la shareserial-1.0.0.tar.gz
```

---

## 故障排查

| 问题 | 解决方案 |
|------|----------|
| 串口无权限 | `sudo usermod -aG dialout $USER` |
| 端口被占用 | `netstat -tuln | grep 7700` |
| 连接失败 | 检查 Server IP、防火墙 |
| PTY 创建失败 | 检查 /tmp 权限 |
| systemd 启动失败 | `journalctl -u shareserial-server` |

---

## 部署检查清单

### Server 端

- [ ] 串口设备已连接 (`ls /dev/ttyUSB*`)
- [ ] 用户在 dialout 组 (`groups`)
- [ ] 端口未被占用 (`netstat -tuln | grep 7700`)
- [ ] 配置文件已准备 (`configs/server.yaml`)
- [ ] 服务已启动 (`systemctl status`)

### Client 端

- [ ] Server IP 可达 (`ping`)
- [ ] 端口可连接 (`nc -zv IP 7700`)
- [ ] 配置文件已准备 (`configs/client.yaml`)
- [ ] 虚拟串口已创建 (`ls /dev/vttyShare0`)

---

*Last updated: 2026-05-30*