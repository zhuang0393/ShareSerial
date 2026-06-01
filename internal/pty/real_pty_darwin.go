//go:build darwin
// +build darwin

package pty

import (
	"fmt"
	"os"
	"sync"
)

// macOS PTY 常量（与 Linux 不同）
const (
// macOS 使用 TIOCPTYGNAME 获取 slave 名称
// 这里使用简化方法：直接打开 /dev/ptmx 并读取 slave 名称
)

// RealPTYDevice 真实 PTY 实现（macOS 版本）
// macOS 使用相同的 POSIX PTY API (/dev/ptmx)
type RealPTYDevice struct {
	mu          sync.Mutex
	master      *os.File
	slave       *os.File
	slavePath   string
	symlinkPath string
	open        bool
	termios     *TermiosConfig
}

// CreateRealPTY 创建真实 PTY（macOS 版本）
func CreateRealPTY(symlinkPath string) (*RealPTYDevice, error) {
	// macOS 使用 openpty 或直接打开 /dev/ptmx
	// 简化实现：使用 os.Open 打开 /dev/ptmx

	masterFile, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	// 在 macOS 上，获取 slave 名称需要调用 ptsname
	// 由于 ptsname 在 syscall 中不可用，我们使用替代方法
	// 通过 ioctl 获取 slave 名称（简化：假设 /dev/ttyp0）

	// 简化方法：尝试查找可用的 tty 设备
	slavePath := "/dev/ttyp0"

	// 尝试打开 slave
	slaveFile, err := os.OpenFile(slavePath, os.O_RDWR, 0)
	if err != nil {
		// 尝试其他路径
		for i := 0; i < 10; i++ {
			slavePath = fmt.Sprintf("/dev/ttyp%d", i)
			slaveFile, err = os.OpenFile(slavePath, os.O_RDWR, 0)
			if err == nil {
				break
			}
		}
		if err != nil {
			masterFile.Close()
			return nil, fmt.Errorf("cannot find available slave: %v", err)
		}
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
		r.slave.Write(data)
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
