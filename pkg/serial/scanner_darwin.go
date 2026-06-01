//go:build darwin
// +build darwin

package serial

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

// ScannerDarwin macOS 串口扫描器
type ScannerDarwin struct {
	searchPaths []string // 搜索路径
}

// NewScannerDarwin 创建 macOS 扫描器
func NewScannerDarwin() *ScannerDarwin {
	return &ScannerDarwin{
		searchPaths: []string{
			"/dev/tty.usbserial*",
			"/dev/tty.usbmodem*",
			"/dev/cu.usbserial*",
			"/dev/cu.usbmodem*",
		},
	}
}

// NewScanner 创建扫描器（macOS 版本）
func NewScanner() *ScannerDarwin {
	return NewScannerDarwin()
}

// Scan 扫描 macOS 串口设备
// 返回可用的串口设备列表
func (s *ScannerDarwin) Scan() ([]string, error) {
	available := []string{}

	for _, pattern := range s.searchPaths {
		// 使用 glob 匹配
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, path := range matches {
			// 检查文件是否存在且可访问
			if s.isAccessible(path) {
				available = append(available, path)
			}
		}
	}

	return available, nil
}

// ScanTTY 扫描 tty.* 设备（输入设备）
func (s *ScannerDarwin) ScanTTY() ([]string, error) {
	available := []string{}

	for _, pattern := range []string{
		"/dev/tty.usbserial*",
		"/dev/tty.usbmodem*",
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, path := range matches {
			if s.isAccessible(path) {
				available = append(available, path)
			}
		}
	}

	return available, nil
}

// ScanCU 扫描 cu.* 设备（输出设备）
func (s *ScannerDarwin) ScanCU() ([]string, error) {
	available := []string{}

	for _, pattern := range []string{
		"/dev/cu.usbserial*",
		"/dev/cu.usbmodem*",
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, path := range matches {
			if s.isAccessible(path) {
				available = append(available, path)
			}
		}
	}

	return available, nil
}

// isAccessible 检查设备是否可访问
func (s *ScannerDarwin) isAccessible(path string) bool {
	// 检查文件是否存在
	_, err := ioutil.ReadFile(path)
	if err != nil {
		return false
	}

	// 尝试打开设备
	port := NewRealSerialPort()
	err = port.Open(path)
	if err != nil {
		return false
	}
	port.Close()
	return true
}

// SetSearchPaths 设置搜索路径
func (s *ScannerDarwin) SetSearchPaths(paths []string) {
	s.searchPaths = paths
}

// AddSearchPath 添加搜索路径
func (s *ScannerDarwin) AddSearchPath(pattern string) {
	s.searchPaths = append(s.searchPaths, pattern)
}

// IsPortAvailable 检查单个端口是否可用
func (s *ScannerDarwin) IsPortAvailable(path string) bool {
	return s.isAccessible(path)
}

// GetDeviceType 获取设备类型
func (s *ScannerDarwin) GetDeviceType(path string) string {
	if strings.HasPrefix(path, "/dev/tty.usbserial") || strings.HasPrefix(path, "/dev/cu.usbserial") {
		return "USB Serial (FTDI/PL2303)"
	}
	if strings.HasPrefix(path, "/dev/tty.usbmodem") || strings.HasPrefix(path, "/dev/cu.usbmodem") {
		return "USB Modem (CDC-ACM)"
	}
	return "Unknown"
}

// GetCommonPorts 返回常用 macOS 串口路径列表
func GetCommonPorts() []string {
	return []string{
		"/dev/tty.usbserial",
		"/dev/tty.usbmodem",
		"/dev/cu.usbserial",
		"/dev/cu.usbmodem",
	}
}