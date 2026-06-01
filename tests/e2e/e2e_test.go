package e2e

import (
	"net"
	"testing"
	"time"

	"shareserial/internal/cli"
	"shareserial/internal/server"
	"shareserial/pkg/arbiter"
	"shareserial/pkg/serial"
)

// TestE2EFullFlow 测试完整流程
func TestE2EFullFlow(t *testing.T) {
	// 1. 启动服务端
	srv := server.NewTCPServer()
	mockSerial := serial.NewMockSerialPort()
	mockSerial.Open("/dev/ttyMock0")
	srv.SetSerial(mockSerial)

	err := srv.Start("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer srv.Stop()

	// 2. 客户端连接
	conn1, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("Failed to connect client1: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("Failed to connect client2: %v", err)
	}
	defer conn2.Close()

	time.Sleep(100 * time.Millisecond)

	// 验证两个客户端已连接
	if srv.ClientCount() != 2 {
		t.Errorf("Expected 2 clients, got %d", srv.ClientCount())
	}

	// 3. 模拟串口数据
	mockSerial.InjectInput([]byte("[17:30:00] INFO: System starting\n"))
	mockSerial.InjectInput([]byte("[17:30:01] ERROR: Failed to mount\n"))

	// 4. 服务端广播
	srv.Broadcast([]byte("[17:30:00] INFO: System starting\n"))
	srv.Broadcast([]byte("[17:30:01] ERROR: Failed to mount\n"))

	time.Sleep(100 * time.Millisecond)

	// 5. 验证客户端收到数据
	buf1 := make([]byte, 1024)
	conn1.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n1, _ := conn1.Read(buf1)

	if n1 < 20 {
		t.Errorf("Client1 received too little data: %d bytes", n1)
	}

	buf2 := make([]byte, 1024)
	conn2.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n2, _ := conn2.Read(buf2)

	if n2 < 20 {
		t.Errorf("Client2 received too little data: %d bytes", n2)
	}

	t.Logf("E2E full flow completed: client1=%d bytes, client2=%d bytes", n1, n2)
}

// TestE2EMultiClient 测试多客户端场景
func TestE2EMultiClient(t *testing.T) {
	srv := server.NewTCPServer()
	srv.Start("127.0.0.1:0")
	defer srv.Stop()

	// 5 个客户端连接
	conns := make([]net.Conn, 5)
	for i := 0; i < 5; i++ {
		conn, err := net.Dial("tcp", srv.Addr())
		if err != nil {
			t.Fatalf("Failed to connect client%d: %v", i+1, err)
		}
		conns[i] = conn
	}

	time.Sleep(100 * time.Millisecond)

	if srv.ClientCount() != 5 {
		t.Errorf("Expected 5 clients, got %d", srv.ClientCount())
	}

	// 广播数据
	data := []byte("Broadcast to all clients\n")
	srv.Broadcast(data)

	time.Sleep(100 * time.Millisecond)

	// 验证所有客户端收到
	for i, conn := range conns {
		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil {
			t.Errorf("Client%d failed to read: %v", i+1, err)
		}
		if n != len(data) {
			t.Errorf("Client%d: expected %d bytes, got %d", i+1, len(data), n)
		}
		conn.Close()
	}

	t.Log("E2E multi-client test passed")
}

// TestE2EWriteLock 测试写锁功能
func TestE2EWriteLock(t *testing.T) {
	srv := server.NewTCPServer()
	mockSerial := serial.NewMockSerialPort()
	mockSerial.Open("/dev/ttyMock0")
	srv.SetSerial(mockSerial)

	srv.Start("127.0.0.1:0")
	defer srv.Stop()

	// 两个客户端
	conn1, _ := net.Dial("tcp", srv.Addr())
	conn2, _ := net.Dial("tcp", srv.Addr())
	defer conn1.Close()
	defer conn2.Close()

	time.Sleep(100 * time.Millisecond)

	// 获取客户端 ID
	ids := srv.GetClientIDs()
	if len(ids) < 2 {
		t.Fatal("Expected at least 2 clients")
	}

	clientID1 := ids[0]

	// Client1 获取写锁
	arb := arbiter.NewArbiter(30 * time.Second)
	ok, _ := arb.Acquire(clientID1)
	if !ok {
		t.Error("Expected write lock to be acquired")
	}

	// Client2 尝试获取写锁（应该失败）
	ok, _ = arb.Acquire(ids[1])
	if ok {
		t.Error("Expected write lock acquisition to fail (already locked)")
	}

	// Client1 释放写锁
	arb.Release(clientID1)

	// Client2 现在可以获取写锁
	ok, _ = arb.Acquire(ids[1])
	if !ok {
		t.Error("Expected Client2 to acquire lock after release")
	}

	t.Log("E2E write lock test passed")
}

// TestE2ECLIIntegration 测试 CLI 集成
func TestE2ECLIIntegration(t *testing.T) {
	// 使用 Mock CLI 测试
	mockCLI := cli.NewMockCLI()

	// 设置 Log 数据
	mockCLI.SetLogData([]string{
		"[17:30:00] INFO: System starting",
		"[17:30:01] ERROR: Failed to mount",
		"[17:30:02] WARN: Low memory",
	})

	// 测试 log 命令
	output, err := mockCLI.Execute("log")
	if err != nil {
		t.Fatalf("CLI log command failed: %v", err)
	}
	if len(output) < 30 {
		t.Errorf("Expected more log output, got: %s", output)
	}

	// 测试过滤
	output, err = mockCLI.Execute("log", "--filter", "ERROR")
	if err != nil {
		t.Fatalf("CLI filter failed: %v", err)
	}
	if len(output) == 0 {
		t.Error("Expected filtered output")
	}

	// 测试 JSON 格式
	output, err = mockCLI.Execute("log", "--format", "json")
	if err != nil {
		t.Fatalf("CLI JSON format failed: %v", err)
	}
	if output == "[]" || len(output) < 50 {
		t.Errorf("Expected JSON output, got: %s", output)
	}

	// 测试 status 命令
	mockCLI.SetStatus(map[string]interface{}{
		"server":    "192.168.1.100:7700",
		"connected": true,
	})
	output, err = mockCLI.Execute("status")
	if err != nil {
		t.Fatalf("CLI status command failed: %v", err)
	}

	t.Log("E2E CLI integration test passed")
}

// TestE2EDisconnectReconnect 测试断线重连
func TestE2EDisconnectReconnect(t *testing.T) {
	srv := server.NewTCPServer()
	srv.Start("127.0.0.1:0")
	defer srv.Stop()

	// 客户端连接
	conn, _ := net.Dial("tcp", srv.Addr())
	time.Sleep(50 * time.Millisecond)

	if srv.ClientCount() != 1 {
		t.Fatal("Expected 1 client before disconnect")
	}

	// 客户端断开
	conn.Close()
	time.Sleep(100 * time.Millisecond)

	if srv.ClientCount() != 0 {
		t.Errorf("Expected 0 clients after disconnect, got %d", srv.ClientCount())
	}

	// 重连
	conn2, _ := net.Dial("tcp", srv.Addr())
	time.Sleep(50 * time.Millisecond)

	if srv.ClientCount() != 1 {
		t.Errorf("Expected 1 client after reconnect, got %d", srv.ClientCount())
	}
	conn2.Close()

	t.Log("E2E disconnect/reconnect test passed")
}

// TestE2EPerformance 测试性能（延迟）
func TestE2EPerformance(t *testing.T) {
	srv := server.NewTCPServer()
	srv.Start("127.0.0.1:0")
	defer srv.Stop()

	conn, _ := net.Dial("tcp", srv.Addr())
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// 测量延迟
	data := []byte("Performance test data\n")

	start := time.Now()
	srv.Broadcast(data)

	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	conn.Read(buf)

	elapsed := time.Since(start)

	// 验证延迟 < 10ms
	if elapsed > 10*time.Millisecond {
		t.Errorf("Expected latency < 10ms, got %v", elapsed)
	}

	t.Logf("E2E performance test: latency=%v", elapsed)
}
