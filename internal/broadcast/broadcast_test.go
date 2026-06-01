package broadcast

import (
	"testing"
	"time"
)

// TestBroadcasterCreate 测试广播器创建
func TestBroadcasterCreate(t *testing.T) {
	bc := NewBroadcaster()
	if bc == nil {
		t.Fatal("expected broadcaster to be created")
	}
}

// TestBroadcasterOneToMany 测试多客户端广播
func TestBroadcasterOneToMany(t *testing.T) {
	bc := NewBroadcaster()

	// 创建两个 Mock 客户端
	client1 := NewMockClient("client1")
	client2 := NewMockClient("client2")

	bc.AddClient(client1)
	bc.AddClient(client2)

	// 广播数据
	data := []byte("Test broadcast data\n")
	bc.Broadcast(data)

	// 等待数据处理完成
	time.Sleep(50 * time.Millisecond)

	// 检查两个客户端都收到数据
	received1 := client1.GetReceivedData()
	received2 := client2.GetReceivedData()

	if string(received1) != string(data) {
		t.Errorf("client1: expected '%s', got '%s'", string(data), string(received1))
	}
	if string(received2) != string(data) {
		t.Errorf("client2: expected '%s', got '%s'", string(data), string(received2))
	}
}

// TestBroadcasterClientQueue 测试客户端独立队列
func TestBroadcasterClientQueue(t *testing.T) {
	bc := NewBroadcaster()

	client := NewMockClient("client")
	bc.AddClient(client)

	// 广播多条数据
	bc.Broadcast([]byte("Line 1\n"))
	bc.Broadcast([]byte("Line 2\n"))
	bc.Broadcast([]byte("Line 3\n"))

	// 等待数据处理完成
	time.Sleep(50 * time.Millisecond)

	// 检查所有数据都被接收（由于并发，顺序可能不同）
	received := client.GetReceivedData()
	expectedLen := len("Line 1\n") + len("Line 2\n") + len("Line 3\n")

	if len(received) != expectedLen {
		t.Errorf("expected %d bytes, got %d", expectedLen, len(received))
	}

	// 检查所有三行都被接收
	if !containsAll(received, "Line 1\n", "Line 2\n", "Line 3\n") {
		t.Errorf("expected all lines to be received, got '%s'", string(received))
	}
}

func containsAll(data []byte, lines ...string) bool {
	for _, line := range lines {
		if !contains(data, line) {
			return false
		}
	}
	return true
}

func contains(data []byte, substr string) bool {
	return len(data) >= len(substr) &&
		(string(data)[:len(substr)] == substr ||
			string(data)[len(data)-len(substr):] == substr ||
			indexOf(data, substr) >= 0)
}

func indexOf(data []byte, substr string) int {
	for i := 0; i <= len(data)-len(substr); i++ {
		if string(data[i:i+len(substr)]) == substr {
			return i
		}
	}
	return -1
}

// TestBroadcasterSlowClient 测试慢客户端不阻塞快客户端
func TestBroadcasterSlowClient(t *testing.T) {
	bc := NewBroadcaster()

	// 创建快客户端和慢客户端
	fastClient := NewMockClient("fast")
	slowClient := NewMockClientWithDelay("slow", 50*time.Millisecond)

	bc.AddClient(fastClient)
	bc.AddClient(slowClient)

	// 广播数据（启动后台处理）
	data := []byte("Test data\n")
	bc.Broadcast(data)

	// 快客户端应该很快收到数据
	time.Sleep(10 * time.Millisecond)
	fastReceived := fastClient.GetReceivedData()

	if len(fastReceived) == 0 {
		t.Error("fast client should have received data immediately")
	}

	// 检查慢客户端最终也收到数据
	time.Sleep(100 * time.Millisecond)
	slowReceived := slowClient.GetReceivedData()

	if string(slowReceived) != string(data) {
		t.Errorf("slow client: expected '%s', got '%s'", string(data), string(slowReceived))
	}
}

// TestBroadcasterRemoveClient 测试移除客户端
func TestBroadcasterRemoveClient(t *testing.T) {
	bc := NewBroadcaster()

	client := NewMockClient("client")
	bc.AddClient(client)

	// 广播一次
	bc.Broadcast([]byte("Before remove\n"))

	// 等待数据处理完成
	time.Sleep(50 * time.Millisecond)

	if len(client.GetReceivedData()) == 0 {
		t.Fatal("client should have received first broadcast")
	}

	// 移除客户端
	bc.RemoveClient(client)

	// 广播第二次
	bc.Broadcast([]byte("After remove\n"))

	// 等待数据处理完成
	time.Sleep(50 * time.Millisecond)

	// 检查客户端没有收到第二次广播
	received := client.GetReceivedData()
	if string(received) != "Before remove\n" {
		t.Errorf("client should not receive data after removal, got '%s'", string(received))
	}
}

// TestBroadcasterClientCount 测试客户端计数
func TestBroadcasterClientCount(t *testing.T) {
	bc := NewBroadcaster()

	if bc.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", bc.ClientCount())
	}

	client1 := NewMockClient("client1")
	client2 := NewMockClient("client2")

	bc.AddClient(client1)
	if bc.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", bc.ClientCount())
	}

	bc.AddClient(client2)
	if bc.ClientCount() != 2 {
		t.Errorf("expected 2 clients, got %d", bc.ClientCount())
	}

	bc.RemoveClient(client1)
	if bc.ClientCount() != 1 {
		t.Errorf("expected 1 client after removal, got %d", bc.ClientCount())
	}
}

// TestBroadcasterConcurrentBroadcast 测试并发广播
func TestBroadcasterConcurrentBroadcast(t *testing.T) {
	bc := NewBroadcaster()
	client := NewMockClient("client")
	bc.AddClient(client)

	// 并发广播多条数据
	for i := 0; i < 10; i++ {
		go bc.Broadcast([]byte("Concurrent line\n"))
	}

	// 等待所有广播完成
	time.Sleep(100 * time.Millisecond)

	// 检查客户端收到所有数据（10 条）
	received := client.GetReceivedData()
	expectedLen := 10 * len("Concurrent line\n")

	if len(received) < expectedLen {
		t.Errorf("expected at least %d bytes, got %d", expectedLen, len(received))
	}
}
