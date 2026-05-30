# ShareSerial Memory Index

## Project Context

- [PRD](.claude/specs/PRD.md) — 产品需求文档，定义业务目标和用户故事（含 AI CLI 需求）
- [SYSTEM_FLOW](.claude/specs/SYSTEM_FLOW.md) — 系统调用流程、AI CLI 时序图、JSON 输出格式
- [TECH_STACK](.claude/specs/TECH_STACK.md) — 技术栈与依赖库定义（含 CLI 设计）
- [CODING_STANDARDS](.claude/specs/CODING_STANDARDS.md) — Go 编码规范与并发安全要求
- [ARCHITECTURE](.claude/specs/ARCHITECTURE.md) — 系统架构与模块划分（简化版）
- [IMPLEMENTATION](.claude/specs/IMPLEMENTATION.md) — TDD 路线图与测试用例清单（含 CLI 测试）

## Deployment & Operations

- [shareserial-deploy](shareserial-deploy.md) — 部署指南：Server/Client 端、go-serial 依赖处理
- [go-dependencies-cn](go-dependencies-cn.md) — Go 依赖下载问题解决（国内网络环境）

## Rules

- [Gerrit Commit Rules](.claude/rules/gerrit-commit-rules.md) — 提交信息格式规范
- [AI Usage Boundary](.claude/rules/ai-usage-boundary.md) — AI 辅助边界定义
- [SDD+TDD](.claude/rules/sdd-tdd.md) — SDD+TDD 开发模式指南
- [Go Build Rules](.claude/rules/go-build-rules.md) — Go 构建规则（GOPROXY、磁盘空间）

## Skills (AI 可调用)

- [shareserial-log](.claude/skills/shareserial-log.md) — 获取远程串口 Log 数据，支持过滤和 JSON 输出
- [shareserial-send](.claude/skills/shareserial-send.md) — 发送命令到远程串口（自动写锁管理）
- [shareserial-status](.claude/skills/shareserial-status.md) — 查看连接状态和锁状态
- [shareserial-deploy](.claude/skills/shareserial-deploy.md) — 部署操作指导

## Key Decisions (马斯克五步工作法)

- Language: Go
- Protocol: ~~RFC2217~~ → 简化为纯数据转发
- Baudrate: 固定 115200（Phase 1）
- Arbitration: 独占模式（写锁）
- Virtual Serial: PTY + symlink (真实 Linux 实现)
- Service Discovery: ~~mDNS~~ → 配置文件手动配置
- AI Interface: CLI + Skills 封装
- Serial Library: go.bug.st/serial (GitHub clone + replace)

## Phase 1 Scope (精简后)

- Ubuntu Server + Ubuntu Client + CLI
- Core features: 串口扫描, TCP 服务, PTY 虚拟串口, 数据广播, 写锁仲裁
- AI features: CLI log/send/status 命令, JSON 输出, Skill 封装
- Deferred: mDNS, RFC2217, 多波特率, Windows, 加密, 认证

## Build Commands

```bash
# 构建
make build

# 测试
make test

# 部署 Client
./bin/shareserial-client --server <IP>:7700

# 部署 Server
./bin/shareserial-server --serial /dev/ttyUSB0 --port 7700
```