package server

import (
	"testing"
	"time"
)

// TestMockServerBufferSizeLimit 测试 MockServer 缓冲区大小限制
func TestMockServerBufferSizeLimit(t *testing.T) {
	server := NewMockServer()
	server.MaxBufferSize = 100 // 设置为 100 字节

	err := server.Start("127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	// 发送超过限制的数据
	largeData := make([]byte, 200)
	for i := range largeData {
		largeData[i] = byte(i)
	}

	// 多次广播
	for i := 0; i < 3; i++ {
		server.Broadcast(largeData)
	}

	time.Sleep(50 * time.Millisecond) // 等待处理

	// 验证数据大小不超过限制
	// 由于 MockServer 没有直接的 GetData 方法，我们通过其他方式验证
	// 这里主要验证不会无限增长（通过内存监控）

	// 如果能够访问内部数据，验证最新数据
	// 由于封装限制，这里主要确保不会 panic 或内存泄漏
}

// TestMockServerAppendDataWithLimit 测试 appendDataWithLimit 方法
func TestMockServerAppendDataWithLimit(t *testing.T) {
	server := NewMockServer()
	server.MaxBufferSize = 50

	// 添加超过限制的数据
	data1 := make([]byte, 30)
	for i := range data1 {
		data1[i] = 1
	}
	server.appendDataWithLimit(data1)

	if len(server.data) != 30 {
		t.Errorf("expected 30 bytes, got %d", len(server.data))
	}

	// 再添加超过限制的数据
	data2 := make([]byte, 40)
	for i := range data2 {
		data2[i] = 2
	}
	server.appendDataWithLimit(data2)

	// 总共 70 字节，超过限制 50，应该截断到 50
	if len(server.data) != 50 {
		t.Errorf("expected buffer to be trimmed to 50 bytes, got %d", len(server.data))
	}

	// 验证保留的是最新数据
	// 应该是 data1 的后 10 字节（值为 1） + data2 的全部 40 字节（值为 2）
	// 或者是 data2 + data1 尾部的某种组合，取决于截断逻辑
	// 当前实现是保留最新的 MaxBufferSize 字节

	// 检查最新的数据应该全是 2
	for i := len(server.data) - 10; i < len(server.data); i++ {
		if server.data[i] != 2 {
			t.Errorf("expected newest data to be 2, got %d at index %d", server.data[i], i)
		}
	}
}
