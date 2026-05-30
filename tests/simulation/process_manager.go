package simulation

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// 项目根目录
var projectRoot string

func init() {
	// 查找项目根目录（向上查找包含 go.mod 的目录）
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			projectRoot = dir
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// 到达根目录，使用当前目录
			projectRoot = dir
			break
		}
		dir = parent
	}
}

// getBinaryPath 获取二进制文件路径
func getBinaryPath(name string) string {
	return filepath.Join(projectRoot, "bin", name)
}

// ProcessManager 管理 Server 和 Client 进程
type ProcessManager struct {
	mu            sync.Mutex
	serverCmd     *exec.Cmd
	clientCmd     *exec.Cmd
	serverRunning bool
	clientRunning bool
	serverPort    int
	serverAddr    string
	serverStdout  io.Reader
	serverStderr  io.Reader
	clientStdout  io.Reader
	clientStderr  io.Reader
}

// ProcessConfig 进程配置
type ProcessConfig struct {
	ServerBinaryPath string
	ClientBinaryPath string
	SerialPort       string
	ServerPort       int
	ClientPTYPath    string
	ServerConfigPath string
	ClientConfigPath string
}

// DefaultProcessConfig 返回默认配置
func DefaultProcessConfig() *ProcessConfig {
	return &ProcessConfig{
		ServerBinaryPath: "./bin/shareserial-server",
		ClientBinaryPath: "./bin/shareserial-client",
		SerialPort:       "/tmp/ttyVPhysical",
		ServerPort:       7702, // 使用非默认端口避免冲突
		ClientPTYPath:    "/tmp/vttyTest0",
	}
}

// NewProcessManager 创建进程管理器
func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		serverPort: 7702,
	}
}

// NewProcessManagerWithConfig 使用配置创建进程管理器
func NewProcessManagerWithConfig(config *ProcessConfig) *ProcessManager {
	return &ProcessManager{
		serverPort: config.ServerPort,
	}
}

// StartServer 启动 Server 进程
func (pm *ProcessManager) StartServer(serialPort string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.serverRunning {
		return fmt.Errorf("server already running")
	}

	// 检查二进制文件
	binaryPath := getBinaryPath("shareserial-server")
	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("server binary not found at %s: %v", binaryPath, err)
	}

	// 创建命令
	pm.serverCmd = exec.Command(binaryPath,
		"--serial", serialPort,
		"--port", fmt.Sprintf("%d", pm.serverPort))

	// 设置进程组
	pm.serverCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// 获取 stdout/stderr 用于监控
	pm.serverStdout, _ = pm.serverCmd.StdoutPipe()
	pm.serverStderr, _ = pm.serverCmd.StderrPipe()

	// 启动进程
	if err := pm.serverCmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	pm.serverRunning = true
	pm.serverAddr = fmt.Sprintf("127.0.0.1:%d", pm.serverPort)

	// 等待 Server 就绪
	if err := pm.waitForServerReady(); err != nil {
		pm.StopServer()
		return err
	}

	return nil
}

// waitForServerReady 等待 Server 就绪
func (pm *ProcessManager) waitForServerReady() error {
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for server to be ready")
		case <-ticker.C:
			// 尝试连接 Server
			conn, err := net.Dial("tcp", pm.serverAddr)
			if err == nil {
				conn.Close()
				return nil
			}
		}
	}
}

// StartClient 启动 Client 进程
func (pm *ProcessManager) StartClient() error {
	return pm.StartClientWithPTY("/tmp/vttyTest0")
}

// StartClientWithPTY 启动 Client 进程（指定 PTY 路径）
func (pm *ProcessManager) StartClientWithPTY(ptyPath string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.clientRunning {
		return fmt.Errorf("client already running")
	}

	// 检查二进制文件
	binaryPath := getBinaryPath("shareserial-client")
	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("client binary not found at %s: %v", binaryPath, err)
	}

	// 创建命令
	pm.clientCmd = exec.Command(binaryPath,
		"--server", pm.serverAddr,
		"--pty", ptyPath)

	// 设置进程组
	pm.clientCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// 获取 stdout/stderr
	pm.clientStdout, _ = pm.clientCmd.StdoutPipe()
	pm.clientStderr, _ = pm.clientCmd.StderrPipe()

	// 启动进程
	if err := pm.clientCmd.Start(); err != nil {
		return fmt.Errorf("failed to start client: %v", err)
	}

	pm.clientRunning = true

	// 等待 PTY 创建
	time.Sleep(500 * time.Millisecond)

	return nil
}

