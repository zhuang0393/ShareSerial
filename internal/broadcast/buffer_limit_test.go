package broadcast

import (
	"testing"
	"time"
)

// TestMockClientBufferSizeLimit 测试 MockClient 缓冲区大小限制
func TestMockClientBufferSizeLimit(t *testing.T) {
	// 创建一个小缓冲区的 MockClient
	mc := NewMockClient("test")
	mc.MaxBufferSize = 100 // 设置为 100 字节

	// 发送超过限制的数据
	largeData := make([]byte, 200)
	for i := range largeData {
		largeData[i] = byte(i)
	}

	mc.Send(largeData)
	time.Sleep(10 * time.Millisecond) // 等待处理

	data := mc.GetReceivedData()

	// 验证数据大小不超过限制
	if len(data) > mc.MaxBufferSize {
		t.Errorf("buffer size %d exceeds limit %d", len(data), mc.MaxBufferSize)
	}

	// 验证保留的是最新数据（后 100 字节）
	for i := 0; i < len(data); i++ {
		expected := largeData[100+i] // 应该是原始数据的后 100 字节
		if data[i] != expected {
			t.Errorf("data[%d] = %d, expected %d", i, data[i], expected)
		}
	}
}

// TestMockClientMultipleSendWithLimit 测试多次发送后的缓冲区限制
func TestMockClientMultipleSendWithLimit(t *testing.T) {
	mc := NewMockClient("test")
	mc.MaxBufferSize = 50 // 设置为 50 字节

	// 多次发送数据
	for i := 0; i < 10; i++ {
		data := make([]byte, 20)
		for j := range data {
			data[j] = byte(i)
		}
		mc.Send(data)
	}

	time.Sleep(50 * time.Millisecond) // 等待处理

	received := mc.GetReceivedData()

	// 验证大小不超过限制
	if len(received) > mc.MaxBufferSize {
		t.Errorf("buffer size %d exceeds limit %d", len(received), mc.MaxBufferSize)
	}

	// 验证保留的是最新数据（应该是最新的 2-3 次发送）
	// 最新发送的数据应该是 byte(9)
	if len(received) > 0 {
		lastByte := received[len(received)-1]
		if lastByte != 9 {
			t.Errorf("last byte should be 9 (latest send), got %d", lastByte)
		}
	}
}