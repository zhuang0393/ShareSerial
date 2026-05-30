package serial

import (
	"errors"
	"io"
	"sync"
)

// 串口配置常量
const (
	BaudRate   = 115200 // 固定波特率
	DataBits   = 8
	StopBits   = 1
)

// Parity 校验位类型
type Parity int

const (
	ParityNone Parity = 0
	ParityOdd  Parity = 1
	ParityEven Parity = 2
)

// Config 串口配置
type Config struct {
	BaudRate  int
	DataBits  int
	StopBits  int
	Parity    Parity
	ReadTimeout int
}

// DefaultConfig 返回默认配置（Phase 1 固定配置）
func DefaultConfig() *Config {
	return &Config{
		BaudRate:  BaudRate,
		DataBits:  DataBits,
		StopBits:  StopBits,
		Parity:    ParityNone,
		ReadTimeout: 0, // 无超时
	}
}

// Port 接口定义（便于 Mock）
type Port interface {
	Open(path string) error
	Close() error
	Read(buf []byte) (int, error)
	Write(buf []byte) (int, error)
	IsOpen() bool
}

// Handler 串口处理器
type Handler struct {
	port        Port
	config      *Config
	writeOwner  string
	mu          sync.Mutex
}

// NewHandler 创建串口处理器
func NewHandler(path string) (*Handler, error) {
	return NewHandlerWithConfig(path, DefaultConfig())
}

// NewHandlerWithConfig 使用自定义配置创建处理器
func NewHandlerWithConfig(path string, config *Config) (*Handler, error) {
	// Phase 1 暂时只支持 Mock，真实串口需要 go.bug.org/serial
	// 生产环境会替换为真实实现
	port := NewMockSerialPort()
	if err := port.Open(path); err != nil {
		return nil, err
	}
	return &Handler{
		port:   port,
		config: config,
	}, nil
}

// NewHandlerWithPort 使用已有 Port 创建处理器（用于测试）
func NewHandlerWithPort(port Port) *Handler {
	return &Handler{
		port:   port,
		config: DefaultConfig(),
	}
}

// Read 读取数据
func (h *Handler) Read(buf []byte) (int, error) {
	return h.port.Read(buf)
}

// Write 写入数据（需要检查写锁）
func (h *Handler) Write(buf []byte) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Phase 1 暂不强制检查锁，允许写入
	return h.port.Write(buf)
}

// WriteWithLock 带锁检查的写入
func (h *Handler) WriteWithLock(clientID string, buf []byte) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.writeOwner != clientID {
		return 0, errors.New("write lock not held by this client")
	}

	return h.port.Write(buf)
}

// SetWriteLockOwner 设置写锁持有者
func (h *Handler) SetWriteLockOwner(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.writeOwner = clientID
}

// ClearWriteLock 清除写锁
func (h *Handler) ClearWriteLock() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.writeOwner = ""
}

// Close 关闭处理器
func (h *Handler) Close() error {
	return h.port.Close()
}

// IsOpen 检查是否打开
func (h *Handler) IsOpen() bool {
	return h.port.IsOpen()
}

// MockSerialPort Mock 串口（用于测试）
type MockSerialPort struct {
	mu          sync.Mutex
	open        bool
	path        string
	inputBuffer []byte
	outputBuffer []byte
}

// NewMockSerialPort 创建 Mock 串口
func NewMockSerialPort() *MockSerialPort {
	return &MockSerialPort{
		inputBuffer: make([]byte, 0),
		outputBuffer: make([]byte, 0),
	}
}

// Open 打开 Mock 串口
func (m *MockSerialPort) Open(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.open = true
	m.path = path
	return nil
}

// Close 关闭 Mock 串口
func (m *MockSerialPort) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.open = false
	return nil
}

// Read 从 Mock 串口读取数据
func (m *MockSerialPort) Read(buf []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.inputBuffer) == 0 {
		return 0, io.EOF
	}

	n := copy(buf, m.inputBuffer)
	m.inputBuffer = m.inputBuffer[n:]
	return n, nil
}

// Write 向 Mock 串口写入数据
func (m *MockSerialPort) Write(buf []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.outputBuffer = append(m.outputBuffer, buf...)
	return len(buf), nil
}

// IsOpen 检查 Mock 串口是否打开
func (m *MockSerialPort) IsOpen() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.open
}

// InjectInput 注入模拟输入数据（用于测试）
func (m *MockSerialPort) InjectInput(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inputBuffer = append(m.inputBuffer, data...)
}

// GetWrittenData 获取写入的数据（用于测试）
func (m *MockSerialPort) GetWrittenData() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.outputBuffer
}

// Scanner 串口扫描器
type Scanner struct{}

// NewScanner 创建扫描器
func NewScanner() *Scanner {
	return &Scanner{}
}

// Scan 扫描可用串口
// 返回 /dev/ttyUSB* 和 /dev/ttyACM* 设备列表
func (s *Scanner) Scan() ([]string, error) {
	// Phase 1 暂返回空列表
	// 生产环境会实现真实扫描
	return nil, nil
}