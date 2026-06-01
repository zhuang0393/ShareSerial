package reconnect

import (
	"errors"
	"net"
	"sync"
	"time"
)

// ReconnectManager 断线重连管理器
type ReconnectManager struct {
	mu            sync.Mutex
	serverAddr    string
	conn          net.Conn
	connected     bool
	maxRetry      int
	retryCount    int
	retryInterval time.Duration
	stopChan      chan struct{}
}

// NewReconnectManager 创建重连管理器
func NewReconnectManager(serverAddr string, maxRetry int, retryInterval time.Duration) *ReconnectManager {
	return &ReconnectManager{
		serverAddr:    serverAddr,
		maxRetry:      maxRetry,
		retryInterval: retryInterval,
		stopChan:      make(chan struct{}),
	}
}

// Connect 连接服务器
func (r *ReconnectManager) Connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn, err := net.Dial("tcp", r.serverAddr)
	if err != nil {
		return err
	}

	r.conn = conn
	r.connected = true
	r.retryCount = 0
	return nil
}

// Disconnect 断开连接
func (r *ReconnectManager) Disconnect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil {
		r.conn.Close()
	}
	r.connected = false
	return nil
}

// IsConnected 检查连接状态
func (r *ReconnectManager) IsConnected() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.connected
}

// GetConnection 获取当前连接
func (r *ReconnectManager) GetConnection() net.Conn {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.conn
}

// AutoReconnect 自动重连（后台运行）
func (r *ReconnectManager) AutoReconnect() <-chan error {
	errChan := make(chan error, 1)

	go func() {
		for {
			select {
			case <-r.stopChan:
				return
			default:
				if !r.IsConnected() {
					err := r.tryReconnect()
					if err != nil {
						errChan <- err
					}
				}
				time.Sleep(r.retryInterval)
			}
		}
	}()

	return errChan
}

// tryReconnect 尝试重连
func (r *ReconnectManager) tryReconnect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.retryCount++

	// 检查是否超过最大重试次数
	if r.maxRetry > 0 && r.retryCount > r.maxRetry {
		return errors.New("max retry count exceeded")
	}

	conn, err := net.Dial("tcp", r.serverAddr)
	if err != nil {
		return err
	}

	r.conn = conn
	r.connected = true
	r.retryCount = 0
	return nil
}

// Stop 停止自动重连
func (r *ReconnectManager) Stop() {
	close(r.stopChan)
}

// MockReconnectManager Mock 重连管理器（用于测试）
type MockReconnectManager struct {
	mu        sync.Mutex
	connected bool
	conn      net.Conn
}

// NewMockReconnectManager 创建 Mock 重连管理器
func NewMockReconnectManager() *MockReconnectManager {
	return &MockReconnectManager{}
}

// Connect Mock 连接
func (m *MockReconnectManager) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

// Disconnect Mock 断开
func (m *MockReconnectManager) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

// IsConnected Mock 检查
func (m *MockReconnectManager) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

// SetConnection 设置连接
func (m *MockReconnectManager) SetConnection(conn net.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conn = conn
}

// GetConnection 获取连接
func (m *MockReconnectManager) GetConnection() net.Conn {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.conn
}

// AutoReconnect Mock 自动重连
func (m *MockReconnectManager) AutoReconnect() <-chan error {
	errChan := make(chan error, 1)
	return errChan
}

// Stop Mock 停止
func (m *MockReconnectManager) Stop() {}
