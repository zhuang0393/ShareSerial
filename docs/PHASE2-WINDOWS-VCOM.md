# Phase 2: Windows 虚拟 COM 口实现

## 方案概述

使用 **com0com** 开源虚拟串口驱动，在 Windows 上创建虚拟 COM 口，让 MobaXterm 等串口工具可以像使用物理串口一样连接。

---

## 技术架构

```
Ubuntu Server (物理串口 /dev/ttyUSB0)
    ↓ TCP :7700
Windows Client (TCP localhost:8888)
    ↓
com0com 虚拟串口桥接
    ↓
虚拟 COM 口 (COM4)
    ↓
MobaXterm 串口配置 (COM4, 115200)
```

---

## com0com 简介

com0com 是一个开源的 Windows 虚拟串口驱动，可以创建虚拟串口对。

### 核心功能

- 创建虚拟串口对（如 COM3 ↔ COM4）
- 支持 TCP/UDP 桥接模式
- 完全模拟真实串口行为

### 下载地址

- 官方：https://sourceforge.net/projects/com0com/
- 替代：https://github.com/fredowen/com0com

---

## 实现方案

### 方案 A：com0com 桥接模式（推荐）

使用 com0com 的 TCP 桥接功能，直接将 TCP 端口映射到 COM 口：

1. 安装 com0com
2. 创建 TCP 桥接
3. MobaXterm 连接虚拟 COM 口

### 方案 B：虚拟串口对模式

创建两个虚拟 COM 口，一个给 Client 使用，一个给 MobaXterm：

1. 安装 com0com
2. 创建串口对 COM3 ↔ COM4
3. Client 连接 COM3
4. MobaXterm 连接 COM4

---

## 部署步骤

### Step 1：安装 com0com

```powershell
# 下载 com0com
# https://sourceforge.net/projects/com0com/files/latest/download

# 安装（需要管理员权限）
# 运行 setup.exe，选择安装路径

# 验证安装
# 安装后在设备管理器可以看到虚拟串口
```

### Step 2：配置 com0com

**方法 A：使用 GUI 配置工具**

安装后运行 `setupc.exe`（命令行配置工具）：

```cmd
# 创建虚拟串口对 COM3-COM4
setupc install PortName=COM3 PortName=COM4

# 或创建 TCP 桥接（将 TCP 8888 映射到 COM4）
setupc install PortName=COM4,Tcp=127.0.0.1:8888
```

**方法 B：使用命令行脚本**

创建配置脚本 `install-vcom.cmd`：

```cmd
@echo off
REM 安装虚拟串口驱动配置

REM 创建 TCP 桥接：将 localhost:8888 映射到 COM4
setupc install PortName=COM4,Tcp=127.0.0.1:8888

REM 验证
setupc list

echo 虚拟串口 COM4 已创建，连接到 localhost:8888
echo MobaXterm 可以使用：COM4, 115200 baud
```

### Step 3：启动 Windows Client

```powershell
# 启动 Client（本地代理端口 8888）
.\shareserial-client-windows.exe --server 192.168.246.17:7700 --local-port 8888
```

### Step 4：MobaXterm 配置

```
Session type: Serial
Serial port:  COM4
Speed:        115200
Data bits:    8
Stop bits:    1
Parity:       None
Flow control: None
```

---

## 自动化脚本

创建一键部署脚本：

---

## 文件清单

1. `install-vcom.cmd` - 安装虚拟串口配置
2. `start-with-vcom.cmd` - 启动 Client + 虚拟串口
3. `uninstall-vcom.cmd` - 删除虚拟串口配置

---

## 常见问题

### Q1: com0com 安装失败

需要管理员权限，右键以管理员运行 setup.exe

### Q2: COM 口号冲突

使用 `mode` 命令检查现有 COM 口：
```cmd
mode
```

选择未被占用的 COM 口号（如 COM10、COM11）

### Q3: MobaXterm 无法连接 COM 口

检查设备管理器中是否有虚拟串口：
- 设备管理器 → 端口 (COM 和 LPT)
- 应看到 "com0com - serial port emulator"

---

## 替代方案

如果不想安装驱动，可以使用：

### HW VSP3 (Virtual Serial Port)

- 支持 TCP 转 COM
- 免费，无需驱动
- https://www.hw-group.com/products/hw-vsp3

### Virtual Serial Port Driver (VSPD)

- 商业软件，功能更强大
- https://www.eltima.com/products/vspd/

---

*Created: 2026-06-04*