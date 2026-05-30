# ShareSerial Status Skill

## Description

查看 ShareSerial 连接状态、锁状态、客户端列表。

## Trigger

当用户提到以下内容时自动触发：
- "串口状态"
- "连接状态"
- "谁在使用串口"
- "锁状态"

## Usage

```bash
/skill shareserial-status
```

## Output Format

```json
{
  "server": "192.168.1.100:7700",
  "serial": "/dev/ttyUSB0",
  "baudrate": 115200,
  "connected_clients": 3,
  "write_lock": {
    "locked": true,
    "owner": "client-192.168.1.50",
    "acquired_at": "2026-05-28T17:30:00Z",
    "expires_at": "2026-05-28T17:30:30Z"
  },
  "clients": [
    {"id": "client-192.168.1.50", "mode": "write", "connected_at": "17:25:00"},
    {"id": "client-192.168.1.51", "mode": "read", "connected_at": "17:26:00"},
    {"id": "client-192.168.1.52", "mode": "read", "connected_at": "17:28:00"}
  ]
}
```

## Command Execution

```bash
shareserial status --server ${SHARESERIAL_SERVER:-192.168.1.100:7700}
```

## Use Cases

1. **检查是否可以发送命令**：查看锁状态
2. **查看有多少人在线**：了解并发情况
3. **排查连接问题**：检查服务端状态