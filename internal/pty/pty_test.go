package pty

import (
	"testing"
)

// TestPTYCreate 测试 PTY 创建
func TestPTYCreate(t *testing.T) {
	// 在非 Linux 环境可能无法创建真实 PTY
	// 使用 Mock 测试

	pty, err := CreateMockPTY("/dev/vttyShare0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pty == nil {
		t.Fatal("expected PTY to be created")
	}

	// 检查 symlink 路径记录
	if pty.SymlinkPath() != "/dev/vttyShare0" {
		t.Errorf("expected symlink path '/dev/vttyShare0', got '%s'", pty.SymlinkPath())
	}
}

// TestPTYReadWrite 测试 PTY 读写
func TestPTYReadWrite(t *testing.T) {
	pty, err := CreateMockPTY("/dev/vttyShare0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 写入数据到 master
	data := []byte("Test data from master\n")
	n, err := pty.Write(data)
	if err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}

	// 从 slave 读取数据
	buf := make([]byte, 1024)
	n, err = pty.ReadFromSlave(buf)
	if err != nil {
		t.Fatalf("unexpected error reading: %v", err)
	}

	if string(buf[:n]) != string(data) {
		t.Errorf("expected '%s', got '%s'", string(data), string(buf[:n]))
	}
}

// TestPTYClose 测试 PTY 关闭
func TestPTYClose(t *testing.T) {
	pty, err := CreateMockPTY("/dev/vttyShare0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = pty.Close()
	if err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	if pty.IsOpen() {
		t.Error("expected PTY to be closed")
	}
}

// TestPTYSendReceive 测试数据发送和接收
func TestPTYSendReceive(t *testing.T) {
	pty, _ := CreateMockPTY("/dev/vttyShare0")

	// 模拟从外部写入数据（如用户通过 minicom）
	pty.InjectExternalData([]byte("User input\n"))

	// 从 master 读取（服务端接收）
	buf := make([]byte, 1024)
	n, err := pty.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(buf[:n]) != "User input\n" {
		t.Errorf("expected 'User input\\n', got '%s'", string(buf[:n]))
	}

	// 写入数据到 master（服务端发送）
	_, _ = pty.Write([]byte("Server response\n"))

	// 检查 slave 收到数据
	slaveData := pty.GetSlaveData()
	if string(slaveData) != "Server response\n" {
		t.Errorf("expected 'Server response\\n', got '%s'", string(slaveData))
	}
}

// TestPTYTermios 测试 termios 配置
func TestPTYTermios(t *testing.T) {
	pty, _ := CreateMockPTY("/dev/vttyShare0")

	// 设置 termios 配置（模拟 115200 波特率）
	config := &TermiosConfig{
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   ParityNone,
	}

	err := pty.SetTermios(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 检查配置已保存
	saved := pty.GetTermios()
	if saved.BaudRate != 115200 {
		t.Errorf("expected baudrate 115200, got %d", saved.BaudRate)
	}
}
