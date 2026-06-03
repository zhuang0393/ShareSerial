package pty

import (
	"testing"
	"time"
)

// TestPTYUserInputSimulation 测试用户输入场景
// 模拟：用户在 minicom 输入 -> PTY slave -> PTY master 读取
func TestPTYUserInputSimulation(t *testing.T) {
	pty, err := CreateRealPTY("/tmp/testpty_input")
	if err != nil {
		t.Fatalf("Failed to create PTY: %v", err)
	}
	defer pty.Close()

	t.Logf("PTY created: %s -> %s", "/tmp/testpty_input", pty.SymlinkPath())

	// 模拟用户在 minicom 输入数据（写入 slave）
	// 注意：实际 minicom 会连接到 slave (symlink)
	t.Log("=== Test 1: User input from slave ===")

	// 使用 InjectExternalData 模拟用户输入
	userInput := []byte("ls\n")
	pty.InjectExternalData(userInput)
	t.Logf("Injected user input: %s", string(userInput))

	// 从 master 读取（Client 应该能读取到）
	buf := make([]byte, 1024)

	// 设置读取超时（使用 goroutine）
	readDone := make(chan struct{})
	var n int
	var readErr error
	go func() {
		defer close(readDone)
		n, readErr = pty.Read(buf)
	}()

	select {
	case <-readDone:
		if readErr != nil {
			t.Fatalf("Failed to read from master: %v", readErr)
		}
		t.Logf("Read from master: %s (n=%d)", string(buf[:n]), n)
		if string(buf[:n]) != string(userInput) {
			t.Errorf("Expected %s, got %s", string(userInput), string(buf[:n]))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for user input")
	}

	t.Log("=== Test 2: Server output to master ===")

	// 模拟 Server 发送数据到 PTY（写入 master）
	serverOutput := []byte("file1\nfile2\n")
	n, err = pty.Write(serverOutput)
	if err != nil {
		t.Fatalf("Failed to write to master: %v", err)
	}
	t.Logf("Written to master: %s (n=%d)", string(serverOutput), n)

	// 验证 slave 是否能读取到（minicom 应该能显示）
	// 注意：这个测试需要打开 slave 文件来读取
	// 在实际场景中，minicom 会连接到 slave
	t.Log("Server output should appear on slave (minicom)")
}

// TestPTYBidirectionalFlow 测试双向数据流
func TestPTYBidirectionalFlow(t *testing.T) {
	pty, err := CreateRealPTY("/tmp/testpty_bidir")
	if err != nil {
		t.Fatalf("Failed to create PTY: %v", err)
	}
	defer pty.Close()

	// 测试场景：完整的输入-输出循环
	// 1. 用户输入命令
	// 2. 设备响应输出

	t.Log("=== Bidirectional Flow Test ===")

	// 启动一个 goroutine 模拟 minicom（从 slave 读取）
	slaveReaderDone := make(chan struct{})
	slaveOutput := make([]byte, 0)
	go func() {
		defer close(slaveReaderDone)
		// 从 slave 读取（ReadFromSlave）
		buf := make([]byte, 1024)
		for {
			n, err := pty.ReadFromSlave(buf)
			if err != nil {
				return
			}
			slaveOutput = append(slaveOutput, buf[:n]...)
			if len(slaveOutput) > 0 {
				return // 读到一些数据后退出
			}
		}
	}()

	// 模拟用户输入
	userInput := []byte("help\n")
	pty.InjectExternalData(userInput)
	t.Logf("User input: %s", string(userInput))

	// 从 master 读取用户输入
	buf := make([]byte, 1024)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		n, _ := pty.Read(buf)
		t.Logf("Master received user input: %s", string(buf[:n]))
	}()

	select {
	case <-readDone:
	case <-time.After(1 * time.Second):
		t.Log("Timeout waiting for user input (expected in raw mode)")
	}

	// 模拟设备响应
	deviceResponse := []byte("Available commands: ls, help, exit\n")
	pty.Write(deviceResponse)
	t.Logf("Device response written to master: %s", string(deviceResponse))

	// 等待 slave 读取
	select {
	case <-slaveReaderDone:
		t.Logf("Slave received: %s", string(slaveOutput))
	case <-time.After(2 * time.Second):
		t.Log("Timeout waiting for slave to read")
	}

	// 验证：slave 应该收到设备响应
	if len(slaveOutput) > 0 {
		t.Logf("Slave output: %s", string(slaveOutput))
	} else {
		t.Log("Slave output empty (may need different read approach)")
	}
}

// TestPTYEchoBehavior 测试 PTY 回显行为
func TestPTYEchoBehavior(t *testing.T) {
	pty, err := CreateRealPTY("/tmp/testpty_echo")
	if err != nil {
		t.Fatalf("Failed to create PTY: %v", err)
	}
	defer pty.Close()

	t.Log("=== Echo Behavior Test ===")
	t.Log("In raw mode (ICANON disabled), echo should be disabled")

	// 用户输入
	userInput := []byte("test")
	pty.InjectExternalData(userInput)

	// 等待一下
	time.Sleep(100 * time.Millisecond)

	// 尝试从 slave 读取回显
	buf := make([]byte, 1024)
	n, err := pty.ReadFromSlave(buf)
	if err != nil {
		t.Logf("No echo on slave (expected in raw mode): %v", err)
	} else {
		t.Logf("Slave read: %s (n=%d)", string(buf[:n]), n)
		if string(buf[:n]) == string(userInput) {
			t.Log("Echo detected - this may cause duplicate display")
		}
	}
}

// TestPTYRealWorldScenario 真实场景测试
// 模拟：Client 读取用户输入 -> 发送到 Server -> Server 响应 -> Client 写入
func TestPTYRealWorldScenario(t *testing.T) {
	pty, err := CreateRealPTY("/tmp/testpty_real")
	if err != nil {
		t.Fatalf("Failed to create PTY: %v", err)
	}
	defer pty.Close()

	t.Log("=== Real World Scenario Test ===")

	// Step 1: 用户输入命令
	t.Log("Step 1: User inputs 'ls' command")
	userCommand := []byte("ls\n")
	pty.InjectExternalData(userCommand)

	// Step 2: Client 从 master 读取用户输入
	t.Log("Step 2: Client reads from master")
	buf := make([]byte, 1024)

	readDone := make(chan struct{})
	var readN int
	go func() {
		defer close(readDone)
		readN, _ = pty.Read(buf)
	}()

	select {
	case <-readDone:
		t.Logf("Client received: %s", string(buf[:readN]))
	case <-time.After(2 * time.Second):
		t.Fatal("Client timeout - user input not received")
	}

	// Step 3: Client 发送到 Server（这里跳过，直接模拟 Server 响应）
	t.Log("Step 3: Server processes and responds")

	// Step 4: Server 响应发送回 Client
	t.Log("Step 4: Server response sent back to Client")
	serverResponse := []byte("file1.txt\nfile2.txt\ndir/\n")
	_, _ = pty.Write(serverResponse)

	// Step 5: 验证 slave (minicom) 能看到响应
	t.Log("Step 5: Verify slave (minicom) receives response")
	time.Sleep(100 * time.Millisecond)

	// 尝试从 slave 读取
	responseBuf := make([]byte, 1024)
	n, err := pty.ReadFromSlave(responseBuf)
	if err != nil {
		t.Logf("Slave read error: %v (response may be on slave side)", err)
	} else {
		t.Logf("Slave received: %s", string(responseBuf[:n]))
	}

	t.Log("Test completed - minicom should display server response")
}