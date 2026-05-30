# ShareSerial Send Command Skill

## Description

发送命令到远程串口，自动申请和释放写锁，支持交互式命令。

## Trigger

当用户提到以下内容时自动触发：
- "发送命令到串口"
- "执行串口命令"
- "向开发板发送"
- "串口输入"

## Usage

```bash
/skill shareserial-send [options]
```

## Parameters

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--command` | 要发送的命令 | 必填 |
| `--timeout` | 写锁超时时间（秒） | 30 |
| `--server` | 服务器地址 | client.yaml 配置值 |
| `--wait` | 等待响应时间（秒） | 5 |

## Examples

### 发送 reboot 命令

```bash
shareserial send --command "reboot"
```

### 发送 dmesg 获取内核日志

```bash
shareserial send --command "dmesg" --wait 10
```

### 发送自定义命令

```bash
shareserial send --command "cat /proc/version"
```

## Command Execution

```bash
shareserial send --server ${SHARESERIAL_SERVER:-192.168.1.100:7700} \
  --command "${command}" \
  --timeout ${timeout:-30} \
  --wait ${wait:-5}
```

## Behavior

1. 自动申请写锁（独占模式）
2. 发送命令到串口
3. 等待响应（可选）
4. 自动释放写锁

## Output

命令发送后会输出：
- 写锁申请状态
- 发送的命令内容
- 响应数据（如果有）

## Error Handling

- 如果写锁被占用，会提示当前持有者
- 如果超时，自动释放锁并提示
- 如果连接断开，自动重试