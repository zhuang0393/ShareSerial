# ShareSerial Log Analysis Skill

## Description

获取远程串口 Log 数据，支持过滤关键词、时间范围、JSON 格式输出，便于 AI 分析。

## Trigger

当用户提到以下内容时自动触发：
- "查看串口 Log"
- "分析 Log"
- "远程串口日志"
- "获取 Log 数据"
- "查看开发板输出"

## Usage

```bash
/skill shareserial-log [options]
```

## Parameters

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--filter` | 过滤关键词（正则表达式） | 空（显示全部） |
| `--since` | 时间范围起点（如 "5m" 表示最近5分钟） | "1m" |
| `--until` | 时间范围终点 | 空（持续输出） |
| `--format` | 输出格式（text/json） | "text" |
| `--lines` | 最大行数 | 100 |
| `--server` | 服务器地址（从配置读取） | client.yaml 配置值 |

## Examples

### 查看最近1分钟的 Log

```bash
shareserial log --since "1m"
```

### 过滤 ERROR 和 WARN

```bash
shareserial log --filter "ERROR|WARN" --since "5m"
```

### 输出 JSON 格式便于解析

```bash
shareserial log --format json --since "10m"
```

### 查看内核相关 Log

```bash
shareserial log --filter "kernel|Kernel" --lines 50
```

## Command Execution

```bash
shareserial log --server ${SHARESERIAL_SERVER:-192.168.1.100:7700} \
  --filter "${filter:-}" \
  --since "${since:-1m}" \
  --format ${format:-text} \
  --lines ${lines:-100}
```

## Output Format

### Text Format

```
[17:30:00.123] INFO: System starting...
[17:30:00.456] ERROR: Failed to mount /data
[17:30:01.789] WARN: Low memory warning
```

### JSON Format

```json
{"timestamp":"2026-05-28T17:30:00.123Z","level":"INFO","message":"System starting...","raw":"[17:30:00.123] INFO: System starting..."}
{"timestamp":"2026-05-28T17:30:00.456Z","level":"ERROR","message":"Failed to mount /data","raw":"[17:30:00.456] ERROR: Failed to mount /data"}
```

## Integration with Claude Code

Claude Code 可以直接调用此 Skill 来：

1. **实时分析 Log**：获取 Log 后自动识别 ERROR、WARN 模式
2. **问题定位**：根据 Log 内容定位可能的问题模块
3. **命令发送**：分析后可发送调试命令获取更多信息

## Related Skills

- `shareserial-send`: 发送命令到远程串口（需要写锁）
- `shareserial-status`: 查看连接状态和锁状态