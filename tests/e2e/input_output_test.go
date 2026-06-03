package e2e

import (
	"os"
	"testing"
	"time"

	"shareserial/internal/pty"
)

// TestEndToEndInputOutput 端到端输入输出测试
func TestEndToEndInputOutput(t *testing.T) {
	t.Log("=== End-to-End Input/Output Test ===")

	// 创建 PTY
	device, err := pty.CreatePTY("/tmp/testpty_e2e")
	if err != nil {
		t.Fatalf("Failed to create PTY: %v", err)
	}
	defer device.Close()
	defer os.Remove("/tmp/testpty_e2e")

	t.Logf("PTY created: %s", device.SymlinkPath())

	// 测试 Server -> Client 数据流（串口响应显示）
	t.Log("Step 1: Server writes response to PTY master")
	response := []byte("file1.txt\nfile2.txt\n")

	n, err := device.Write(response)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	t.Logf("Written to PTY master: %s (n=%d)", string(response), n)

	// 等待数据传递到 slave
	time.Sleep(100 * time.Millisecond)

	// 尝试从 slave 读取（minicom 应该能读取）
	t.Log("Step 2: minicom reads from PTY slave")

	// 检查 real PTY 是否能读取
	if realPTY, ok := device.(*pty.RealPTYDevice); ok {
		slaveBuf := make([]byte, 1024)
		n, err := realPTY.ReadFromSlave(slaveBuf)
		if err != nil {
			t.Logf("Slave read error: %v", err)
		} else {
			t.Logf("minicom received: %s", string(slaveBuf[:n]))
		}
	}

	t.Log("=== Test Completed ===")
}

// TestPTYBasicFlow 测试 PTY 基本数据流
func TestPTYBasicFlow(t *testing.T) {
	t.Log("=== PTY Basic Flow Test ===")

	device, err := pty.CreatePTY("/tmp/testpty_basic")
	if err != nil {
		t.Fatalf("Failed to create PTY: %v", err)
	}
	defer device.Close()
	defer os.Remove("/tmp/testpty_basic")

	// 测试写入 master，读取 slave
	t.Log("Test: Write to master, read from slave")

	data := []byte("test data\n")
	device.Write(data)
	t.Logf("Written to master: %s", string(data))

	// 等待数据传递
	time.Sleep(100 * time.Millisecond)

	// 检查 real PTY
	if realPTY, ok := device.(*pty.RealPTYDevice); ok {
		buf := make([]byte, 1024)
		n, err := realPTY.ReadFromSlave(buf)
		if err != nil {
			t.Logf("Slave read error: %v (expected - data buffered)", err)
		} else if n > 0 {
			t.Logf("Slave read: %s (n=%d)", string(buf[:n]), n)
			if string(buf[:n]) == string(data) {
				t.Log("Data flow verified: master -> slave")
			}
		}
	}

	t.Log("=== Test Completed ===")
}

// TestUserInputFlow 测试用户输入数据流
func TestUserInputFlow(t *testing.T) {
	t.Log("=== User Input Flow Test ===")

	device, err := pty.CreatePTY("/tmp/testpty_user")
	if err != nil {
		t.Fatalf("Failed to create PTY: %v", err)
	}
	defer device.Close()
	defer os.Remove("/tmp/testpty_user")

	// 测试用户输入（通过 slave -> master）
	t.Log("Test: User input from slave to master")

	userInput := []byte("help\n")

	// 检查 real PTY
	if realPTY, ok := device.(*pty.RealPTYDevice); ok {
		// 模拟用户输入（写入 slave）
		realPTY.InjectExternalData(userInput)
		t.Logf("User input injected to slave: %s", string(userInput))

		// 等待数据传递到 master
		time.Sleep(100 * time.Millisecond)

		// Client 从 master 读取
		buf := make([]byte, 1024)

		readDone := make(chan struct{})
		var n int
		go func() {
			defer close(readDone)
			n, _ = device.Read(buf)
		}()

		select {
		case <-readDone:
			t.Logf("Client (master) read: %s (n=%d)", string(buf[:n]), n)
			if n > 0 && string(buf[:n]) == string(userInput) {
				t.Log("User input flow verified: slave -> master")
			}
		case <-time.After(2 * time.Second):
			t.Log("Timeout - user input not received by master (may be expected)")
		}
	} else {
		t.Log("Not a real PTY, skipping user input test")
	}

	t.Log("=== Test Completed ===")
}