package server

import (
	"net"
	"sync"
	"time"

	"shareserial/pkg/arbiter"
	"shareserial/pkg/serial"
)

// Server 接口定义
type Server interface {
	Start(addr string) error
	Stop() error
	IsRunning() bool
	Addr() string
	ClientCount() int
	Broadcast(data []byte)
	AcquireWriteLock(clientID string) (bool, error)
	ReleaseWriteLock(clientID string)
	HasWriteLock() bool
	GetClientIDs() []string
}

// TCPServer TCP 服务器实现
type TCPServer struct {
	mu        sync.Mutex
	listener  net.Listener
	addr      string
	running   bool
	clients   map[string]*ClientConn
	arbiter   *arbiter.Arbiter
	serial    serial.Port
	broadcast chan []byte
}

// ClientConn 客户端连接
type ClientConn struct {
	id     string
	conn   net.Conn
	server *TCPServer
}

// NewTCPServer 创建 TCP 服务器
func NewTCPServer() *TCPServer {
	return &TCPServer{
		clients:   make(map[string]*ClientConn),
		arbiter:   arbiter.NewArbiter(30 * time.Second),
		broadcast: make(chan []byte, 1024),
	}
}

// Start 启动服务器
func (s *TCPServer) Start(addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.listener = listener
	s.addr = listener.Addr().String()
	s.running = true

	// 启动接受连接的 goroutine
	go s.acceptLoop()

	// 启动广播 goroutine
	go s.broadcastLoop()

	return nil
}

// acceptLoop 接受连接循环
func (s *TCPServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return // 服务器已停止
		}

		client := &ClientConn{
			id:     generateClientID(conn),
			conn:   conn,
			server: s,
		}

		s.mu.Lock()
		s.clients[client.id] = client
		s.mu.Unlock()

		// 启动客户端处理 goroutine
		go s.handleClient(client)
	}
}

// handleClient 处理客户端连接
func (s *TCPServer) handleClient(client *ClientConn) {
	// 读取客户端数据（用于写入串口）
	buf := make([]byte, 1024)
	for {
		n, err := client.conn.Read(buf)
		if err != nil {
			// 客户端断开
			s.mu.Lock()
			delete(s.clients, client.id)
			// 如果是写锁持有者，释放锁
			if s.arbiter.Owner() == client.id {
				s.arbiter.ForceRelease(client.id)
			}
			s.mu.Unlock()
			return
		}

		if n > 0 {
			// 自动获取写锁：如果当前没有写锁或锁持有者不是自己，尝试获取
			if s.arbiter.Owner() != client.id {
				// 尝试获取写锁（不阻塞，立即返回结果）
				acquired, _ := s.arbiter.Acquire(client.id)
				if !acquired {
					// 获取锁失败，等待锁释放
					for i := 0; i < 10; i++ { // 减少等待时间到 1 秒
						acquired, _ = s.arbiter.Acquire(client.id)
						if acquired {
							break
						}
						time.Sleep(100 * time.Millisecond)
					}
				}
			}

			// 写入串口
			if s.arbiter.Owner() == client.id && s.serial != nil {
				_, _ = s.serial.Write(buf[:n])

				// 【关键】写入后立即尝试读取串口响应
				// 使用小缓冲区和短超时避免阻塞
				responseBuf := make([]byte, 4096)
				// 尝试读取多次，确保获取完整响应
				for i := 0; i < 3; i++ {
					// 使用非阻塞方式读取（serial.Read 应该有超时机制）
					responseN, responseErr := s.serial.Read(responseBuf)
					if responseErr == nil && responseN > 0 {
						// 立即广播响应到所有客户端
						responseData := make([]byte, responseN)
						copy(responseData, responseBuf[:responseN])
						s.Broadcast(responseData)
						break // 读到响应后退出循环
					}
					// 没有响应，短暂等待后再次尝试
					time.Sleep(50 * time.Millisecond)
				}
			}
		}
	}
}

