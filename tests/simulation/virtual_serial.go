package simulation

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// VirtualSerialPair 使用 socat 创建虚拟串口对
// 用于仿真物理串口设备
type VirtualSerialPair struct {
	mu           sync.Mutex
	physicalPort string // 模拟物理串口（Server 连接）
	terminalPort string // 模拟终端串口（用于数据注入）
	socatProcess *exec.Cmd
	socatPID     int
	physicalFile *os.File
	terminalFile *os.File
	ready        bool
}

// VirtualSerialConfig 虚拟串口配置
type VirtualSerialConfig struct {
	PhysicalPortPath string // 物理串口路径（默认 /tmp/ttyVPhysical）
	TerminalPortPath string // 终端串口路径（默认 /tmp/ttyVTerminal）
	BaudRate         int    // 波特率（模拟，实际不影响）
}

// DefaultVirtualSerialConfig 返回默认配置
func DefaultVirtualSerialConfig() *VirtualSerialConfig {
	return &VirtualSerialConfig{
		PhysicalPortPath: "/tmp/ttyVPhysical",
		TerminalPortPath: "/tmp/ttyVTerminal",
		BaudRate:         115200,
	}
}

// CreateVirtualSerialPair 创建虚拟串口对
func CreateVirtualSerialPair() (*VirtualSerialPair, error) {
	return CreateVirtualSerialPairWithConfig(DefaultVirtualSerialConfig())
}

// CreateVirtualSerialPairWithConfig 使用配置创建虚拟串口对
func CreateVirtualSerialPairWithConfig(config *VirtualSerialConfig) (*VirtualSerialPair, error) {
	vsp := &VirtualSerialPair{
		physicalPort: config.PhysicalPortPath,
		terminalPort: config.TerminalPortPath,
	}

	// 检查 socat 是否可用
	if _, err := exec.LookPath("socat"); err != nil {
		return nil, fmt.Errorf("socat not found: %v", err)
	}

	// 清理旧文件
	os.Remove(vsp.physicalPort)
	os.Remove(vsp.terminalPort)

	// 启动 socat 创建 PTY 对
	// socat pty,link=PORT1,raw,echo=0 pty,link=PORT2,raw,echo=0
	cmd := exec.Command("socat",
		"-d", "-d",
		"pty,link="+vsp.physicalPort+",raw,echo=0",
		"pty,link="+vsp.terminalPort+",raw,echo=0")

	// 设置进程组，便于清理
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// 启动 socat
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start socat: %v", err)
	}

	vsp.socatProcess = cmd
	vsp.socatPID = cmd.Process.Pid

	// 等待设备文件创建
	if err := vsp.waitForDevices(); err != nil {
		vsp.Close()
		return nil, err
	}

	// 打开端口文件
	physicalFile, err := os.OpenFile(vsp.physicalPort, os.O_RDWR, 0)
	if err != nil {
		vsp.Close()
		return nil, fmt.Errorf("failed to open physical port: %v", err)
	}

	terminalFile, err := os.OpenFile(vsp.terminalPort, os.O_RDWR, 0)
	if err != nil {
		physicalFile.Close()
		vsp.Close()
		return nil, fmt.Errorf("failed to open terminal port: %v", err)
	}

	vsp.physicalFile = physicalFile
	vsp.terminalFile = terminalFile
	vsp.ready = true

	return vsp, nil
}

// waitForDevices 等待设备文件创建
func (vsp *VirtualSerialPair) waitForDevices() error {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for virtual serial ports")
		case <-ticker.C:
			if vsp.checkDevicesReady() {
				return nil
			}
		}
	}
}

// checkDevicesReady 检查设备文件是否就绪
func (vsp *VirtualSerialPair) checkDevicesReady() bool {
	info1, err1 := os.Stat(vsp.physicalPort)
	info2, err2 := os.Stat(vsp.terminalPort)

	if err1 != nil || err2 != nil {
		return false
	}

	// 检查是否是字符设备（PTY）
	if info1.Mode()&os.ModeCharDevice == 0 || info2.Mode()&os.ModeCharDevice == 0 {
		return false
	}

	return true
}

