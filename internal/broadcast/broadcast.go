package broadcast

import (
	"sync"
	"time"
)

// Client 接口定义
type Client interface {
	ID() string
	Send(data []byte)
	GetReceivedData() []byte
}

// Broadcaster 数据广播器（One-to-Many）
type Broadcaster struct {
	mu      sync.RWMutex
	clients map[string]Client
	input   chan []byte
	wg      sync.WaitGroup
}

// NewBroadcaster 创建广播器
func NewBroadcaster() *Broadcaster {
	bc := &Broadcaster{
		clients: make(map[string]Client),
		input:   make(chan []byte, 1024),
	}

	// 启动后台处理
	go bc.process()

	return bc
}

// process 处理广播数据
func (bc *Broadcaster) process() {
	for data := range bc.input {
		bc.mu.RLock()
		// 向所有客户端发送数据
		for _, client := range bc.clients {
			go client.Send(data) // 每个 client 独立 goroutine
		}
		bc.mu.RUnlock()
	}
}

// AddClient 添加客户端
func (bc *Broadcaster) AddClient(client Client) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.clients[client.ID()] = client
}

// RemoveClient 移除客户端
func (bc *Broadcaster) RemoveClient(client Client) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	delete(bc.clients, client.ID())
}

// Broadcast 广播数据到所有客户端
func (bc *Broadcaster) Broadcast(data []byte) {
	bc.input <- data
}

// ClientCount 返回客户端数量
func (bc *Broadcaster) ClientCount() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.clients)
}

// Close 关闭广播器
func (bc *Broadcaster) Close() {
	close(bc.input)
	bc.wg.Wait()
}

// MockClient Mock 客户端（用于测试）
// 注意：MockClient 有最大容量限制（MaxBufferSize），防止内存无限增长
type MockClient struct {
	id            string
	delay         time.Duration
	received      []byte
	mu            sync.Mutex
	sendQueue     chan []byte
	MaxBufferSize int // 最大缓冲区大小（字节），超过后丢弃旧数据
}

// DefaultMaxBufferSize 默认最大缓冲区大小（1MB）
const DefaultMaxBufferSize = 1 * 1024 * 1024

// NewMockClient 创建 Mock 客户端
func NewMockClient(id string) *MockClient {
	mc := &MockClient{
		id:            id,
		received:      make([]byte, 0),
		sendQueue:     make(chan []byte, 1024),
		MaxBufferSize: DefaultMaxBufferSize,
	}

	// 启动接收处理
	go mc.processQueue()

	return mc
}

// NewMockClientWithDelay 创建带延迟的 Mock 客户端
func NewMockClientWithDelay(id string, delay time.Duration) *MockClient {
	mc := &MockClient{
		id:            id,
		delay:         delay,
		received:      make([]byte, 0),
		sendQueue:     make(chan []byte, 1024),
		MaxBufferSize: DefaultMaxBufferSize,
	}

	// 启动带延迟的接收处理
	go mc.processQueueWithDelay()

	return mc
}

// appendWithLimit 添加数据到缓冲区，超过限制时丢弃旧数据
func (mc *MockClient) appendWithLimit(data []byte) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// 添加新数据
	mc.received = append(mc.received, data...)

	// 如果超过限制，丢弃旧数据
	if len(mc.received) > mc.MaxBufferSize {
		// 保留最新的 MaxBufferSize 字节
		mc.received = mc.received[len(mc.received)-mc.MaxBufferSize:]
	}
}

// processQueue 处理接收队列
func (mc *MockClient) processQueue() {
	for data := range mc.sendQueue {
		mc.appendWithLimit(data)
	}
}

// processQueueWithDelay 处理接收队列（带延迟）
func (mc *MockClient) processQueueWithDelay() {
	for data := range mc.sendQueue {
		time.Sleep(mc.delay)
		mc.appendWithLimit(data)
	}
}

// ID 返回客户端 ID
func (mc *MockClient) ID() string {
	return mc.id
}

// Send 发送数据到客户端
func (mc *MockClient) Send(data []byte) {
	mc.sendQueue <- data
}

// GetReceivedData 获取接收的数据
func (mc *MockClient) GetReceivedData() []byte {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.received
}
