---
name: go-dependencies-cn
description: Go 依赖下载问题解决（国内网络环境）
metadata: 
  node_type: memory
  type: reference
  originSessionId: 3ffbc492-3e30-4cde-9ed1-1e7c10e8ebd1
---

# Go 依赖下载问题解决（国内）

## 常见问题

### 1. GOPROXY 设置

```bash
# 国内代理
go env -w GOPROXY=https://goproxy.cn,direct

# 或使用七牛云
go env -w GOPROXY=https://goproxy.io,direct

# 查看当前设置
go env GOPROXY
```

### 2. 磁盘空间不足

Go 默认将模块缓存到 `$HOME/go`，可能占满用户目录。

**解决方案：**

```bash
# 指定缓存目录到 /tmp
go env -w GOMODCACHE=/tmp/go-mod
mkdir -p /tmp/go-mod

# 或指定到项目目录
export GOMODCACHE=/workspace/hengzhuang.jin/go-mod
```

### 3. 特定库无法下载

某些库（如 `go.bug.org/serial`）在国内无法访问。

**解决方案：使用 replace 指令**

```bash
# 1. 从 GitHub 克隆替代源
git clone https://github.com/bugst/go-serial.git /path/to/go-serial

# 2. 在 go.mod 中添加 replace
replace go.bug.st/serial => /path/to/go-serial

# 3. 注意模块名可能不同
# 检查克隆库的 go.mod 确认模块名
```

### 4. 离线依赖

在有网络的机器下载后拷贝：

```bash
# 下载依赖
go mod download

# 打包缓存
tar -czvf go-mod-cache.tar.gz $(go env GOMODCACHE)

# 拷贝到目标机器，解压到 GOMODCACHE 目录
```

---

## 清理用户目录

```bash
# 查看占用
du -sh ~.[!.]* * | sort -rh

# 清理缓存（安全）
rm -rf ~/.cache/pip ~/.cache/go-build ~/.npm/_cacache

# 清理开发工具缓存（如不常用）
rm -rf ~/.cargo ~/.bun ~/.nvm ~/.local/lib/python3.12
```

---

## 相关命令

```bash
# 查看依赖
go list -m all

# 查看模块缓存路径
go env GOMODCACHE

# 清理模块缓存
go clean -modcache

# 验证依赖
go mod verify
```