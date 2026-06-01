//go:build linux
// +build linux

package serial

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

// ScannerLinux Linux 串口扫描器
type ScannerLinux struct {
	searchPaths []string // 搜索路径
}

// NewScannerLinux 创建 Linux 扫描器
func NewScannerLinux() *ScannerLinux {
	return &ScannerLinux{
		searchPaths: []string{
			"/dev/ttyUSB*",
			"/dev/ttyACM*",
			"/dev/ttyS*",
		},
	}
}

// NewScanner 创建扫描器（Linux 版本）
func NewScanner() *ScannerLinux {
	return NewScannerLinux()
}

// Scan 扫描 Linux 串口设备
// 返回可用的串口设备列表
func (s *ScannerLinux) Scan() ([]string, error) {
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

// ScanUSB 扫描 USB 串口设备
func (s *ScannerLinux) ScanUSB() ([]string, error) {
	return s.scanPattern("/dev/ttyUSB*")
}

// ScanACM 扫描 ACM 串口设备（通常是 Arduino 等）
func (s *ScannerLinux) ScanACM() ([]string, error) {
	return s.scanPattern("/dev/ttyACM*")
}

// ScanLegacy 扫描传统串口设备
func (s *ScannerLinux) ScanLegacy() ([]string, error) {
	return s.scanPattern("/dev/ttyS*")
}

// scanPattern 扫描指定模式的设备
func (s *ScannerLinux) scanPattern(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	available := []string{}
	for _, path := range matches {
		if s.isAccessible(path) {
			available = append(available, path)
		}
	}

	return available, nil
}

// isAccessible 检查设备是否可访问
func (s *ScannerLinux) isAccessible(path string) bool {
	// 检查文件是否存在
	info, err := ioutil.ReadFile(path)
	if err != nil {
		return false
	}

	// 简单检查：文件不为空（实际上串口设备文件通常为空，这里只是检查可访问）
	_ = info

	// 更准确的方法：尝试打开设备
	port := NewRealSerialPort()
	err = port.Open(path)
	if err != nil {
		return false
	}
	port.Close()
	return true
}

// SetSearchPaths 设置搜索路径
func (s *ScannerLinux) SetSearchPaths(paths []string) {
	s.searchPaths = paths
}

// AddSearchPath 添加搜索路径
func (s *ScannerLinux) AddSearchPath(pattern string) {
	s.searchPaths = append(s.searchPaths, pattern)
}

// IsPortAvailable 检查单个端口是否可用
func (s *ScannerLinux) IsPortAvailable(path string) bool {
	return s.isAccessible(path)
}

// GetDeviceType 获取设备类型
func (s *ScannerLinux) GetDeviceType(path string) string {
	if strings.HasPrefix(path, "/dev/ttyUSB") {
		return "USB Serial"
	}
	if strings.HasPrefix(path, "/dev/ttyACM") {
		return "ACM Serial (Arduino/CDC)"
	}
	if strings.HasPrefix(path, "/dev/ttyS") {
		return "Legacy Serial"
	}
	return "Unknown"
}

// ScanDetailed 扫描并返回详细信息
func (s *ScannerLinux) ScanDetailed() ([]PortInfo, error) {
	available, err := s.Scan()
	if err != nil {
		return nil, err
	}

	infoList := []PortInfo{}
	for _, path := range available {
		infoList = append(infoList, PortInfo{
			Path:    path,
			Type:    s.GetDeviceType(path),
			Vendor:  "", // 可通过 sysfs 获取
			Product: "", // 可通过 sysfs 获取
		})
	}

	return infoList, nil
}

// PortInfo 串口设备信息
type PortInfo struct {
	Path    string
	Type    string
	Vendor  string
	Product string
}

// GetCommonPorts 返回常用 Linux 串口路径列表
func GetCommonPorts() []string {
	return []string{
		"/dev/ttyUSB0",
		"/dev/ttyUSB1",
		"/dev/ttyACM0",
		"/dev/ttyS0",
	}
}
