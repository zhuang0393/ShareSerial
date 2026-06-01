package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// CLI 命令行接口
type CLI struct {
	logData  []string
	sendResp string
	status   map[string]interface{}
}

// NewMockCLI 创建 Mock CLI
func NewMockCLI() *CLI {
	return &CLI{
		logData: make([]string, 0),
		status:  make(map[string]interface{}),
	}
}

// SetLogData 设置模拟 Log 数据
func (c *CLI) SetLogData(data []string) {
	c.logData = data
}

// SetSendResponse 设置模拟发送响应
func (c *CLI) SetSendResponse(resp string) {
	c.sendResp = resp
}

// SetStatus 设置模拟状态
func (c *CLI) SetStatus(status map[string]interface{}) {
	c.status = status
}

// Execute 执行命令
func (c *CLI) Execute(args ...string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("no command specified")
	}

	cmd := args[0]

	switch cmd {
	case "version":
		return "shareserial v1.0.0", nil

	case "log":
		return c.handleLog(args[1:])

	case "send":
		return c.handleSend(args[1:])

	case "status":
		return c.handleStatus(args[1:])

	case "help":
		return c.handleHelp(), nil

	default:
		return "", fmt.Errorf("unknown command: %s", cmd)
	}
}

// handleLog 处理 log 命令
func (c *CLI) handleLog(args []string) (string, error) {
	filter := ""
	format := "text"

	// 解析参数
	for i := 0; i < len(args); i++ {
		if args[i] == "--filter" && i+1 < len(args) {
			filter = args[i+1]
			i++
		}
		if args[i] == "--format" && i+1 < len(args) {
			format = args[i+1]
			i++
		}
	}

	// 过滤数据
	var filtered []string
	for _, line := range c.logData {
		if filter == "" || strings.Contains(line, filter) {
			filtered = append(filtered, line)
		}
	}

	// 格式化输出
	if format == "json" {
		var entries []map[string]string
		for _, line := range filtered {
			// 解析时间戳和级别 - 更宽松的解析
			entry := map[string]string{
				"timestamp": extractTimestamp(line),
				"level":     extractLevel(line),
				"message":   extractMessage(line),
				"raw":       line,
			}
			entries = append(entries, entry)
		}
		if len(entries) == 0 {
			return "[]", nil
		}
		data, _ := json.Marshal(entries)
		return string(data), nil
	}

	// 文本格式
	return strings.Join(filtered, "\n"), nil
}

// extractTimestamp 从 Log 行提取时间戳
func extractTimestamp(line string) string {
	// 格式: [17:30:00] INFO: message
	if strings.HasPrefix(line, "[") {
		end := strings.Index(line, "]")
		if end > 0 {
			return strings.Trim(line[1:end], " ")
		}
	}
	return ""
}

// extractLevel 从 Log 行提取级别
func extractLevel(line string) string {
	// 格式: [17:30:00] INFO: message
	parts := strings.SplitN(line, "]", 2)
	if len(parts) >= 2 {
		levelPart := strings.TrimSpace(parts[1])
		if strings.Contains(levelPart, ":") {
			return strings.TrimSpace(strings.SplitN(levelPart, ":", 2)[0])
		}
	}
	return "INFO"
}

// extractMessage 从 Log 行提取消息
func extractMessage(line string) string {
	// 格式: [17:30:00] INFO: message
	parts := strings.SplitN(line, ": ", 2)
	if len(parts) >= 2 {
		return parts[1]
	}
	return line
}

// handleSend 处理 send 命令
func (c *CLI) handleSend(args []string) (string, error) {
	command := ""

	for i := 0; i < len(args); i++ {
		if args[i] == "--command" && i+1 < len(args) {
			command = args[i+1]
			i++
		}
	}

	if command == "" {
		return "", errors.New("no command specified")
	}

	return c.sendResp, nil
}

// handleStatus 处理 status 命令
func (c *CLI) handleStatus(args []string) (string, error) {
	data, _ := json.MarshalIndent(c.status, "", "  ")
	return string(data), nil
}

// handleHelp 处理 help 命令
func (c *CLI) handleHelp() string {
	return `ShareSerial CLI - Remote Serial Port Sharing Tool

Commands:
  version           Show version information
  log [options]     Get remote serial log data
    --filter <regex> Filter by keyword (regex)
    --since <time>   Time range start (e.g., "5m")
    --format <fmt>   Output format: text/json
    --lines <n>      Max lines to output
  send [options]    Send command to remote serial
    --command <cmd>  Command to send
    --timeout <sec>  Lock timeout (default 30)
  status            Show connection status
  help              Show this help message

Examples:
  shareserial log --filter ERROR --since 5m
  shareserial send --command reboot
  shareserial status`
}