// broadcastLoop 广播数据循环
func (s *TCPServer) broadcastLoop() {
	for data := range s.broadcast {
		// 复制数据，避免并发写入同一内存
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		s.mu.Lock()
		clients := make([]*ClientConn, 0, len(s.clients))
		for _, client := range s.clients {
			clients = append(clients, client)
		}
		s.mu.Unlock()

		// 同步写入每个客户端，避免 goroutine 爆炸
		for _, client := range clients {
			_, _ = client.conn.Write(dataCopy)
		}
	}
}

// Stop 停止服务器
func (s *TCPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	// 关闭所有客户端连接
	for _, client := range s.clients {
		client.conn.Close()
	}
	s.clients = make(map[string]*ClientConn)

	// 关闭监听器
	if s.listener != nil {
		s.listener.Close()
	}

	// 关闭广播通道
	close(s.broadcast)

	s.running = false
	return nil
}

// IsRunning 检查服务器是否运行
func (s *TCPServer) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Addr 获取服务器地址
func (s *TCPServer) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addr
}

// ClientCount 获取客户端数量
func (s *TCPServer) ClientCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients)
}

// Broadcast 广播数据
func (s *TCPServer) Broadcast(data []byte) {
	s.broadcast <- data
}

// AcquireWriteLock 获取写锁
func (s *TCPServer) AcquireWriteLock(clientID string) (bool, error) {
	return s.arbiter.Acquire(clientID)
}

// ReleaseWriteLock 释放写锁
func (s *TCPServer) ReleaseWriteLock(clientID string) {
	_ = s.arbiter.Release(clientID)
}

// HasWriteLock 检查是否有写锁
func (s *TCPServer) HasWriteLock() bool {
	return s.arbiter.IsLocked()
}

// GetClientIDs 获取客户端 ID 列表
func (s *TCPServer) GetClientIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.clients))
	for id := range s.clients {
		ids = append(ids, id)
	}
	return ids
}

// SetSerial 设置串口
func (s *TCPServer) SetSerial(port serial.Port) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serial = port
}

// generateClientID 生成客户端 ID
func generateClientID(conn net.Conn) string {
	return conn.RemoteAddr().String()
}

// MockServer Mock 服务器（用于测试）
// 注意：MockServer 有最大容量限制（MaxBufferSize），防止内存无限增长
type MockServer struct {
	mu            sync.Mutex
	running       bool
	addr          string
	listener      net.Listener
	clients       map[string]net.Conn
	arbiter       *arbiter.Arbiter
	serial        serial.Port // 串口设备
	data          []byte
	MaxBufferSize int // 最大缓冲区大小（字节），超过后丢弃旧数据
}

// DefaultServerMaxBufferSize 默认最大缓冲区大小（1MB）
const DefaultServerMaxBufferSize = 1 * 1024 * 1024

// NewMockServer 创建 Mock 服务器
func NewMockServer() *MockServer {
	return &MockServer{
		clients:       make(map[string]net.Conn),
		arbiter:       arbiter.NewArbiter(30 * time.Second),
		data:          make([]byte, 0),
		MaxBufferSize: DefaultServerMaxBufferSize,
	}
}

// appendDataWithLimit 添加数据到缓冲区，超过限制时丢弃旧数据
func (m *MockServer) appendDataWithLimit(newData []byte) {
	// 添加新数据
	m.data = append(m.data, newData...)

	// 如果超过限制，丢弃旧数据
	if len(m.data) > m.MaxBufferSize {
		// 保留最新的 MaxBufferSize 字节
		m.data = m.data[len(m.data)-m.MaxBufferSize:]
	}
}

// Start 启动 Mock 服务器
func (m *MockServer) Start(addr string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 使用真实 TCP 监听
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	m.listener = listener
	m.addr = listener.Addr().String()
	m.running = true

	// 后台接受连接
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			clientAddr := conn.RemoteAddr().String()
			m.mu.Lock()
			m.clients[clientAddr] = conn
			m.mu.Unlock()

			// 处理连接
			go m.handleMockConn(conn)
		}
	}()

	return nil
}