// StartMultipleClients 启动多个 Client
func (pm *ProcessManager) StartMultipleClients(count int) ([]string, error) {
	ptyPaths := make([]string, count)

	for i := 0; i < count; i++ {
		ptyPath := fmt.Sprintf("/tmp/vttyTest%d", i)
		ptyPaths[i] = ptyPath

		if i == 0 {
			// 第一个 Client 使用主进程
			if err := pm.StartClientWithPTY(ptyPath); err != nil {
				return nil, err
			}
		} else {
			// 后续 Client 使用单独进程
			if err := pm.startAdditionalClient(ptyPath); err != nil {
				return nil, err
			}
		}
	}

	return ptyPaths, nil
}

// startAdditionalClient 启动额外的 Client
func (pm *ProcessManager) startAdditionalClient(ptyPath string) error {
	binaryPath := getBinaryPath("shareserial-client")

	cmd := exec.Command(binaryPath,
		"--server", pm.serverAddr,
		"--pty", ptyPath)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start additional client: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	return nil
}

// StopServer 停止 Server 进程
func (pm *ProcessManager) StopServer() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.serverRunning || pm.serverCmd == nil {
		return nil
	}

	// 发送 SIGTERM
	if pm.serverCmd.Process != nil {
		syscall.Kill(-pm.serverCmd.Process.Pid, syscall.SIGTERM)
		pm.serverCmd.Wait()
	}

	pm.serverRunning = false
	pm.serverCmd = nil

	return nil
}

// StopClient 停止 Client 进程
func (pm *ProcessManager) StopClient() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.clientRunning || pm.clientCmd == nil {
		return nil
	}

	// 发送 SIGTERM
	if pm.clientCmd.Process != nil {
		syscall.Kill(-pm.clientCmd.Process.Pid, syscall.SIGTERM)
		pm.clientCmd.Wait()
	}

	pm.clientRunning = false
	pm.clientCmd = nil

	return nil
}

// Cleanup 清理所有进程
func (pm *ProcessManager) Cleanup() error {
	pm.StopClient()
	pm.StopServer()

	// 清理 PTY 文件
	for i := 0; i < 10; i++ {
		ptyPath := fmt.Sprintf("/tmp/vttyTest%d", i)
		os.Remove(ptyPath)
	}

	return nil
}

// IsServerRunning 检查 Server 是否运行
func (pm *ProcessManager) IsServerRunning() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.serverRunning
}

// IsClientRunning 检查 Client 是否运行
func (pm *ProcessManager) IsClientRunning() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.clientRunning
}

// ServerPID 返回 Server 进程 PID
func (pm *ProcessManager) ServerPID() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.serverCmd == nil || pm.serverCmd.Process == nil {
		return 0
	}

	return pm.serverCmd.Process.Pid
}

// ClientPID 返回 Client 进程 PID
func (pm *ProcessManager) ClientPID() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.clientCmd == nil || pm.clientCmd.Process == nil {
		return 0
	}

	return pm.clientCmd.Process.Pid
}

// ServerAddr 返回 Server 地址
func (pm *ProcessManager) ServerAddr() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.serverAddr
}

// ReadFromPTY 从虚拟串口读取数据
func (pm *ProcessManager) ReadFromPTY(ptyPath string, timeout time.Duration) ([]byte, error) {
	// 打开 PTY 文件（使用 NONBLOCK 避免阻塞）
	file, err := os.OpenFile(ptyPath, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open PTY: %v", err)
	}
	defer file.Close()

	// 设置超时读取
	buf := make([]byte, 4096)
	resultChan := make(chan struct {
		data []byte
		err  error
	})

	go func() {
		// 等待一小段时间让数据到达
		time.Sleep(100 * time.Millisecond)
		n, err := file.Read(buf)
		resultChan <- struct {
			data []byte
			err  error
		}{buf[:n], err}
	}()

	select {
	case result := <-resultChan:
		return result.data, result.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout reading from PTY")
	}
}

// WriteToPTY 向虚拟串口写入数据（模拟用户发送命令）
func (pm *ProcessManager) WriteToPTY(ptyPath string, data []byte) (int, error) {
	file, err := os.OpenFile(ptyPath, os.O_WRONLY, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to open PTY for writing: %v", err)
	}
	defer file.Close()

	return file.Write(data)
}