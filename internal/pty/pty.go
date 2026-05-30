package pty

import (
	"errors"
	"sync"
	"time"
)

// Parity 校验位类型
type Parity int

const (
	ParityNone Parity = 0
	ParityOdd  Parity = 1
	ParityEven Parity = 2
)

// TermiosConfig termios 配置
type TermiosConfig struct {
	BaudRate int
	DataBits int
	StopBits int
	Parity   Parity
}

// PTYDevice PTY 设备接口
type PTYDevice interface {
	SymlinkPath() string
	Read(buf []byte) (int, error)
	Write(buf []byte) (int, error)
	Close() error
	IsOpen() bool
	SetTermios(config *TermiosConfig) error
	GetTermios() *TermiosConfig
}

// MockPTY Mock PTY 设备（用于测试）
type MockPTY struct {
	mu           sync.Mutex
	symlinkPath  string
	masterBuffer []byte
	slaveBuffer  []byte
	externalData []byte
	open         bool
	termios      *TermiosConfig
	dataReady    chan struct{} // 通知有数据可用
}

// CreateMockPTY 创建 Mock PTY
func CreateMockPTY(symlinkPath string) (*MockPTY, error) {
	return &MockPTY{
		symlinkPath:  symlinkPath,
		masterBuffer: make([]byte, 0),
		slaveBuffer:  make([]byte, 0),
		externalData: make([]byte, 0),
		open:         true,
		termios: &TermiosConfig{
			BaudRate: 115200,
			DataBits: 8,
			StopBits: 1,
			Parity:   ParityNone,
		},
		dataReady: make(chan struct{}, 1),
	}, nil
}

// SymlinkPath 返回 symlink 路径
func (m *MockPTY) SymlinkPath() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.symlinkPath
}

// Read 从 master 读取数据（服务端接收用户输入）
func (m *MockPTY) Read(buf []byte) (int, error) {
	m.mu.Lock()
	if !m.open {
		m.mu.Unlock()
		return 0, errors.New("PTY is closed")
	}
	m.mu.Unlock()

	// 如果没有数据，阻塞等待
	m.mu.Lock()
	for len(m.externalData) == 0 && m.open {
		m.mu.Unlock()
		// 等待数据通知
		select {
		case <-m.dataReady:
		case <-time.After(100 * time.Millisecond):
			// 短超时，定期检查状态
		}
		m.mu.Lock()
	}
	m.mu.Unlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.open {
		return 0, errors.New("PTY is closed")
	}

	if len(m.externalData) == 0 {
		return 0, nil // 没有数据，返回 0（不应该到这里）
	}

	n := copy(buf, m.externalData)
	m.externalData = m.externalData[n:]
	return n, nil
}

// ReadFromSlave 从 slave 读取数据（用户读取服务端数据）
func (m *MockPTY) ReadFromSlave(buf []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.open {
		return 0, errors.New("PTY is closed")
	}

	if len(m.slaveBuffer) == 0 {
		return 0, errors.New("no data available")
	}

	n := copy(buf, m.slaveBuffer)
	m.slaveBuffer = m.slaveBuffer[n:]
	return n, nil
}

// Write 写入数据到 master（服务端发送数据）
func (m *MockPTY) Write(buf []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.open {
		return 0, errors.New("PTY is closed")
	}

	m.slaveBuffer = append(m.slaveBuffer, buf...)
	m.masterBuffer = append(m.masterBuffer, buf...)
	return len(buf), nil
}

// InjectExternalData 注入外部数据（模拟用户输入）
func (m *MockPTY) InjectExternalData(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.externalData = append(m.externalData, data...)
	// 通知有数据可用
	select {
	case m.dataReady <- struct{}{}:
	default:
		// 已经有通知，不需要再发
	}
}

// GetSlaveData 获取 slave 缓冲数据（用于测试）
func (m *MockPTY) GetSlaveData() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.slaveBuffer
}

// Close 关闭 PTY
func (m *MockPTY) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.open = false
	return nil
}

// IsOpen 检查是否打开
func (m *MockPTY) IsOpen() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.open
}

// SetTermios 设置 termios 配置
func (m *MockPTY) SetTermios(config *TermiosConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.termios = config
	return nil
}

// GetTermios 获取 termios 配置
func (m *MockPTY) GetTermios() *TermiosConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.termios
}

// CreatePTY 创建 PTY（Linux 真实实现，其他平台 Mock）
// Linux 平台请使用 real_pty_linux.go 中的 CreatePTY