// handleMockConn 处理 Mock 连接
func (m *MockServer) handleMockConn(conn net.Conn) {
	buf := make([]byte, 1024)
	clientAddr := conn.RemoteAddr().String()
	for {
		n, err := conn.Read(buf)
		if err != nil {
			m.mu.Lock()
			delete(m.clients, clientAddr)
			m.mu.Unlock()
			conn.Close()
			return
		}

		// 如果有写锁持有者，记录数据
		if m.arbiter.IsLocked() && m.arbiter.Owner() == clientAddr {
			m.mu.Lock()
			m.appendDataWithLimit(buf[:n])
			m.mu.Unlock()
		}
	}
}

// Stop 停止 Mock 服务器
func (m *MockServer) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false

	// 关闭所有客户端连接
	for _, conn := range m.clients {
		conn.Close()
	}
	m.clients = make(map[string]net.Conn)

	// 关闭监听器
	if m.listener != nil {
		m.listener.Close()
	}

	return nil
}

// IsRunning 检查是否运行
func (m *MockServer) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// Addr 获取地址
func (m *MockServer) Addr() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.addr
}

// ClientCount 获取客户端数量
func (m *MockServer) ClientCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.clients)
}

// Broadcast 广播数据（向所有客户端发送）
func (m *MockServer) Broadcast(data []byte) {
	m.mu.Lock()
	// 记录数据
	m.appendDataWithLimit(data)
	// 向所有客户端发送数据
	for _, conn := range m.clients {
		conn := conn // 创建局部变量避免闭包捕获循环变量
		go func() { _, _ = conn.Write(data) }()
	}
	m.mu.Unlock()
}

// AcquireWriteLock 获取写锁
func (m *MockServer) AcquireWriteLock(clientID string) (bool, error) {
	return m.arbiter.Acquire(clientID)
}

// ReleaseWriteLock 释放写锁
func (m *MockServer) ReleaseWriteLock(clientID string) {
	_ = m.arbiter.Release(clientID)
}

// HasWriteLock 检查是否有写锁
func (m *MockServer) HasWriteLock() bool {
	return m.arbiter.IsLocked()
}

// GetClientIDs 获取客户端 ID 列表
func (m *MockServer) GetClientIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]string, 0, len(m.clients))
	for id := range m.clients {
		ids = append(ids, id)
	}
	return ids
}

// NewMockServerWithSerial 创建带串口的 Mock 服务器（用于测试）
func NewMockServerWithSerial(arb *arbiter.Arbiter, serialPort serial.Port) *MockServer {
	return &MockServer{
		clients:       make(map[string]net.Conn),
		arbiter:       arb,
		data:          make([]byte, 0),
		MaxBufferSize: DefaultServerMaxBufferSize,
		serial:        serialPort,
	}
}

// MockClientConn Mock 客户端连接（用于测试）
type MockClientConn struct {
	id   string
	data []byte
}

// 实现 net.Conn 接口
func (m *MockClientConn) Read(b []byte) (n int, err error) {
	if len(m.data) == 0 {
		return 0, nil
	}
	n = copy(b, m.data)
	m.data = m.data[n:]
	return n, nil
}

func (m *MockClientConn) Write(b []byte) (n int, err error) {
	m.data = append(m.data, b...)
	return len(b), nil
}

func (m *MockClientConn) Close() error {
	return nil
}

func (m *MockClientConn) LocalAddr() net.Addr {
	return nil
}

func (m *MockClientConn) RemoteAddr() net.Addr {
	return nil
}

func (m *MockClientConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *MockClientConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockClientConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// AddMockClient 添加 Mock 客户端（用于测试）
func (m *MockServer) AddMockClient(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[clientID] = &MockClientConn{id: clientID, data: make([]byte, 0)}
}

// GetMockClientData 获取 Mock 客户端数据（用于测试）
func (m *MockServer) GetMockClientData(clientID string) []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conn, ok := m.clients[clientID]; ok {
		if mockConn, ok := conn.(*MockClientConn); ok {
			return mockConn.data
		}
	}
	return nil
}
