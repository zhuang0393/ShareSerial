# ShareSerial Complete Deploy Skill

## Description

ShareSerial 完整部署流程：打包、传输、Server/Client 端部署、验证。

## Trigger

当用户提到以下内容时触发：
- "部署 ShareSerial"
- "shareserial 部署 server/client"
- "打包 shareserial"
- "串口共享部署"

## Usage

```bash
/skill shareserial-deploy
```

## Deployment Flow

### Step 1: 打包项目（当前机器）

```bash
# 打包所有必要文件
./scripts/package.sh

# 生成 shareserial-v1.0.0.tar.gz
```

### Step 2: 传输到目标机器

```bash
# SCP 传输
scp shareserial-v1.0.0.tar.gz user@server-ip:/tmp/

# 或使用其他方式传输
```

### Step 3: Server 端部署（目标机器）

```bash
# 解压
cd /tmp
tar -xzvf shareserial-v1.0.0.tar.gz

# 进入目录
cd shareserial-v1.0.0

# 一键部署 Server
./scripts/deploy.sh server
```

### Step 4: Client 端部署（当前机器）

```bash
# 连接 Server
./scripts/deploy.sh client <SERVER_IP>

# 使用虚拟串口
minicom -D /dev/vttyShare0
```

## Prerequisites

### Server 端要求
- Ubuntu Linux
- 物理串口设备（/dev/ttyUSB0 或 /dev/ttyACM0）
- 用户在 dialout 组：`sudo usermod -aG dialout $USER`

### Client 端要求
- Linux
- 网络可访问 Server

## Common Issues

| 问题 | 解决方案 |
|------|----------|
| 串口权限不足 | `sudo usermod -aG dialout $USER` 后重新登录 |
| 未检测到串口 | `ls -la /dev/ttyUSB*` 检查设备 |
| 连接失败 | 检查 Server IP、端口、防火墙 |
| Go 未安装 | 使用打包的 bin/ 目录，无需重新构建 |

## Verification

```bash
# Server 端验证
./bin/shareserial-server --help

# Client 端验证
./bin/shareserial status --server <IP>:7700

# CLI 测试
./bin/shareserial log --server <IP>:7700 --since 1m
```

## Related Skills

- `shareserial-log` - 获取远程 Log
- `shareserial-send` - 发送命令
- `shareserial-status` - 查看状态