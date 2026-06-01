package server

import (
	"net"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestTCPServerHighFrequencyBroadcast 测试高频广播不会导致 goroutine 爆炸
func TestTCPServerHighFrequencyBroadcast(t *testing.T) {
	server := NewTCPServer()
	err := server.Start("127.0.0.1:0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 连接多个客户端
	numClients := 5
	conns := make([]net.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conn, err := net.Dial("tcp", server.Addr())
		if err != nil {
			t.Fatalf("unexpected error connecting: %v", err)
		}
		conns[i] = conn
		defer conn.Close()
	}

	time.Sleep(100 * time.Millisecond)

	// 记录初始 goroutine 数量
	initialGoroutines := runtime.NumGoroutine()

	// 高频广播（1000 次）
	data := []byte("High frequency broadcast test\n")
	for i := 0; i < 1000; i++ {
		server.Broadcast(data)
	}

	// 等待处理完成
	time.Sleep(500 * time.Millisecond)

	// 检查 goroutine 数量不应该爆炸式增长
	finalGoroutines := runtime.NumGoroutine()
	growth := finalGoroutines - initialGoroutines

	// 允许少量增长（正常情况），但不能爆炸（之前 bug 会创建数百万 goroutine）
	if growth > 100 {
		t.Errorf("goroutine count exploded: initial=%d, final=%d, growth=%d",
			initialGoroutines, finalGoroutines, growth)
	}

	server.Stop()
}

// TestTCPServerConcurrentBroadcast 测试并发广播安全
func TestTCPServerConcurrentBroadcast(t *testing.T) {
	server := NewTCPServer()
	server.Start("127.0.0.1:0")

	// 连接客户端
	conn, _ := net.Dial("tcp", server.Addr())
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	// 并发广播
	var wg sync.WaitGroup
	numBroadcasts := 100
	wg.Add(numBroadcasts)

	for i := 0; i < numBroadcasts; i++ {
		go func(id int) {
			defer wg.Done()
			server.Broadcast([]byte("Concurrent broadcast\n"))
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// 检查服务器仍然运行
	if !server.IsRunning() {
		t.Error("server should still be running after concurrent broadcasts")
	}

	server.Stop()
}

// TestTCPServerBroadcastToMultipleClients 测试广播到多个客户端
func TestTCPServerBroadcastToMultipleClients(t *testing.T) {
	server := NewTCPServer()
	server.Start("127.0.0.1:0")

	// 连接多个客户端
	numClients := 10
	conns := make([]net.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conn, err := net.Dial("tcp", server.Addr())
		if err != nil {
			t.Fatalf("unexpected error connecting: %v", err)
		}
		conns[i] = conn
		defer conn.Close()
	}

	time.Sleep(100 * time.Millisecond)

	// 广播数据
	data := []byte("Broadcast to all\n")
	server.Broadcast(data)
	time.Sleep(100 * time.Millisecond)

	// 检查所有客户端收到数据
	for i, conn := range conns {
		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil {
			t.Errorf("client%d: unexpected error reading: %v", i, err)
			continue
		}
		if string(buf[:n]) != string(data) {
			t.Errorf("client%d: expected '%s', got '%s'", i, string(data), string(buf[:n]))
		}
	}

	server.Stop()
}

// TestTCPServerClientDisconnectDuringBroadcast 测试广播期间客户端断开
func TestTCPServerClientDisconnectDuringBroadcast(t *testing.T) {
	server := NewTCPServer()
	server.Start("127.0.0.1:0")

	// 连接客户端
	conn1, _ := net.Dial("tcp", server.Addr())
	conn2, _ := net.Dial("tcp", server.Addr())
	defer conn2.Close()
	time.Sleep(50 * time.Millisecond)

	// 广播期间断开一个客户端
	go func() {
		time.Sleep(10 * time.Millisecond)
		conn1.Close()
	}()

	// 多次广播
	for i := 0; i < 100; i++ {
		server.Broadcast([]byte("Broadcast during disconnect\n"))
	}

	time.Sleep(100 * time.Millisecond)

	// 服务器应该仍然正常运行
	if !server.IsRunning() {
		t.Error("server should still be running after client disconnect during broadcast")
	}

	// conn2 应该仍然能收到数据
	buf := make([]byte, 1024)
	conn2.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n, err := conn2.Read(buf)
	if err != nil {
		t.Errorf("conn2: unexpected error reading: %v", err)
	}
	if n == 0 {
		t.Error("conn2: expected to receive data")
	}

	server.Stop()
}

// TestTCPServerLongRunning 测试长时间运行稳定性
func TestTCPServerLongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long running test in short mode")
	}

	server := NewTCPServer()
	server.Start("127.0.0.1:0")

	conn, _ := net.Dial("tcp", server.Addr())
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	// 模拟 5 秒的持续广播
	duration := 5 * time.Second
	data := []byte("Continuous data stream\n")
	start := time.Now()

	for time.Since(start) < duration {
		server.Broadcast(data)
		time.Sleep(10 * time.Millisecond) // 100 Hz
	}

	time.Sleep(100 * time.Millisecond)

	// 检查服务器状态
	if !server.IsRunning() {
		t.Error("server should still be running")
	}

	// 检查 goroutine 数量
	goroutines := runtime.NumGoroutine()
	if goroutines > 50 {
		t.Errorf("too many goroutines after long run: %d", goroutines)
	}

	server.Stop()
}
