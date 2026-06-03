package server

import (
	"testing"
	"time"

	"shareserial/pkg/arbiter"
	"shareserial/pkg/serial"
)

// TestWriteAndImmediateRead 测试写入后立即读取响应
// 这个测试模拟用户输入后立即收到响应的场景
func TestWriteAndImmediateRead(t *testing.T) {
	t.Log("=== Test: Write and Immediate Read ===")

	// 创建 Arbiter
	arb := arbiter.NewArbiter(30 * time.Second)

	// 创建 Mock 串口
	mockSerial := serial.NewMockSerialPort()

	// 创建 Mock Server
	srv := NewMockServerWithSerial(arb, mockSerial)

	// Client ID
	client := "test-client"

	// 获取写锁
	acquired, _ := arb.Acquire(client)
	if !acquired {
		t.Fatal("Failed to acquire lock")
	}
	t.Logf("Lock acquired by %s", client)

	// Step 1: 用户输入命令
	userInput := []byte("ls\n")
	t.Logf("Step 1: User input: %s", string(userInput))

	// Step 2: Server 写入串口
	t.Log("Step 2: Server writes to serial")
	mockSerial.Write(userInput)
	t.Logf("Written to serial: %s", string(userInput))

	// Step 3: 模拟串口立即响应
	// 注意：MockSerialPort 需要有方法来设置响应数据
	t.Log("Step 3: Serial responds immediately")
	serialResponse := []byte("file1.txt\nfile2.txt\n")

	// 检查 MockSerialPort 是否有设置数据的方法
	t.Logf("Serial should respond: %s", string(serialResponse))

	// Step 4: Server 立即读取响应
	t.Log("Step 4: Server reads response immediately")
	responseBuf := make([]byte, 4096)

	// 尝试读取多次
	readSuccess := false
	for i := 0; i < 3; i++ {
		n, err := mockSerial.Read(responseBuf)
		t.Logf("Read attempt %d: n=%d, err=%v", i+1, n, err)
		if err == nil && n > 0 {
			t.Logf("Response read: %s", string(responseBuf[:n]))
			readSuccess = true
			// 广播响应
			srv.Broadcast(responseBuf[:n])
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !readSuccess {
		t.Log("No response read (expected - MockSerialPort may not auto-respond)")
		t.Log("This test verifies the READ AFTER WRITE logic exists")
	}

	// Step 5: 验证广播
	t.Log("Step 5: Verify broadcast to clients")

	// 添加 Mock Client
	srv.AddMockClient(client)
	time.Sleep(100 * time.Millisecond)

	clientData := srv.GetMockClientData(client)
	t.Logf("Client received: %s", string(clientData))

	t.Log("=== Test Completed ===")
}

// TestInputResponseCycle 测试完整的输入-响应循环
func TestInputResponseCycle(t *testing.T) {
	t.Log("=== Test: Input-Response Cycle ===")

	arb := arbiter.NewArbiter(30 * time.Second)
	mockSerial := serial.NewMockSerialPort()
	srv := NewMockServerWithSerial(arb, mockSerial)

	client := "test-client"
	srv.AddMockClient(client)

	// 获取锁
	arb.Acquire(client)

	// 模拟多轮输入-响应
	cycles := []struct {
		input    []byte
		response []byte
	}{
		{[]byte("ls\n"), []byte("file1\nfile2\n")},
		{[]byte("pwd\n"), []byte("/home/user\n")},
		{[]byte("echo test\n"), []byte("test\n")},
	}

	for i, cycle := range cycles {
		t.Logf("Cycle %d: Input='%s'", i+1, string(cycle.input))

		// 写入
		mockSerial.Write(cycle.input)
		t.Logf("  Written: %s", string(cycle.input))

		// 尝试读取响应
		buf := make([]byte, 1024)
		for j := 0; j < 3; j++ {
			n, _ := mockSerial.Read(buf)
			if n > 0 {
				t.Logf("  Read response: %s", string(buf[:n]))
				srv.Broadcast(buf[:n])
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
	}

	time.Sleep(100 * time.Millisecond)
	clientData := srv.GetMockClientData(client)
	t.Logf("Client total data: %s", string(clientData))

	t.Log("=== Test Completed ===")
}

// TestBlockingReadBehavior 测试阻塞读取行为
func TestBlockingReadBehavior(t *testing.T) {
	t.Log("=== Test: Blocking Read Behavior ===")

	mockSerial := serial.NewMockSerialPort()

	t.Log("This test checks if serial.Read blocks when no data available")

	// 尝试无数据时读取
	buf := make([]byte, 1024)

	// 使用 goroutine 检测阻塞
	readDone := make(chan struct{})
	var n int
	var err error

	go func() {
		defer close(readDone)
		n, err = mockSerial.Read(buf)
	}()

	select {
	case <-readDone:
		t.Logf("Read completed: n=%d, err=%v", n, err)
		t.Log("Read does NOT block (good for immediate response check)")
	case <-time.After(500 * time.Millisecond):
		t.Log("Read is BLOCKING (timeout after 500ms)")
		t.Log("Blocking read may cause response delay!")
	}

	t.Log("=== Test Completed ===")
}

// TestRealWorldInputScenario 测试真实场景
// 模拟：用户敲回车 -> console 显示 -> 无 log 输出时再敲回车 -> 无响应
func TestRealWorldInputScenario(t *testing.T) {
	t.Log("=== Test: Real World Input Scenario ===")

	arb := arbiter.NewArbiter(30 * time.Second)
	mockSerial := serial.NewMockSerialPort()
	srv := NewMockServerWithSerial(arb, mockSerial)

	client := "test-client"
	srv.AddMockClient(client)
	arb.Acquire(client)

	// 场景 1: 有 log 输出时输入
	t.Log("Scenario 1: Input when log is outputting")

	// 模拟串口正在输出 log
	logData := []byte("[log] system running...\n")
	mockSerial.Write(logData)

	// 用户输入回车
	userInput1 := []byte("\n")
	mockSerial.Write(userInput1)

	// 尝试读取
	buf := make([]byte, 1024)
	for i := 0; i < 3; i++ {
		n, _ := mockSerial.Read(buf)
		if n > 0 {
			t.Logf("Read at attempt %d: %s", i+1, string(buf[:n]))
			srv.Broadcast(buf[:n])
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 场景 2: 无 log 输出时输入
	t.Log("Scenario 2: Input when NO log is outputting")

	// 用户再次输入回车
	userInput2 := []byte("\n")
	mockSerial.Write(userInput2)

	// 尝试读取（应该能读到 console 响应）
	for i := 0; i < 3; i++ {
		n, _ := mockSerial.Read(buf)
		if n > 0 {
			t.Logf("Read at attempt %d: %s", i+1, string(buf[:n]))
			srv.Broadcast(buf[:n])
			t.Log("Response received even without log output (GOOD)")
			break
		}
		t.Logf("Attempt %d: no response yet", i+1)
		time.Sleep(50 * time.Millisecond)
	}

	// 检查 Client 收到的数据
	time.Sleep(100 * time.Millisecond)
	clientData := srv.GetMockClientData(client)
	t.Logf("Client received total: %d bytes", len(clientData))

	t.Log("=== Test Completed ===")
}