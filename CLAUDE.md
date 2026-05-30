# ShareSerial - 远程串口共享工具

## 项目概述

跨平台、零配置、无感体验的串口共享系统。通过网络将一台机器上的物理串口虚拟化到多台远程机器上，使多位工程师可以像使用本地物理串口一样，同时读取 Log 并有序进行 Shell 交互。

## 核心设计理念

1. **无感（Transparent）：** 客户端虚拟出的串口在系统底层表现为标准串口（`/dev/ttyXXX` / `COMX`），完美兼容既有串口终端和烧录工具
2. **便捷（Zero-Config）：** 支持 mDNS 局域网自动发现，客户端无需手动输入 IP
3. **有序（Ordered）：** 内置输入仲裁机制（独占模式），防止多人同时输入导致数据乱码

## 技术选型

| 维度 | 选择 |
|------|------|
| 语言 | Go |
| 协议 | RFC2217 (Telnet Serial) |
| 仲裁模式 | 独占模式（写锁超时释放） |
| 服务发现 | mDNS |
| UI框架 | 待定（Phase 2 Windows 客户端考虑） |

## 开发阶段

- **Phase 1（当前）：** Ubuntu 服务端 + Ubuntu 客户端
- **Phase 2：** Windows 客户端
- **Phase 3：** Windows 服务端

## 非功能性需求

- 延迟 < 5ms
- 单文件绿色版部署
- 支持高波特率（921600 / 1500000）
- 24小时稳定性测试通过

## 马斯克五步工作法

1. **质疑需求** - 删除不必要的流程和部件
2. **删除部分** - 简化系统，聚焦核心功能
3. **简化优化** - 在删减基础上优化
4. **加速迭代** - 加快开发周期
5. **自动化** - 最后再自动化

## SDD+TDD 开发流程

本项目采用 SDD+TDD 开发模式，详见 `.claude/specs/` 目录下的规格文档。

### 规格文档（六件套）

| 文档 | 说明 |
|------|------|
| PRD.md | 业务需求、用户故事 |
| SYSTEM_FLOW.md | 系统调用链、状态机 |
| TECH_STACK.md | 技术栈、依赖库 |
| CODING_STANDARDS.md | 编码规范 |
| ARCHITECTURE.md | 系统架构、模块划分 |
| IMPLEMENTATION.md | TDD 路线图、测试用例 |

### 三道门审查

1. **需求审查** - PRD.md + SYSTEM_FLOW.md
2. **设计审查** - ARCHITECTURE.md + 接口定义
3. **任务分解审查** - IMPLEMENTATION.md

## 项目结构

```
/workspace/hengzhuang.jin/ss/
├── .claude/
│   ├── rules/              # 规则文件
│   └── specs/              # 规格文档
│       ├── PRD.md
│       ├── SYSTEM_FLOW.md
│       ├── TECH_STACK.md
│       ├── CODING_STANDARDS.md
│       ├── ARCHITECTURE.md
│       └── IMPLEMENTATION.md
├── cmd/                    # 命令行入口
│   ├── server/             # 服务端入口
│   └── client/             # 客户端入口
├── pkg/                    # 核心包
│   ├── serial/             # 串口操作
│   ├── rfc2217/            # RFC2217 协议实现
│   ├── mdns/               # mDNS 服务发现
│   └── arbiter/            # 输入仲裁
├── internal/               # 内部实现
├── tests/                  # 测试用例
│   ├── integration/        # 集成测试
│   └── e2e/                # 端到端测试
├── go.mod
├── go.sum
├── Makefile
└── CLAUDE.md
```

## 快速开始

```bash
# 构建服务端
make build-server

# 构建客户端
make build-client

# 运行测试
make test
```

---

*Created: 2026-05-28*