//go:build windows
// +build windows

package serial

import (
	"fmt"
	"time"
)

// ScannerWindows Windows 串口扫描器
type ScannerWindows struct {
	maxPort int // 最大扫描端口号
}

// NewScannerWindows 创建 Windows 扫描器
func NewScannerWindows() *ScannerWindows {
	return &ScannerWindows{
		maxPort: 30, // 默认扫描 COM1-COM30
	}
}

// NewScanner 创建扫描器（Windows 版本）
func NewScanner() *ScannerWindows {
	return NewScannerWindows()
}

// Scan 扫描 Windows COM 端口
// 返回可用的 COM 端口列表（COM1-COM30）
func (s *ScannerWindows) Scan() ([]string, error) {
	available := []string{}

	for i := 1; i <= s.maxPort; i++ {
		portName := fmt.Sprintf("COM%d", i)

		// 尝试打开串口判断是否可用
		port := NewRealSerialPort()

		// 设置较短超时避免阻塞
		err := port.Open(portName)
		if err == nil {
			// 端口可用，立即关闭
			port.Close()
			available = append(available, portName)
		}
	}

	return available, nil
}

// ScanExtended 扫描扩展范围（COM1-COM255）
// 用于需要检测更多串口的场景
func (s *ScannerWindows) ScanExtended() ([]string, error) {
	s.maxPort = 255
	return s.Scan()
}

// ScanWithTimeout 带超时的扫描
// 每个端口尝试时间不超过指定超时
func (s *ScannerWindows) ScanWithTimeout(timeout time.Duration) ([]string, error) {
	available := []string{}

	for i := 1; i <= s.maxPort; i++ {
		portName := fmt.Sprintf("COM%d", i)

		// 使用 goroutine 和超时控制
		done := make(chan bool, 1)
		var portErr error

		go func() {
			port := NewRealSerialPort()
			portErr = port.Open(portName)
			if portErr == nil {
				port.Close()
				done <- true
			} else {
				done <- false
			}
		}()

		select {
		case success := <-done:
			if success {
				available = append(available, portName)
			}
		case <-time.After(timeout):
			// 超时，跳过该端口
			continue
		}
	}

	return available, nil
}

// SetMaxPort 设置最大扫描端口号
func (s *ScannerWindows) SetMaxPort(max int) {
	s.maxPort = max
}

// IsPortAvailable 检查单个端口是否可用
func (s *ScannerWindows) IsPortAvailable(portName string) bool {
	port := NewRealSerialPort()
	err := port.Open(portName)
	if err != nil {
		return false
	}
	port.Close()
	return true
}

// GetCommonPorts 返回常用 COM 端口名称列表
func GetCommonPorts() []string {
	return []string{"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "COM10"}
}