package cli

import (
	"testing"
)

// TestCLIVersion 测试版本命令
func TestCLIVersion(t *testing.T) {
	cli := NewMockCLI()

	output, err := cli.Execute("version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output != "shareserial v1.0.0" {
		t.Errorf("expected 'shareserial v1.0.0', got '%s'", output)
	}
}

// TestCLILogCommand 测试 log 命令
func TestCLILogCommand(t *testing.T) {
	cli := NewMockCLI()

	// 设置模拟 Log 数据
	cli.SetLogData([]string{
		"[17:30:00] INFO: System starting",
		"[17:30:01] ERROR: Failed to mount",
		"[17:30:02] WARN: Low memory",
	})

	output, err := cli.Execute("log")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 检查输出包含所有 Log
	if len(output) < 50 {
		t.Errorf("expected more log output, got '%s'", output)
	}
}

// TestCLILogFilter 测试 log 过滤功能
func TestCLILogFilter(t *testing.T) {
	cli := NewMockCLI()

	cli.SetLogData([]string{
		"[17:30:00] INFO: System starting",
		"[17:30:01] ERROR: Failed to mount",
		"[17:30:02] WARN: Low memory",
		"[17:30:03] ERROR: Disk error",
	})

	// 过滤 ERROR
	output, err := cli.Execute("log", "--filter", "ERROR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 检查输出只包含 ERROR 行
	if contains(output, "INFO") {
		t.Errorf("output should not contain INFO, got '%s'", output)
	}
	if !contains(output, "ERROR") {
		t.Errorf("output should contain ERROR, got '%s'", output)
	}
}

// TestCLILogJSONFormat 测试 JSON 格式输出
func TestCLILogJSONFormat(t *testing.T) {
	cli := NewMockCLI()

	cli.SetLogData([]string{
		"[17:30:00] INFO: System starting",
		"[17:30:01] ERROR: Failed to mount",
	})

	output, err := cli.Execute("log", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 检查输出是 JSON 格式
	if !contains(output, "{") || !contains(output, "}") {
		t.Errorf("expected JSON format, got '%s'", output)
	}
	if !contains(output, "\"timestamp\"") {
		t.Errorf("expected JSON with timestamp field, got '%s'", output)
	}
}

// TestCLISendCommand 测试 send 命令
func TestCLISendCommand(t *testing.T) {
	cli := NewMockCLI()

	// 设置模拟响应
	cli.SetSendResponse("Command executed: reboot\nSystem rebooting...")

	output, err := cli.Execute("send", "--command", "reboot")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contains(output, "reboot") {
		t.Errorf("expected response containing 'reboot', got '%s'", output)
	}
}

// TestCLIStatusCommand 测试 status 命令
func TestCLIStatusCommand(t *testing.T) {
	cli := NewMockCLI()

	// 设置模拟状态
	cli.SetStatus(map[string]interface{}{
		"server":     "192.168.1.100:7700",
		"serial":     "/dev/ttyUSB0",
		"connected":  true,
		"write_lock": false,
	})

	output, err := cli.Execute("status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 检查输出包含状态信息
	if !contains(output, "192.168.1.100") {
		t.Errorf("expected status with server address, got '%s'", output)
	}
	if !contains(output, "ttyUSB0") {
		t.Errorf("expected status with serial port, got '%s'", output)
	}
}

// TestCLIHelp 测试帮助命令
func TestCLIHelp(t *testing.T) {
	cli := NewMockCLI()

	output, err := cli.Execute("help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 检查输出包含可用命令
	if !contains(output, "log") || !contains(output, "send") || !contains(output, "status") {
		t.Errorf("expected help with available commands, got '%s'", output)
	}
}

// TestCLIUnknownCommand 测试未知命令
func TestCLIUnknownCommand(t *testing.T) {
	cli := NewMockCLI()

	_, err := cli.Execute("unknown")
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOfString(s, substr) >= 0)
}

func indexOfString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
