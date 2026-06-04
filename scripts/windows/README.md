# ShareSerial Windows 虚拟 COM 口工具

## 文件说明

| 文件 | 功能 |
|------|------|
| `install-vcom.cmd` | 安装虚拟 COM 口（需要管理员权限） |
| `start-with-vcom.cmd` | 启动 Client（自动检测虚拟 COM 口） |
| `uninstall-vcom.cmd` | 删除虚拟 COM 口配置 |

---

## 使用步骤

### Step 1：安装 com0com 驱动

1. 下载 com0com：https://sourceforge.net/projects/com0com/
2. 运行 `setup.exe` 安装（需要管理员权限）
3. 安装完成后重启电脑（可选）

### Step 2：创建虚拟 COM 口

以管理员身份运行：

```cmd
install-vcom.cmd
```

输出示例：
```
========================================
ShareSerial Phase 2 - 虚拟 COM 口安装
========================================

[OK] 管理员权限已确认
[OK] com0com 已安装: C:\Program Files\com0com
[OK] 可用 COM 口: COM4
[OK] 虚拟串口已创建

========================================
安装完成！
========================================

虚拟串口: COM4
TCP 桥接: localhost:8888
```

### Step 3：启动 Client

```cmd
start-with-vcom.cmd
```

或手动启动：
```cmd
shareserial-client-windows.exe --server 192.168.246.17:7700 --local-port 8888
```

### Step 4：MobaXterm 连接

```
Session type: Serial
Serial port:  COM4
Speed:        115200
Data bits:    8
Stop bits:    1
Parity:       None
```

---

## 一键启动示例

```cmd
REM 使用默认服务器 192.168.246.17:7700
start-with-vcom.cmd

REM 指定服务器 IP
start-with-vcom.cmd 192.168.1.100

REM 指定服务器 IP 和本地端口
start-with-vcom.cmd 192.168.1.100 9999
```

---

## 卸载

```cmd
REM 以管理员身份运行
uninstall-vcom.cmd
```

---

## 工作原理

```
┌──────────────────────────────────────────────────────────────┐
│  Ubuntu (192.168.246.17)                                     │
│                                                              │
│  /dev/ttyUSB2 ── Server (:7700)                              │
│                                                              │
└────────────────────┬─────────────────────────────────────────┘
                     │ TCP
                     │
┌────────────────────┴─────────────────────────────────────────┐
│  Windows PC                                                 │
│                                                             │
│  ShareSerial Client                                         │
│  TCP localhost:8888                                         │
│       ↓                                                     │
│  com0com TCP 桥接                                            │
│       ↓                                                     │
│  虚拟 COM4                                                   │
│       ↓                                                     │
│  MobaXterm (Serial 配置)                                     │
│                                                             │
└──────────────────────────────────────────────────────────────┘
```

---

## 常见问题

### Q1: install-vcom.cmd 报错"需要管理员权限"

右键点击脚本 → **以管理员身份运行**

### Q2: 找不到可用的 COM 口

检查现有 COM 口：
```cmd
mode
```

可能需要卸载其他虚拟串口软件。

### Q3: com0com 安装失败

1. 确保以管理员运行 setup.exe
2. 检查 Windows 版本兼容性
3. 尝试重启后重新安装

### Q4: MobaXterm 无法打开 COM 口

检查设备管理器：
- 设备管理器 → 端口 (COM 和 LPT)
- 应看到 "com0com - serial port emulator (COM4)"

---

## 替代方案

如果不想使用 com0com，可以使用：

| 软件 | 说明 |
|------|------|
| **HW VSP3** | 免费，TCP 转 COM，无需驱动 |
| **VSPD** | 商业软件，功能更强大 |
| **Python 脚本** | 使用 pyserial 实现 TCP 转 COM |

---

*Created: 2026-06-04*