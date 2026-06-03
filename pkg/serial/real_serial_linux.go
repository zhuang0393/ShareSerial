//go:build linux
// +build linux

package serial

import (
	"go.bug.st/serial"
)

// RealSerialPort 真实串口实现
// 注意：移除互斥锁，因为 Read 和 Write 应该能够并发执行
// 否则 Read 阻塞时会阻止 Write 执行，导致用户输入无法发送
type RealSerialPort struct {
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
	if r.port != nil {
		r.port.Close()
	}
	r.open = false
	return nil
}

// Read 读取数据（无锁，允许并发）
// go.bug.st/serial 的 Read 是线程安全的
func (r *RealSerialPort) Read(buf []byte) (int, error) {
	if !r.open || r.port == nil {
		return 0, ErrPortNotOpen
	}
	return r.port.Read(buf)
}

// Write 写入数据（无锁，允许并发）
// go.bug.st/serial 的 Write 是线程安全的
func (r *RealSerialPort) Write(buf []byte) (int, error) {
	if !r.open || r.port == nil {
		return 0, ErrPortNotOpen
	}
	return r.port.Write(buf)
}

// IsOpen 检查是否打开
func (r *RealSerialPort) IsOpen() bool {
	return r.open
}

// SetConfig 设置配置
func (r *RealSerialPort) SetConfig(config *Config) error {
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
