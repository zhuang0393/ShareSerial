---
name: shareserial-deploy
description: ShareSerial 部署指南 - Server/Client 端部署、go-serial 依赖处理、常见问题解决
metadata: 
  node_type: memory
  type: reference
  originSessionId: 3ffbc492-3e30-4cde-9ed1-1e7c10e8ebd1
---

# ShareSerial 部署指南

## 项目概述

ShareSerial 是远程串口共享工具：
- Server 端：连接物理串口，通过网络共享
- Client 端：创建虚拟串口，远程访问

## 快速部署

### Client 端（编译服务器）

```bash
# 构建
make build-client

# 连接 Server
./bin/shareserial-client --server <SERVER_IP>:7700

# 使用虚拟串口
minicom -D /dev/vttyShare0
```

### Server 端（连接板子的机器）

需要拷贝项目或重新构建：

```bash
# 启动 Server
./bin/shareserial-server --serial /dev/ttyUSB0 --port 7700
```

---

## go-serial 依赖处理

### 问题：国内网络无法下载 go.bug.org/serial

**解决方案：使用 GitHub 克隆 + replace 指令**

```bash
# 1. 克隆到本地
git clone https://github.com/bugst/go-serial.git /workspace/hengzhuang.jin/go-serial

# 2. 修改 go.mod，添加 replace
# replace go.bug.st/serial => /workspace/hengzhuang.jin/go-serial

# 3. 更新导入路径
# go.bug.org/serial → go.bug.st/serial

# 4. 构建
go build ./cmd/server ./cmd/client ./cmd/cli
```

### 注意事项

- go-serial 的模块名是 `go.bug.st/serial`（不是 `go.bug.org`）
- 需要 golang.org/x/sys 依赖
- Go 版本要求 1.22+

---

## 常见问题

### 1. 磁盘空间不足

```bash
# 清理缓存
rm -rf ~/.cache/pip ~/.cache/go-build ~/.npm/_cacache
rm -rf ~/.cargo/registry ~/.cargo/git
rm -rf ~/.bun ~/.nvm  # 如不常用
```

### 2. Go 代理设置（国内）

```bash
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOMODCACHE=/tmp/go-mod  # 指定缓存目录
```

### 3. 串口权限

```bash
# 添加用户到 dialout 组
sudo usermod -aG dialout $USER

# 重新登录生效
```

### 4. 检测串口设备

```bash
ls -la /dev/ttyUSB* /dev/ttyACM*
```

---

## CLI 工具使用

```bash
# 获取 Log
./bin/shareserial log --server <IP>:7700

# 过滤 ERROR
./bin/shareserial log --filter ERROR --since 5m

# JSON 格式（AI 解析）
./bin/shareserial log --format json

# 发送命令
./bin/shareserial send --command reboot

# 查看状态
./bin/shareserial status
```

---

## 相关文件

- PRD: `.claude/specs/PRD.md`
- Architecture: `.claude/specs/ARCHITECTURE.md`
- Skills: `.claude/skills/shareserial-*.md`
- Memory: `memory/MEMORY.md`