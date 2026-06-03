//go:build linux
// +build linux

package pty

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

// termios 结构体用于配置 PTY
type termios struct {
	iflag  uint32
	oflag  uint32
	cflag  uint32
	lflag  uint32
	cc     [20]uint8
}

// TCSETS 和 TCGETS 常量
const (
	TCGETS = 0x5401
	TCSETS = 0x5402

	// 输入模式标志
	ICANON = 0x2   // 行缓冲模式
	ECHO   = 0x8   // 回显
	ICRNL  = 0x100 // CR 转 NL

	// 输出模式标志
	OPOST  = 0x1   // 输出处理
	ONLCR  = 0x4   // NL 转 CR-NL

	// 控制模式标志
	CREAD  = 0x80  // 启用接收
	CS8    = 0x30  // 8 位数据

	// 本地模式标志
	ISIG   = 0x1   // 信号处理
)

// RealPTYDevice 真实 PTY 实现
type RealPTYDevice struct {
	mu          sync.Mutex
	master      *os.File
	slave       *os.File
	slavePath   string
	symlinkPath string
	open        bool
	termios     *TermiosConfig
}

// CreateRealPTY 创建真实 PTY
func CreateRealPTY(symlinkPath string) (*RealPTYDevice, error) {
	// 使用 syscall 创建 PTY
	masterFd, err := syscall.Open("/dev/ptmx", syscall.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, err
	}

	masterFile := os.NewFile(uintptr(masterFd), "pty-master")

	// unlockpt - 解锁 slave（传入 0 表示解锁）
	var unlock int32 = 0
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL,
		uintptr(masterFd), uintptr(syscall.TIOCSPTLCK),
		uintptr(unsafe.Pointer(&unlock)), 0, 0, 0)
	if errno != 0 {
		masterFile.Close()
		return nil, fmt.Errorf("unlockpt failed: %d", errno)
	}

	// 获取 slave ID
	var slaveId int32
	_, _, errno = syscall.Syscall6(syscall.SYS_IOCTL,
		uintptr(masterFd), uintptr(syscall.TIOCGPTN),
		uintptr(unsafe.Pointer(&slaveId)), 0, 0, 0)
	if errno != 0 {
		masterFile.Close()
		return nil, fmt.Errorf("get slave id failed: %d", errno)
	}

	slavePath := fmt.Sprintf("/dev/pts/%d", slaveId)

	// 打开 slave
	slaveFd, err := syscall.Open(slavePath, syscall.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		masterFile.Close()
		return nil, fmt.Errorf("open slave failed: %v", err)
	}

	slaveFile := os.NewFile(uintptr(slaveFd), "pty-slave")

	// 设置 termios 为 raw 模式（禁用行缓冲和输出处理）
	var t termios
	_, _, errno = syscall.Syscall6(syscall.SYS_IOCTL,
		uintptr(slaveFd), uintptr(TCGETS),
		uintptr(unsafe.Pointer(&t)), 0, 0, 0)
	if errno == 0 {
		// 禁用行缓冲模式、回显、信号处理
		t.lflag &= uint32(^uint32(ICANON | ECHO | ISIG))
		// 禁用输出处理（NL 转 CR-NL）
		t.oflag &= uint32(^uint32(OPOST))
		// 禁用输入处理（CR 转 NL）
		t.iflag &= uint32(^uint32(ICRNL))
		// 设置 8 位数据
		t.cflag |= CREAD | CS8
		// 应用设置
		syscall.Syscall6(syscall.SYS_IOCTL,
			uintptr(slaveFd), uintptr(TCSETS),
			uintptr(unsafe.Pointer(&t)), 0, 0, 0)
	}

	// 创建 symlink
	if symlinkPath != "" {
		os.Remove(symlinkPath)
		err = os.Symlink(slavePath, symlinkPath)
		if err != nil {
			masterFile.Close()
			slaveFile.Close()
			return nil, err
		}
	}

	// 设置默认 termios
	config := &TermiosConfig{
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   ParityNone,
	}

	return &RealPTYDevice{
		master:      masterFile,
		slave:       slaveFile,
		slavePath:   slavePath,
		symlinkPath: symlinkPath,
		open:        true,
		termios:     config,
	}, nil
}

// SymlinkPath 返回 symlink 路径
func (r *RealPTYDevice) SymlinkPath() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.symlinkPath
}

// Read 从 master 读取数据
func (r *RealPTYDevice) Read(buf []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.open || r.master == nil {
		return 0, ErrPTYNotOpen
	}

	return r.master.Read(buf)
}

// Write 写入数据到 master
func (r *RealPTYDevice) Write(buf []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.open || r.master == nil {
		return 0, ErrPTYNotOpen
	}

	return r.master.Write(buf)
}

// Close 关闭 PTY
func (r *RealPTYDevice) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.master != nil {
		r.master.Close()
	}
	if r.slave != nil {
		r.slave.Close()
	}
	if r.symlinkPath != "" {
		os.Remove(r.symlinkPath)
	}
	r.open = false
	return nil
}

// IsOpen 检查是否打开
func (r *RealPTYDevice) IsOpen() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.open
}

// SetTermios 设置 termios 配置
func (r *RealPTYDevice) SetTermios(config *TermiosConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.termios = config
	return nil
}

// GetTermios 获取 termios 配置
func (r *RealPTYDevice) GetTermios() *TermiosConfig {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.termios
}

// InjectExternalData 注入外部数据
func (r *RealPTYDevice) InjectExternalData(data []byte) {
	// 真实 PTY 通过 slave 写入
	if r.slave != nil {
		_, _ = r.slave.Write(data)
	}
}

// GetSlaveData 获取 slave 数据
func (r *RealPTYDevice) GetSlaveData() []byte {
	return nil
}

// ReadFromSlave 从 slave 读取
func (r *RealPTYDevice) ReadFromSlave(buf []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.open || r.slave == nil {
		return 0, ErrPTYNotOpen
	}

	return r.slave.Read(buf)
}

// CreatePTY 创建 PTY（自动选择真实或 Mock）
func CreatePTY(symlinkPath string) (PTYDevice, error) {
	// 尝试真实 PTY
	realPTY, err := CreateRealPTY(symlinkPath)
	if err != nil {
		// 回退到 Mock
		return CreateMockPTY(symlinkPath)
	}
	return realPTY, nil
}
