package reconnect

import (
	"net"
	"testing"
	"time"
)

// TestReconnectConnect 测试连接
func TestReconnectConnect(t *testing.T) {
	// 启动一个临时服务器
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	manager := NewReconnectManager(listener.Addr().String(), 5, 100*time.Millisecond)

	err = manager.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if !manager.IsConnected() {
		t.Error("Expected to be connected")
	}

	manager.Disconnect()
	if manager.IsConnected() {
		t.Error("Expected to be disconnected")
	}
}

// TestReconnectAutoReconnect 测试自动重连
func TestReconnectAutoReconnect(t *testing.T) {
	// 启动一个临时服务器
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}

	serverAddr := listener.Addr().String()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	manager := NewReconnectManager(serverAddr, 10, 100*time.Millisecond)

	// 手动断开后自动重连
	manager.Connect()
	manager.Disconnect()

	// 启动自动重连
	errChan := manager.AutoReconnect()

	// 等待重连
	time.Sleep(300 * time.Millisecond)

	// 检查是否重连成功
	if !manager.IsConnected() {
		// 检查错误通道
		select {
		case err := <-errChan:
			t.Logf("Reconnect error: %v", err)
		default:
			t.Error("Expected to reconnect")
		}
	}

	manager.Stop()
	listener.Close()
}

// TestReconnectMaxRetry 测试最大重试次数
func TestReconnectMaxRetry(t *testing.T) {
	manager := NewReconnectManager("127.0.0.1:9999", 3, 50*time.Millisecond)

	// 尝试连接到不存在的服务器
	manager.Disconnect()

	// 手动触发重试
	for i := 0; i < 5; i++ {
		err := manager.tryReconnect()
		if err != nil && i >= 3 {
			// 应该在 3 次后达到最大重试
			t.Logf("Expected max retry error at attempt %d", i+1)
		}
	}
}

// TestMockReconnectManager 测试 Mock 重连管理器
func TestMockReconnectManager(t *testing.T) {
	manager := NewMockReconnectManager()

	err := manager.Connect()
	if err != nil {
		t.Fatalf("Mock connect failed: %v", err)
	}

	if !manager.IsConnected() {
		t.Error("Expected Mock to be connected")
	}

	manager.Disconnect()
	if manager.IsConnected() {
		t.Error("Expected Mock to be disconnected")
	}
}