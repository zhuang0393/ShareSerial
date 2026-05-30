package server

import (
	"net"
	"testing"
	"time"
)

// TestTCPServerStart 测试服务器启动
func TestTCPServerStart(t *testing.T) {
	server := NewMockServer()

	err := server.Start("127.0.0.1:0") // 使用随机端口
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !server.IsRunning() {
		t.Error("expected server to be running")
	}

	// 检查地址已分配
	addr := server.Addr()
	if addr == "" {
		t.Error("expected server to have an address")
	}

	server.Stop()
}

// TestTCPServerAcceptClient 测试接受客户端连接
func TestTCPServerAcceptClient(t *testing.T) {
	server := NewMockServer()
	server.Start("127.0.0.1:0")

	// 客户端连接
	conn, err := net.Dial("tcp", server.Addr())
	if err != nil {
		t.Fatalf("unexpected error connecting: %v", err)
	}
	defer conn.Close()

	// 等待服务器处理连接
	time.Sleep(50 * time.Millisecond)

	// 检查客户端已添加
	if server.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", server.ClientCount())
	}

	server.Stop()
}

// TestTCPServerDisconnectClient 测试断开客户端
func TestTCPServerDisconnectClient(t *testing.T) {
	server := NewMockServer()
	server.Start("127.0.0.1:0")

	// 客户端连接
	conn, _ := net.Dial("tcp", server.Addr())
	time.Sleep(50 * time.Millisecond)

	if server.ClientCount() != 1 {
		t.Fatalf("expected 1 client before disconnect")
	}

	// 客户端断开
	conn.Close()
	time.Sleep(50 * time.Millisecond)

	// 检查客户端已移除
	if server.ClientCount() != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", server.ClientCount())
	}

	server.Stop()
}

// TestTCPServerBroadcast 测试数据广播
func TestTCPServerBroadcast(t *testing.T) {
	server := NewMockServer()
	server.Start("127.0.0.1:0")

	// 两个客户端连接
	conn1, _ := net.Dial("tcp", server.Addr())
	conn2, _ := net.Dial("tcp", server.Addr())
	defer conn1.Close()
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)

	// 服务器广播数据
	data := []byte("Broadcast message\n")
	server.Broadcast(data)

	// 检查两个客户端都收到数据
	buf1 := make([]byte, 1024)
	conn1.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n1, _ := conn1.Read(buf1)

	buf2 := make([]byte, 1024)
	conn2.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n2, _ := conn2.Read(buf2)

	if string(buf1[:n1]) != string(data) {
		t.Errorf("client1: expected '%s', got '%s'", string(data), string(buf1[:n1]))
	}
	if string(buf2[:n2]) != string(data) {
		t.Errorf("client2: expected '%s', got '%s'", string(data), string(buf2[:n2]))
	}

	server.Stop()
}

// TestTCPServerWriteLock 测试写锁检查
func TestTCPServerWriteLock(t *testing.T) {
	server := NewMockServer()
	server.Start("127.0.0.1:0")

	// 客户端连接
	conn, _ := net.Dial("tcp", server.Addr())
	time.Sleep(50 * time.Millisecond)

	// 获取第一个客户端 ID
	clientID := server.GetClientIDs()[0]

	// Client1 获取写锁
	ok, _ := server.AcquireWriteLock(clientID)
	if !ok {
		t.Error("expected write lock to be acquired")
	}

	// 检查锁状态
	if !server.HasWriteLock() {
		t.Error("expected server to have write lock")
	}

	// Client1 释放写锁
	server.ReleaseWriteLock(clientID)

	// 检查锁已释放
	if server.HasWriteLock() {
		t.Error("expected write lock to be released")
	}

	conn.Close()
	server.Stop()
}

// TestTCPServerMultiClient 测试多客户端场景
func TestTCPServerMultiClient(t *testing.T) {
	server := NewMockServer()
	server.Start("127.0.0.1:0")

	// 5 个客户端连接
	conns := make([]net.Conn, 5)
	for i := 0; i < 5; i++ {
		conns[i], _ = net.Dial("tcp", server.Addr())
		defer conns[i].Close()
	}

	time.Sleep(100 * time.Millisecond)

	// 检查所有客户端已添加
	if server.ClientCount() != 5 {
		t.Errorf("expected 5 clients, got %d", server.ClientCount())
	}

	// 广播数据，检查所有客户端收到
	server.Broadcast([]byte("Multi broadcast\n"))

	for i, conn := range conns {
		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil {
			t.Errorf("client%d: unexpected error reading: %v", i+1, err)
		}
		if string(buf[:n]) != "Multi broadcast\n" {
			t.Errorf("client%d: expected broadcast data", i+1)
		}
	}

	server.Stop()
}

// TestTCPServerStop 测试服务器停止
func TestTCPServerStop(t *testing.T) {
	server := NewMockServer()
	server.Start("127.0.0.1:0")

	// 客户端连接
	_, _ = net.Dial("tcp", server.Addr())
	time.Sleep(50 * time.Millisecond)

	// 停止服务器
	server.Stop()

	if server.IsRunning() {
		t.Error("expected server to be stopped")
	}

	// 新连接应该失败
	_, err := net.Dial("tcp", server.Addr())
	if err == nil {
		t.Error("expected connection to fail after server stopped")
	}
}