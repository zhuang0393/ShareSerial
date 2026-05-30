# ShareSerial 简化部署指南（Mock 测试版）

## 一键部署

### Server 端（Ubuntu）

```bash
# 下载项目
cd /workspace/hengzhuang.jin/ss

# 构建服务端
export PATH=/tmp/go/bin:$PATH
make build-server

# 启动服务端（Mock 模式，无需真实串口）
./bin/shareserial-server --port 7700
```

### Client 端（Linux 编译服务器）

```bash
# 构建客户端
make build-client

# 连接服务端（替换 IP）
./bin/shareserial-client --server <SERVER_IP>:7700

# 使用虚拟串口
minicom -D /dev/vttyShare0
```

### CLI 工具（AI 调用）

```bash
# 获取 Log
./bin/shareserial log --server <SERVER_IP>:7700

# 过滤 ERROR
./bin/shareserial log --filter ERROR

# JSON 格式
./bin/shareserial log --format json
```

---

## 真实串口部署（需要下载依赖）

在**有网络的环境**中执行：

```bash
# 下载依赖
go mod download

# 构建
make build

# 启动真实串口
./bin/shareserial-server --serial /dev/ttyUSB0 --port 7700
```

---

## 功能验证（Mock 模式）

当前版本可以验证：
- ✅ TCP 连接和数据传输
- ✅ 多客户端广播
- ✅ 写锁仲裁
- ✅ CLI 命令接口
- ✅ 断线重连

无法验证：
- ❌ 真实串口读写（需 go.bug.org/serial）
- ❌ 真实 PTY（需 golang.org/x/sys/unix）