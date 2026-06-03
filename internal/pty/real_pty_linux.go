//go:build linux
// +build linux

package pty

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
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
	// 使用 unix.Open 创建 PTY master
	masterFd, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, err
	}

	masterFile := os.NewFile(uintptr(masterFd), "pty-master")

	// unlockpt - 解锁 slave（传入 0 表示解锁）
	var unlock int32 = 0
	_, _, errno := unix.Syscall6(unix.SYS_IOCTL,
		uintptr(masterFd), uintptr(unix.TIOCSPTLCK),
		uintptr(unsafe.Pointer(&unlock)), 0, 0, 0)
	if errno != 0 {
		masterFile.Close()
		return nil, fmt.Errorf("unlockpt failed: %d", errno)
	}

	// 获取 slave ID
	var slaveId int32
	_, _, errno = unix.Syscall6(unix.SYS_IOCTL,
		uintptr(masterFd), uintptr(unix.TIOCGPTN),
		uintptr(unsafe.Pointer(&slaveId)), 0, 0, 0)
	if errno != 0 {
		masterFile.Close()
		return nil, fmt.Errorf("get slave id failed: %d", errno)
	}

	slavePath := fmt.Sprintf("/dev/pts/%d", slaveId)

	// 打开 slave
	slaveFd, err := unix.Open(slavePath, unix.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		masterFile.Close()
		return nil, fmt.Errorf("open slave failed: %v", err)
	}

	slaveFile := os.NewFile(uintptr(slaveFd), "pty-slave")

	// 设置 slave 为 raw 模式（关键！）
	// 使用 IoctlGetTermios/IoctlSetTermios 设置 termios
	termios, err := unix.IoctlGetTermios(slaveFd, unix.TCGETS)
	if err == nil {
		// 设置 raw 模式（参考 cfmakeraw）
		termios.Iflag &= uint32(^uint32(unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON))
		termios.Oflag &= uint32(^uint32(unix.OPOST))
		termios.Lflag &= uint32(^uint32(unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN))
		termios.Cflag &= uint32(^uint32(unix.CSIZE | unix.PARENB))
		termios.Cflag |= unix.CS8
		termios.Cc[unix.VMIN] = 1
		termios.Cc[unix.VTIME] = 0
		// 应用设置
		unix.IoctlSetTermios(slaveFd, unix.TCSETS, termios)
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

	// 设置默认 termios 配置记录
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

// Read 从 master 读取数据（阻塞调用）
func (r *RealPTYDevice) Read(buf []byte) (int, error) {
	r.mu.Lock()
	if !r.open || r.master == nil {
		r.mu.Unlock()
		return 0, ErrPTYNotOpen
	}
	master := r.master
	r.mu.Unlock()

	return master.Read(buf)
}

// Write 写入数据到 master
func (r *RealPTYDevice) Write(buf []byte) (int, error) {
	r.mu.Lock()
	if !r.open || r.master == nil {
		r.mu.Unlock()
		return 0, ErrPTYNotOpen
	}
	master := r.master
	r.mu.Unlock()

	return master.Write(buf)
}

// Close 关闭 PTY（强制解除 Read 阻塞）
func (r *RealPTYDevice) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.open {
		return nil
	}

	r.open = false

	// 先关闭文件，强制解除任何阻塞的 Read
	if r.master != nil {
		r.master.Close()
	}
	if r.slave != nil {
		r.slave.Close()
	}
	if r.symlinkPath != "" {
		os.Remove(r.symlinkPath)
	}
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
	r.mu.Lock()
	slave := r.slave
	r.mu.Unlock()

	if slave != nil {
		_, _ = slave.Write(data)
	}
}

// GetSlaveData 获取 slave 数据
func (r *RealPTYDevice) GetSlaveData() []byte {
	return nil
}

// ReadFromSlave 从 slave 读取
func (r *RealPTYDevice) ReadFromSlave(buf []byte) (int, error) {
	r.mu.Lock()
	if !r.open || r.slave == nil {
		r.mu.Unlock()
		return 0, ErrPTYNotOpen
	}
	slave := r.slave
	r.mu.Unlock()

	return slave.Read(buf)
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