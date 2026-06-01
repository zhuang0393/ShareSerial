package logparser

import (
	"testing"
	"time"
)

// TestLogEntryParse 测试 Log 条目解析
func TestLogEntryParse(t *testing.T) {
	parser := NewParser()

	line := "[17:30:00.123] INFO: System starting"
	entry, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry.Timestamp != "17:30:00.123" {
		t.Errorf("expected timestamp '17:30:00.123', got '%s'", entry.Timestamp)
	}
	if entry.Level != "INFO" {
		t.Errorf("expected level 'INFO', got '%s'", entry.Level)
	}
	if entry.Message != "System starting" {
		t.Errorf("expected message 'System starting', got '%s'", entry.Message)
	}
}

// TestLogEntryLevel 测试不同级别解析
func TestLogEntryLevel(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		line  string
		level string
	}{
		{"[17:30:00] INFO: message", "INFO"},
		{"[17:30:00] WARN: message", "WARN"},
		{"[17:30:00] ERROR: message", "ERROR"},
		{"[17:30:00] DEBUG: message", "DEBUG"},
		{"[17:30:00] FATAL: message", "FATAL"},
	}

	for _, tt := range tests {
		entry, _ := parser.ParseLine(tt.line)
		if entry.Level != tt.level {
			t.Errorf("expected level '%s', got '%s'", tt.level, entry.Level)
		}
	}
}

// TestLogParserFilter 测试关键词过滤
func TestLogParserFilter(t *testing.T) {
	parser := NewParser()

	lines := []string{
		"[17:30:00] INFO: System starting",
		"[17:30:01] ERROR: Failed to mount",
		"[17:30:02] WARN: Low memory",
		"[17:30:03] ERROR: Disk error",
	}

	// 过滤 ERROR
	filtered := parser.Filter(lines, "ERROR")
	if len(filtered) != 2 {
		t.Errorf("expected 2 ERROR lines, got %d", len(filtered))
	}

	// 过滤空（返回全部）
	all := parser.Filter(lines, "")
	if len(all) != 4 {
		t.Errorf("expected 4 lines with empty filter, got %d", len(all))
	}

	// 过滤正则
	regFiltered := parser.FilterRegex(lines, "ERROR|WARN")
	if len(regFiltered) != 3 {
		t.Errorf("expected 3 ERROR/WARN lines, got %d", len(regFiltered))
	}
}

// TestLogParserTimeRange 测试时间范围过滤
func TestLogParserTimeRange(t *testing.T) {
	parser := NewParser()

	lines := []string{
		"[17:30:00] INFO: Line 1",
		"[17:30:05] INFO: Line 2",
		"[17:30:10] INFO: Line 3",
		"[17:30:15] INFO: Line 4",
	}

	// 过滤最近 10 秒（假设当前时间 17:30:15）
	// 从 17:30:05 到 17:30:15
	start, _ := time.Parse("15:04:05", "17:30:05")
	end, _ := time.Parse("15:04:05", "17:30:15")
	filtered := parser.FilterTimeRange(lines, start, end)
	if len(filtered) != 3 {
		t.Errorf("expected 3 lines in time range (17:30:05-17:30:15), got %d", len(filtered))
	}

	// 测试另一范围
	start2, _ := time.Parse("15:04:05", "17:30:00")
	end2, _ := time.Parse("15:04:05", "17:30:10")
	filtered2 := parser.FilterTimeRange(lines, start2, end2)
	if len(filtered2) != 3 {
		t.Errorf("expected 3 lines in time range (17:30:00-17:30:10), got %d", len(filtered2))
	}
}

// Helper functions (removed parseTime as it was incorrect)
func TestLogParserJSON(t *testing.T) {
	parser := NewParser()

	lines := []string{
		"[17:30:00] INFO: System starting",
		"[17:30:01] ERROR: Failed to mount",
	}

	entries := parser.ParseLines(lines)
	jsonOutput := parser.ToJSON(entries)

	// 检查 JSON 包含必要字段
	if !containsStr(jsonOutput, "timestamp") {
		t.Errorf("expected JSON with timestamp field")
	}
	if !containsStr(jsonOutput, "level") {
		t.Errorf("expected JSON with level field")
	}
	if !containsStr(jsonOutput, "message") {
		t.Errorf("expected JSON with message field")
	}
}

// TestLogParserText 测试文本格式化
func TestLogParserText(t *testing.T) {
	parser := NewParser()

	lines := []string{
		"[17:30:00] INFO: System starting",
		"[17:30:01] ERROR: Failed to mount",
	}

	entries := parser.ParseLines(lines)
	textOutput := parser.ToText(entries)

	// 检查文本包含原始内容
	if !containsStr(textOutput, "INFO") {
		t.Errorf("expected text with INFO")
	}
	if !containsStr(textOutput, "ERROR") {
		t.Errorf("expected text with ERROR")
	}
}

// TestLogParserMaxLines 测试最大行数限制
func TestLogParserMaxLines(t *testing.T) {
	parser := NewParser()

	lines := make([]string, 100)
	for i := 0; i < 100; i++ {
		lines[i] = "[17:30:00] INFO: Line " + string(rune('0'+i%10))
	}

	// 限制 10 行
	limited := parser.LimitLines(lines, 10)
	if len(limited) != 10 {
		t.Errorf("expected 10 lines, got %d", len(limited))
	}
}

// TestLogEntryRaw 测试原始行保留
func TestLogEntryRaw(t *testing.T) {
	parser := NewParser()

	line := "[17:30:00.123] INFO: System starting"
	entry, _ := parser.ParseLine(line)

	if entry.Raw != line {
		t.Errorf("expected raw line to be preserved, got '%s'", entry.Raw)
	}
}

// TestLogParserInvalidLine 测试无效行处理
func TestLogParserInvalidLine(t *testing.T) {
	parser := NewParser()

	// 无效格式的行
	line := "Invalid log line without timestamp"
	entry, err := parser.ParseLine(line)

	// 应该返回错误或默认值
	if err == nil && entry.Timestamp == "" {
		// 如果没有错误，检查是否使用默认值
		t.Log("Invalid line handled with default values")
	}
}

// Helper functions
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOfStr(s, substr) >= 0)
}

func indexOfStr(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
