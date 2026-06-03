package serial

import (
	"testing"
	"time"
)

// TestMockSerialPortReadWrite 测试 Mock 串口读写行为
func TestMockSerialPortReadWrite(t *testing.T) {
	t.Log("=== Test: Mock Serial Port Read/Write ===")

	port := NewMockSerialPort()
	port.Open("/dev/ttyTest")

	// 测试写入
	t.Log("Test 1: Write to serial")
	data := []byte("test command\n")
	n, err := port.Write(data)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	t.Logf("Written: %d bytes", n)

	// 检查写入的数据
	written := port.GetWrittenData()
	t.Logf("Written data: %s", string(written))

	// 测试读取（无数据时）
	t.Log("Test 2: Read when no input data")
	buf := make([]byte, 1024)
	n, err = port.Read(buf)
	t.Logf("Read result: n=%d, err=%v", n, err)
	if err != nil {
		t.Logf("Error is: %v (EOF expected when no data)", err)
	}

	// 测试注入数据后读取
	t.Log("Test 3: Read after injecting input")
	port.InjectInput([]byte("serial response\n"))
	n, err = port.Read(buf)
	if err != nil {
		t.Logf("Read error: %v", err)
	} else {
		t.Logf("Read: %s (n=%d)", string(buf[:n]), n)
	}

	t.Log("=== Test Completed ===")
}

// TestSerialPortBlockingBehavior 测试串口阻塞行为
// 验证：真实串口 Read 是否阻塞？Mock 串口不阻塞
func TestSerialPortBlockingBehavior(t *testing.T) {
	t.Log("=== Test: Serial Port Blocking Behavior ===")

	port := NewMockSerialPort()
	port.Open("/dev/ttyTest")

	t.Log("Mock serial port Read behavior:")

	// 测试多次读取（无数据）
	for i := 0; i < 5; i++ {
		buf := make([]byte, 1024)

		readDone := make(chan struct{})
		var n int
		var err error
		go func() {
			defer close(readDone)
			n, err = port.Read(buf)
		}()

		select {
		case <-readDone:
			t.Logf("Read %d: completed immediately (n=%d, err=%v)", i+1, n, err)
		case <-time.After(100 * time.Millisecond):
			t.Logf("Read %d: BLOCKING (timeout after 100ms)", i+1)
		}
	}

	t.Log("")
	t.Log("Key finding:")
	t.Log("- Mock serial port Read returns EOF when no data (NOT blocking)")
	t.Log("- Real serial port Read BLOCKS until data arrives")
	t.Log("")
	t.Log("Problem:")
	t.Log("- Server handleClient writes to serial then tries to Read response")
	t.Log("- If serial has no response data, Read will BLOCK forever")
	t.Log("- This causes user input response to never be displayed!")
	t.Log("")
	t.Log("Solution:")
	t.Log("- Need to set ReadTimeout on serial port")
	t.Log("- Or use goroutine with timeout for response read")

	t.Log("=== Test Completed ===")
}

// TestInjectInputForResponse 测试通过 InjectInput 模拟串口响应
func TestInjectInputForResponse(t *testing.T) {
	t.Log("=== Test: Inject Input for Response ===")

	port := NewMockSerialPort()
	port.Open("/dev/ttyTest")

	// 写入用户命令
	command := []byte("ls\n")
	port.Write(command)
	t.Logf("Command written: %s", string(command))

	// 模拟串口设备响应（注入输入数据）
	response := []byte("file1.txt\nfile2.txt\n")
	port.InjectInput(response)
	t.Logf("Response injected: %s", string(response))

	// 立即读取响应
	buf := make([]byte, 1024)
	n, err := port.Read(buf)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	t.Logf("Response read: %s (n=%d)", string(buf[:n]), n)

	// 验证数据
	if string(buf[:n]) != string(response) {
		t.Errorf("Response mismatch")
	}

	t.Log("=== Test Completed ===")
}

// TestMultipleReadWriteCycle 测试多次读写循环
func TestMultipleReadWriteCycle(t *testing.T) {
	t.Log("=== Test: Multiple Read/Write Cycle ===")

	port := NewMockSerialPort()
	port.Open("/dev/ttyTest")

	cycles := []struct {
		command  []byte
		response []byte
	}{
		{[]byte("help\n"), []byte("Commands: ls, pwd, help\n")},
		{[]byte("ls\n"), []byte("file1\nfile2\n")},
		{[]byte("pwd\n"), []byte("/home\n")},
	}

	for i, cycle := range cycles {
		t.Logf("Cycle %d:", i+1)

		// 写入命令
		port.Write(cycle.command)
		t.Logf("  Command: %s", string(cycle.command))

		// 模拟响应
		port.InjectInput(cycle.response)

		// 读取响应
		buf := make([]byte, 1024)
		n, err := port.Read(buf)
		if err != nil {
			t.Logf("  Read error: %v", err)
			continue
		}
		t.Logf("  Response: %s", string(buf[:n]))
	}

	t.Log("=== Test Completed ===")
}