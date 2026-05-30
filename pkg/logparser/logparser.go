package logparser

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
)

// LogEntry Log 条目
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Raw       string `json:"raw"`
}

// Parser Log 解析器
type Parser struct {
	timeRegex *regexp.Regexp
	levelRegex *regexp.Regexp
}

// NewParser 创建解析器
func NewParser() *Parser {
	return &Parser{
		timeRegex:  regexp.MustCompile(`\[(\d{2}:\d{2}:\d{2}(\.\d+)?)\]`),
		levelRegex: regexp.MustCompile(`\]\s*(INFO|WARN|WARNING|ERROR|DEBUG|FATAL|TRACE)\s*:`),
	}
}

// ParseLine 解析单行 Log
func (p *Parser) ParseLine(line string) (*LogEntry, error) {
	entry := &LogEntry{
		Raw: line,
	}

	// 提取时间戳
	timeMatch := p.timeRegex.FindStringSubmatch(line)
	if len(timeMatch) >= 2 {
		entry.Timestamp = timeMatch[1]
	} else {
		entry.Timestamp = ""
	}

	// 提取级别
	levelMatch := p.levelRegex.FindStringSubmatch(line)
	if len(levelMatch) >= 2 {
		entry.Level = levelMatch[1]
		// 标准化 WARNING 为 WARN
		if entry.Level == "WARNING" {
			entry.Level = "WARN"
		}
	} else {
		entry.Level = "INFO" // 默认级别
	}

	// 提取消息
	// 格式: [timestamp] LEVEL: message
	parts := strings.SplitN(line, ": ", 2)
	if len(parts) >= 2 {
		entry.Message = parts[1]
	} else {
		entry.Message = line
	}

	if entry.Timestamp == "" {
		return entry, errors.New("invalid log format")
	}

	return entry, nil
}

// ParseLines 解析多行 Log
func (p *Parser) ParseLines(lines []string) []*LogEntry {
	entries := make([]*LogEntry, 0, len(lines))
	for _, line := range lines {
		entry, _ := p.ParseLine(line)
		if entry != nil {
			entries = append(entries, entry)
		}
	}
	return entries
}

// Filter 关键词过滤（简单字符串匹配）
func (p *Parser) Filter(lines []string, keyword string) []string {
	if keyword == "" {
		return lines
	}

	filtered := make([]string, 0)
	for _, line := range lines {
		if strings.Contains(line, keyword) {
			filtered = append(filtered, line)
		}
	}
	return filtered
}

// FilterRegex 正则表达式过滤
func (p *Parser) FilterRegex(lines []string, pattern string) []string {
	if pattern == "" {
		return lines
	}

	re := regexp.MustCompile(pattern)
	filtered := make([]string, 0)
	for _, line := range lines {
		if re.MatchString(line) {
			filtered = append(filtered, line)
		}
	}
	return filtered
}

// FilterTimeRange 时间范围过滤
func (p *Parser) FilterTimeRange(lines []string, start, end time.Time) []string {
	filtered := make([]string, 0)
	for _, line := range lines {
		entry, err := p.ParseLine(line)
		if err != nil {
			continue
		}

		// 解析时间戳（格式 HH:MM:SS 或 HH:MM:SS.mmm）
		ts := entry.Timestamp
		// 去掉毫秒部分
		if strings.Contains(ts, ".") {
			ts = strings.Split(ts, ".")[0]
		}

		lineTime, err := time.Parse("15:04:05", ts)
		if err != nil {
			continue
		}

		// 检查是否在范围内（只比较时分秒）
		// 简化版：假设在同一天内
		lineSeconds := lineTime.Hour()*3600 + lineTime.Minute()*60 + lineTime.Second()
		startSeconds := start.Hour()*3600 + start.Minute()*60 + start.Second()
		endSeconds := end.Hour()*3600 + end.Minute()*60 + end.Second()

		// 处理跨天情况
		if startSeconds > endSeconds {
			// 例如 start=23:00, end=01:00
			if lineSeconds >= startSeconds || lineSeconds <= endSeconds {
				filtered = append(filtered, line)
			}
		} else {
			if lineSeconds >= startSeconds && lineSeconds <= endSeconds {
				filtered = append(filtered, line)
			}
		}
	}
	return filtered
}

// LimitLines 限制行数
func (p *Parser) LimitLines(lines []string, max int) []string {
	if max <= 0 || len(lines) <= max {
		return lines
	}
	return lines[:max]
}

// ToJSON 转换为 JSON 格式
func (p *Parser) ToJSON(entries []*LogEntry) string {
	if len(entries) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(entries)
	return string(data)
}

// ToText 转换为文本格式
func (p *Parser) ToText(entries []*LogEntry) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, entry.Raw)
	}
	return strings.Join(lines, "\n")
}

// FilterByLevel 按级别过滤
func (p *Parser) FilterByLevel(entries []*LogEntry, level string) []*LogEntry {
	filtered := make([]*LogEntry, 0)
	for _, entry := range entries {
		if entry.Level == level {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// FilterByLevels 按多个级别过滤
func (p *Parser) FilterByLevels(entries []*LogEntry, levels []string) []*LogEntry {
	levelSet := make(map[string]bool)
	for _, l := range levels {
		levelSet[l] = true
	}

	filtered := make([]*LogEntry, 0)
	for _, entry := range entries {
		if levelSet[entry.Level] {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// Stats 统计 Log 信息
func (p *Parser) Stats(entries []*LogEntry) map[string]int {
	stats := make(map[string]int)
	stats["total"] = len(entries)

	for _, entry := range entries {
		stats[entry.Level]++
	}

	return stats
}