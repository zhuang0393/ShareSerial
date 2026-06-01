package localproxy

import (
	"net"
	"sync"
	"testing"
	"time"
)

// MockConn Mock 连接
type MockConn struct {
	mu       sync.Mutex
	readBuf  []byte
	writeBuf []byte
	closed   bool
}

// NewMockConn 创建 Mock 连接
func NewMockConn() *MockConn {
	return &MockConn{
		readBuf:  make([]byte, 0),
		writeBuf: make([]byte, 0),
	}
}

// Read 读取数据
func (m *MockConn) Read(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.readBuf) == 0 {
		return 0, nil
	}

	n := len(b)
	if n > len(m.readBuf) {
		n = len(m.readBuf)
	}

	copy(b, m.readBuf[:n])
	m.readBuf = m.readBuf[n:]

	return n, nil
}

// Write 写入数据
func (m *MockConn) Write(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.writeBuf = append(m.writeBuf, b...)
	return len(b), nil
}

// Close 关闭连接
func (m *MockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// LocalAddr 本地地址
func (m *MockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}

// RemoteAddr 远程地址
func (m *MockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("192.168.1.100"), Port: 7700}
}

// SetDeadline 设置超时
func (m *MockConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline 设置读取超时
func (m *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline 设置写入超时
func (m *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// InjectData 注入数据（模拟远程数据）
func (m *MockConn) InjectData(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readBuf = append(m.readBuf, data...)
}

// GetWrittenData 获取写入的数据
func (m *MockConn) GetWrittenData() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeBuf
}

// TestLocalProxyCreate 测试创建本地代理
func TestLocalProxyCreate(t *testing.T) {
	proxy := NewLocalProxy(8889)

	if proxy.LocalPort() != 8889 {
		t.Errorf("Expected port 8889, got %d", proxy.LocalPort())
	}

	if proxy.LocalAddr() != "127.0.0.1:8889" {
		t.Errorf("Expected addr 127.0.0.1:8889, got %s", proxy.LocalAddr())
	}

	if proxy.IsRunning() {
		t.Error("Proxy should not be running initially")
	}
}

// TestLocalProxyStart 测试启动代理
func TestLocalProxyStart(t *testing.T) {
	// 创建 Mock 远程连接
	remoteConn := NewMockConn()

	proxy := NewLocalProxy(8889)
	err := proxy.Start(remoteConn)
	if err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}

	if !proxy.IsRunning() {
		t.Error("Proxy should be running after start")
	}

	// 等待监听启动
	time.Sleep(100 * time.Millisecond)

	// 尝试连接本地端口
	localConn, err := net.Dial("tcp", "127.0.0.1:8889")
	if err != nil {
		t.Fatalf("Failed to connect to local port: %v", err)
	}
	defer localConn.Close()

	// 检查连接数
	time.Sleep(100 * time.Millisecond)
	if proxy.ConnectionCount() != 1 {
		t.Errorf("Expected 1 connection, got %d", proxy.ConnectionCount())
	}

	// 停止代理
	proxy.Stop()

	if proxy.IsRunning() {
		t.Error("Proxy should not be running after stop")
	}
}

// TestLocalProxyDataForward 测试数据转发
func TestLocalProxyDataForward(t *testing.T) {
	remoteConn := NewMockConn()

	proxy := NewLocalProxy(8889)
	err := proxy.Start(remoteConn)
	if err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	// 等待启动
	time.Sleep(100 * time.Millisecond)

	// 连接本地端口
	localConn, err := net.Dial("tcp", "127.0.0.1:8889")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer localConn.Close()

	// 测试远程 -> 本地数据转发
	remoteConn.InjectData([]byte("REMOTE_DATA"))

	time.Sleep(200 * time.Millisecond)

	buf := make([]byte, 1024)
	localConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n, err := localConn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from local connection: %v", err)
	}

	if string(buf[:n]) != "REMOTE_DATA" {
		t.Errorf("Expected REMOTE_DATA, got %s", string(buf[:n]))
	}

	// 测试本地 -> 远程数据转发
	localConn.Write([]byte("LOCAL_DATA"))

	time.Sleep(200 * time.Millisecond)

	written := remoteConn.GetWrittenData()
	if string(written) != "LOCAL_DATA" {
		t.Errorf("Expected LOCAL_DATA in remote, got %s", string(written))
	}
}

// TestLocalProxyMultiConnection 测试多连接
func TestLocalProxyMultiConnection(t *testing.T) {
	remoteConn := NewMockConn()

	proxy := NewLocalProxy(8889)
	proxy.Start(remoteConn)
	defer proxy.Stop()

	time.Sleep(100 * time.Millisecond)

	// 连接多个本地客户端
	conns := make([]net.Conn, 3)
	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", "127.0.0.1:8889")
		if err != nil {
			t.Fatalf("Failed to connect %d: %v", i, err)
		}
		conns[i] = conn
	}

	time.Sleep(100 * time.Millisecond)

	// 检查连接数
	if proxy.ConnectionCount() != 3 {
		t.Errorf("Expected 3 connections, got %d", proxy.ConnectionCount())
	}

	// 广播数据到所有连接
	remoteConn.InjectData([]byte("BROADCAST"))

	time.Sleep(200 * time.Millisecond)

	// 验证所有连接收到数据
	for i, conn := range conns {
		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil {
			t.Errorf("Connection %d failed to read: %v", i, err)
		}
		if string(buf[:n]) != "BROADCAST" {
			t.Errorf("Connection %d: expected BROADCAST, got %s", i, string(buf[:n]))
		}
		conn.Close()
	}
}

// TestLocalProxyUpdateConnection 测试更新连接
func TestLocalProxyUpdateConnection(t *testing.T) {
	oldConn := NewMockConn()
	newConn := NewMockConn()

	proxy := NewLocalProxy(8889)
	proxy.Start(oldConn)
	defer proxy.Stop()

	time.Sleep(100 * time.Millisecond)

	// 更新连接
	proxy.UpdateRemoteConnection(newConn)

	// 等待更新生效
	time.Sleep(100 * time.Millisecond)

	// 新连接注入数据
	newConn.InjectData([]byte("NEW_CONN_DATA"))

	// 连接本地端口
	localConn, err := net.Dial("tcp", "127.0.0.1:8889")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer localConn.Close()

	time.Sleep(200 * time.Millisecond)

	buf := make([]byte, 1024)
	localConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n, err := localConn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if string(buf[:n]) != "NEW_CONN_DATA" {
		t.Errorf("Expected NEW_CONN_DATA, got %s", string(buf[:n]))
	}
}

// TestLocalProxyBuffer 测试数据缓冲
func TestLocalProxyBuffer(t *testing.T) {
	remoteConn := NewMockConn()

	proxy := NewLocalProxy(8889)
	proxy.Start(remoteConn)
	defer proxy.Stop()

	time.Sleep(100 * time.Millisecond)

	// 注入数据（没有本地连接）
	remoteConn.InjectData([]byte("BUFFERED_DATA"))

	time.Sleep(200 * time.Millisecond)

	// 检查缓冲
	if proxy.BufferSize() == 0 {
		t.Error("Expected data to be buffered")
	}

	// 连接本地端口（应该收到缓冲数据）
	localConn, err := net.Dial("tcp", "127.0.0.1:8889")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer localConn.Close()

	time.Sleep(200 * time.Millisecond)

	buf := make([]byte, 1024)
	localConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n, err := localConn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read buffered data: %v", err)
	}

	if string(buf[:n]) != "BUFFERED_DATA" {
		t.Errorf("Expected buffered data, got %s", string(buf[:n]))
	}
}