// PhysicalPort 返回物理串口路径
func (vsp *VirtualSerialPair) PhysicalPort() string {
	return vsp.physicalPort
}

// TerminalPort 返回终端串口路径
func (vsp *VirtualSerialPair) TerminalPort() string {
	return vsp.terminalPort
}

// WriteToPhysical 向物理串口写入数据（模拟开发板发送）
// Server 会读取这些数据并广播
func (vsp *VirtualSerialPair) WriteToPhysical(data []byte) (int, error) {
	vsp.mu.Lock()
	defer vsp.mu.Unlock()

	if !vsp.ready || vsp.physicalFile == nil {
		return 0, fmt.Errorf("virtual serial pair not ready")
	}

	return vsp.physicalFile.Write(data)
}

// WriteToTerminal 向终端串口写入数据（模拟用户发送命令）
// 这些数据会被 Server 接收并写入物理串口
func (vsp *VirtualSerialPair) WriteToTerminal(data []byte) (int, error) {
	vsp.mu.Lock()
	defer vsp.mu.Unlock()

	if !vsp.ready || vsp.terminalFile == nil {
		return 0, fmt.Errorf("virtual serial pair not ready")
	}

	return vsp.terminalFile.Write(data)
}

// ReadFromPhysical 从物理串口读取数据
func (vsp *VirtualSerialPair) ReadFromPhysical(buf []byte) (int, error) {
	vsp.mu.Lock()
	defer vsp.mu.Unlock()

	if !vsp.ready || vsp.physicalFile == nil {
		return 0, fmt.Errorf("virtual serial pair not ready")
	}

	return vsp.physicalFile.Read(buf)
}

// ReadFromTerminal 从终端串口读取数据
func (vsp *VirtualSerialPair) ReadFromTerminal(buf []byte) (int, error) {
	vsp.mu.Lock()
	defer vsp.mu.Unlock()

	if !vsp.ready || vsp.terminalFile == nil {
		return 0, fmt.Errorf("virtual serial pair not ready")
	}

	return vsp.terminalFile.Read(buf)
}

// InjectData 注入数据（模拟开发板输出日志）
// 这是测试的主要方法，向物理串口写入数据，Server 会读取并广播
func (vsp *VirtualSerialPair) InjectData(data string) error {
	_, err := vsp.WriteToPhysical([]byte(data))
	return err
}

// InjectLogLine 注入一行日志（自动添加换行符）
func (vsp *VirtualSerialPair) InjectLogLine(level, message string) error {
	timestamp := time.Now().Format("15:04:05.000")
	line := fmt.Sprintf("[%s] %s: %s\n", timestamp, level, message)
	return vsp.InjectData(line)
}

// InjectMultipleLines 注入多行日志
func (vsp *VirtualSerialPair) InjectMultipleLines(lines []string) error {
	for _, line := range lines {
		if err := vsp.InjectData(line + "\n"); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond) // 模拟真实输出间隔
	}
	return nil
}

// IsReady 检查是否就绪
func (vsp *VirtualSerialPair) IsReady() bool {
	vsp.mu.Lock()
	defer vsp.mu.Unlock()
	return vsp.ready
}

// Close 关闭虚拟串口对
func (vsp *VirtualSerialPair) Close() error {
	vsp.mu.Lock()
	defer vsp.mu.Unlock()

	vsp.ready = false

	// 关闭文件
	if vsp.physicalFile != nil {
		vsp.physicalFile.Close()
		vsp.physicalFile = nil
	}
	if vsp.terminalFile != nil {
		vsp.terminalFile.Close()
		vsp.terminalFile = nil
	}

	// 终止 socat 进程
	if vsp.socatProcess != nil && vsp.socatProcess.Process != nil {
		// 发送 SIGTERM 到进程组
		syscall.Kill(-vsp.socatPID, syscall.SIGTERM)
		vsp.socatProcess.Wait()
		vsp.socatProcess = nil
	}

	// 清理文件
	os.Remove(vsp.physicalPort)
	os.Remove(vsp.terminalPort)

	return nil
}

// GetPID 返回 socat 进程 PID
func (vsp *VirtualSerialPair) GetPID() int {
	return vsp.socatPID
}
