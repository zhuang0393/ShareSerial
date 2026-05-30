# Go 项目构建规则

## 国内网络环境

### GOPROXY 设置（必须）

```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

### 磁盘空间管理

用户目录可能有限，指定模块缓存目录：

```bash
go env -w GOMODCACHE=/tmp/go-mod
mkdir -p /tmp/go-mod
```

### 无法下载的依赖处理

1. 尝试从 GitHub 克隆替代源
2. 使用 go.mod replace 指令指向本地路径
3. 检查克隆库的 go.mod 确认正确模块名

示例：
```go
// go.mod
replace go.bug.st/serial => /path/to/local/go-serial
```

## 构建命令

```bash
# 标准构建
make build

# 单独构建
go build ./cmd/server ./cmd/client ./cmd/cli

# 测试
make test
```

## 清理

```bash
# 清理缓存
rm -rf ~/.cache/pip ~/.cache/go-build ~/.npm/_cacache

# 清理构建产物
make clean
```

## 常见错误处理

| 错误 | 原因 | 解决方案 |
|------|------|----------|
| `go: command not found` | Go 未安装 | `apt install golang-go` 或使用 /tmp/go/bin/go |
| `no space left on device` | 用户目录满 | 设置 GOMODCACHE 到 /tmp |
| `unrecognized import path` | 网络无法访问 | 使用 replace 指令 |
| `redeclared in this block` | 重复定义 | 清理重复文件 |