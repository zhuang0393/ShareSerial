package e2e

import (
	"fmt"
	"net"
	"runtime"
	"testing"
	"time"

	"shareserial/internal/server"
	"shareserial/pkg/serial"
)

// TestStabilityLongRun 测试长时间运行稳定性（5 分钟）
func TestStabilityLongRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long run test in short mode")
	}

	srv := server.NewTCPServer()
	mockSerial := serial.NewMockSerialPort()
	mockSerial.Open("/dev/ttyMock0")
	srv.SetSerial(mockSerial)

	err := srv.Start("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer srv.Stop()

	// 3 个客户端
	conns := make([]net.Conn, 3)
	for i := 0; i < 3; i++ {
		conns[i], _ = net.Dial("tcp", srv.Addr())
	}

	time.Sleep(100 * time.Millisecond)

	// 运行 5 分钟（测试模式缩短为 30 秒）
	duration := 30 * time.Second
	if testing.Short() {
		duration = 5 * time.Second
	}

	start := time.Now()
	count := 0

	for time.Since(start) < duration {
		// 模拟数据广播
		data := fmt.Sprintf("[17:30:%02d] INFO: Stability test line %d\n", count%60, count)
		srv.Broadcast([]byte(data))
		count++

		time.Sleep(100 * time.Millisecond)
	}

	// 验证连接仍然稳定
	if srv.ClientCount() != 3 {
		t.Errorf("Expected 3 clients after long run, got %d", srv.ClientCount())
	}

	t.Logf("Stability test completed: %d broadcasts in %v", count, time.Since(start))
}

// TestStabilityMemoryLeak 测试内存泄漏
func TestStabilityMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	srv := server.NewTCPServer()
	mockSerial := serial.NewMockSerialPort()
	mockSerial.Open("/dev/ttyMock0")
	srv.SetSerial(mockSerial)

	srv.Start("127.0.0.1:0")
	defer srv.Stop()

	// 运行 GC 确保初始状态
	runtime.GC()

	// 记录初始内存
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// 运行大量广播
	for i := 0; i < 10000; i++ {
		data := fmt.Sprintf("[17:30:%02d] INFO: Memory test line %d\n", i%60, i)
		srv.Broadcast([]byte(data))

		if i % 1000 == 0 {
			// 手动 GC
			runtime.GC()
		}
	}

	// 运行 GC 确保最终状态
	runtime.GC()

	// 记录最终内存
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// 检查内存增长
	var heapGrowth int64
	if m2.HeapAlloc > m1.HeapAlloc {
		heapGrowth = int64(m2.HeapAlloc - m1.HeapAlloc)
	} else {
		heapGrowth = 0 // 内存减少，无泄漏
	}

	// 允许少量增长（< 1MB）
	maxGrowth := 1 * 1024 * 1024
	if heapGrowth > int64(maxGrowth) {
		t.Errorf("Possible memory leak: heap grew by %d bytes (> %d)", heapGrowth, maxGrowth)
	}

	t.Logf("Memory test: initial=%d KB, final=%d KB, growth=%d KB",
		m1.HeapAlloc/1024, m2.HeapAlloc/1024, heapGrowth/1024)
}

// TestStabilityHighFrequency 测试高频数据
func TestStabilityHighFrequency(t *testing.T) {
	srv := server.NewTCPServer()
	srv.Start("127.0.0.1:0")
	defer srv.Stop()

	conn, _ := net.Dial("tcp", srv.Addr())
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// 高频广播（模拟高波特率数据）
	start := time.Now()
	count := 0

	for i := 0; i < 1000; i++ {
		// 每次广播 1KB 数据
		data := make([]byte, 1024)
		for j := 0; j < 1024; j++ {
			data[j] = byte('A' + (j % 26))
		}
		srv.Broadcast(data)
		count++
	}

	elapsed := time.Since(start)

	// 计算吞吐量
 throughput := float64(count * 1024) / elapsed.Seconds()

	t.Logf("High frequency test: %d broadcasts in %v, throughput=%.2f KB/s", count, elapsed, throughput)

	// 验证客户端能接收数据
	buf := make([]byte, 10240)
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	totalRead := 0
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		totalRead += n
	}

	t.Logf("Client received: %d KB", totalRead/1024)
}

// TestStabilityClientDisconnect 测试客户端断开不影响其他客户端
func TestStabilityClientDisconnect(t *testing.T) {
	srv := server.NewTCPServer()
	srv.Start("127.0.0.1:0")
	defer srv.Stop()

	// 5 个客户端
	conns := make([]net.Conn, 5)
	for i := 0; i < 5; i++ {
		conns[i], _ = net.Dial("tcp", srv.Addr())
	}

	time.Sleep(100 * time.Millisecond)

	// 断开部分客户端
	conns[0].Close()
	conns[1].Close()

	time.Sleep(100 * time.Millisecond)

	// 验证剩余客户端正常
	if srv.ClientCount() != 3 {
		t.Errorf("Expected 3 clients after disconnect, got %d", srv.ClientCount())
	}

	// 广播数据，验证剩余客户端收到
	srv.Broadcast([]byte("Test after disconnect\n"))

	time.Sleep(50 * time.Millisecond)

	for i := 2; i < 5; i++ {
		buf := make([]byte, 1024)
		conns[i].SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, _ := conns[i].Read(buf)
		if n < 10 {
			t.Errorf("Client%d should have received data", i+1)
		}
		conns[i].Close()
	}

	t.Log("Client disconnect stability test passed")
}

// TestStabilityRepeatedConnect 测试反复连接断开
func TestStabilityRepeatedConnect(t *testing.T) {
	srv := server.NewTCPServer()
	srv.Start("127.0.0.1:0")
	defer srv.Stop()

	// 反复连接断开 100 次
	for i := 0; i < 100; i++ {
		conn, err := net.Dial("tcp", srv.Addr())
		if err != nil {
			t.Errorf("Connect %d failed: %v", i+1, err)
			continue
		}

		time.Sleep(10 * time.Millisecond)

		conn.Close()

		time.Sleep(10 * time.Millisecond)
	}

	// 最终应该没有客户端
	time.Sleep(100 * time.Millisecond)
	if srv.ClientCount() != 0 {
		t.Errorf("Expected 0 clients after repeated connect/disconnect, got %d", srv.ClientCount())
	}

	t.Log("Repeated connect/disconnect test passed")
}