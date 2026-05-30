package simulation

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

// TerminalResult 终端操作结果
type TerminalResult struct {
	Status           string `json:"status"`
	Message          string `json:"message,omitempty"`
	Output           string `json:"output,omitempty"`
	Command          string `json:"command,omitempty"`
	ContainsExpected bool   `json:"contains_expected,omitempty"`
}

// TerminalTester 终端自动化测试器
type TerminalTester struct {
	ptyPath      string
	terminal     string
	pythonScript string
	running      bool
}

// NewTerminalTester 创建终端测试器
func NewTerminalTester(ptyPath string) *TerminalTester {
	return &TerminalTester{
		ptyPath:      ptyPath,
		terminal:     "minicom",
		pythonScript: filepath.Join(projectRoot, "tests", "simulation", "helpers", "minicom_automation.py"),
	}
}

// NewTerminalTesterWithTerminal 使用指定终端创建测试器
func NewTerminalTesterWithTerminal(ptyPath, terminal string) *TerminalTester {
	return &TerminalTester{
		ptyPath:      ptyPath,
		terminal:     terminal,
		pythonScript: filepath.Join(projectRoot, "tests", "simulation", "helpers", "minicom_automation.py"),
	}
}

// Start 启动终端程序
func (t *TerminalTester) Start() (*TerminalResult, error) {
	cmd := exec.Command("python3", t.pythonScript, "start", t.ptyPath, t.terminal)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to start terminal: %v", err)
	}

	var result TerminalResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %v", err)
	}

	if result.Status == "started" {
		t.running = true
	}

	return &result, nil
}

// SendCommandAndVerify 发送命令并验证输出
func (t *TerminalTester) SendCommandAndVerify(command, expected string) (*TerminalResult, error) {
	cmd := exec.Command("python3", t.pythonScript, "test", t.ptyPath, command, expected)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute test: %v", err)
	}

	var result TerminalResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %v", err)
	}

	return &result, nil
}

// ReadOutput 读取终端输出
func (t *TerminalTester) ReadOutput(timeout int) (*TerminalResult, error) {
	cmd := exec.Command("python3", t.pythonScript, "read", t.ptyPath, fmt.Sprintf("%d", timeout))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read output: %v", err)
	}

	var result TerminalResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %v", err)
	}

	return &result, nil
}

// VerifyOutput 验证输出包含特定内容
func (t *TerminalTester) VerifyOutput(expected string) (*TerminalResult, error) {
	cmd := exec.Command("python3", t.pythonScript, "verify", t.ptyPath, expected)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to verify output: %v", err)
	}

	var result TerminalResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %v", err)
	}

	return &result, nil
}

// WaitForPattern 等待特定输出模式
func (t *TerminalTester) WaitForPattern(pattern string, timeout time.Duration) (*TerminalResult, error) {
	_ = int(timeout.Seconds()) // timeout parameter for future use
	return t.SendCommandAndVerify("", pattern)
}

// Close 关闭终端
func (t *TerminalTester) Close() error {
	t.running = false
	// Python 脚本会在测试完成后自动关闭
	return nil
}

// IsRunning 检查是否运行
func (t *TerminalTester) IsRunning() bool {
	return t.running
}

// UseSimpleReader 使用简单的 cat 模式读取（不启动 minicom）
func (t *TerminalTester) UseSimpleReader() *TerminalTester {
	t.terminal = "cat"
	return t
}

// SimpleRead 简单读取 PTY 输出（使用 cat）
func (t *TerminalTester) SimpleRead(timeout int) (string, error) {
	cmd := exec.Command("python3", t.pythonScript, "read", t.ptyPath, fmt.Sprintf("%d", timeout), "cat")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read: %v", err)
	}

	var result TerminalResult
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse result: %v", err)
	}

	return result.Output, nil
}