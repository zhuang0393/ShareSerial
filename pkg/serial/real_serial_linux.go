//go:build linux
// +build linux

package serial

import (
	"go.bug.st/serial"
	"sync"
)

// RealSerialPort 真实串口实现
type RealSerialPort struct {
	mu     sync.Mutex
	port   serial.Port
	open   bool
	path   string
	config *Config
}

// NewRealSerialPort 创建真实串口
func NewRealSerialPort() *RealSerialPort {
	return &RealSerialPort{
		config: DefaultConfig(),
	}
}

// Open 打开真实串口
func (r *RealSerialPort) Open(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 创建串口模式
	mode := &serial.Mode{
		BaudRate: r.config.BaudRate,
	}

	// 打开串口（返回接口类型）
	port, err := serial.Open(path, mode)
	if err != nil {
		return err
	}

	r.port = port
	r.path = path
	r.open = true
	return nil
}

// Close 关闭串口
func (r *RealSerialPort) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.port != nil {
		r.port.Close()
	}
	r.open = false
	return nil
}

// Read 读取数据
func (r *RealSerialPort) Read(buf []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.open || r.port == nil {
		return 0, ErrPortNotOpen
	}

	return r.port.Read(buf)
}

// Write 写入数据
func (r *RealSerialPort) Write(buf []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.open || r.port == nil {
		return 0, ErrPortNotOpen
	}

	return r.port.Write(buf)
}

// IsOpen 检查是否打开
func (r *RealSerialPort) IsOpen() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.open
}

// SetConfig 设置配置
func (r *RealSerialPort) SetConfig(config *Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.config = config

	if r.port == nil {
		return nil
	}

	// 设置新模式
	mode := &serial.Mode{
		BaudRate: config.BaudRate,
	}

	return r.port.SetMode(mode)
}

// NewPort 创建串口（优先使用真实实现）
func NewPort(path string) (Port, error) {
	port := NewRealSerialPort()
	err := port.Open(path)
	if err != nil {
		// 回退到 Mock
		mockPort := NewMockSerialPort()
		mockErr := mockPort.Open(path)
		if mockErr != nil {
			return nil, err
		}
		return mockPort, nil
	}
	return port, nil
}
