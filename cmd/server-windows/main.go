package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"shareserial/internal/config"
	"shareserial/internal/server"
	"shareserial/pkg/serial"
)

var (
	version = "1.0.0"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	serialPath := flag.String("serial", "", "串口路径（如 COM1，覆盖配置文件）")
	port := flag.Int("port", 0, "监听端口（覆盖配置文件，0 表示使用配置文件值）")
	scan := flag.Bool("scan", false, "扫描可用串口")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	if *showVersion {
		fmt.Printf("shareserial-server-windows v%s\n", version)
		return
	}

	// 串口扫描
	if *scan {
		scanPorts()
		return
	}

	// 加载配置文件
	cfg, err := config.LoadServer(*configPath)
	if err != nil {
		log.Printf("Warning: failed to load config: %v, using defaults", err)
		cfg = config.DefaultServerConfig()
	}

	// 命令行参数覆盖配置文件
	if *serialPath != "" {
		cfg.Serial.Path = *serialPath
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}

	log.Printf("Configuration loaded from: %s", getConfigSource(*configPath))
	log.Printf("Serial port: %s (baudrate: %d)", cfg.Serial.Path, cfg.Serial.BaudRate)
	log.Printf("Listen address: %s", cfg.ListenAddress())

	// 创建服务器
	srv := server.NewTCPServer()

	// 尝试打开真实串口
	serialPort, err := serial.NewPort(cfg.Serial.Path)
	if err != nil {
		log.Printf("Warning: cannot open serial port %s: %v", cfg.Serial.Path, err)
		log.Printf("Using mock serial port for testing")
		serialPort = serial.NewMockSerialPort()
		serialPort.Open(cfg.Serial.Path)
	}
	srv.SetSerial(serialPort)

	// 启动服务器
	addr := cfg.ListenAddress()
	if err := srv.Start(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Printf("ShareSerial Windows Server started on %s", srv.Addr())
	log.Printf("Serial port: %s (%d baud)", cfg.Serial.Path, cfg.Serial.BaudRate)
	log.Printf("Arbiter timeout: %d seconds", cfg.Arbiter.Timeout)

	// 停止信号
	stopChan := make(chan struct{})

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 从串口读取数据并广播
	go readSerialAndBroadcast(srv, serialPort, stopChan)

	// 等待信号
	<-sigChan
	log.Println("Shutting down server...")

	// 先停止数据读取
	close(stopChan)

	// 停止服务器
	srv.Stop()
	log.Println("Server stopped")
}

// scanPorts 扫描可用串口
func scanPorts() {
	log.Println("Scanning available COM ports...")

	scanner := serial.NewScanner()
	ports, err := scanner.Scan()
	if err != nil {
		log.Printf("Scan error: %v", err)
		return
	}

	if len(ports) == 0 {
		log.Println("No COM ports found")
		fmt.Println("\nTips:")
		fmt.Println("  - Check if device is connected")
		fmt.Println("  - Install proper drivers (e.g., FTDI, CH340)")
		fmt.Println("  - Run as Administrator for permission issues")
		return
	}

	fmt.Println("\nAvailable COM ports:")
	for i, port := range ports {
		fmt.Printf("  [%d] %s\n", i+1, port)
	}
}

// readSerialAndBroadcast 从串口读取数据并广播到所有客户端
func readSerialAndBroadcast(srv *server.TCPServer, port serial.Port, stop <-chan struct{}) {
	buf := make([]byte, 4096)

	for {
		select {
		case <-stop:
			return
		default:
			// 尝试读取数据
			n, err := port.Read(buf)
			if err != nil {
				// 读取错误，等待后继续
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if n > 0 {
				// 广播到所有客户端
				data := make([]byte, n)
				copy(data, buf[:n])
				srv.Broadcast(data)
			}
		}
	}
}

// getConfigSource 返回配置来源描述
func getConfigSource(path string) string {
	if path != "" {
		return path
	}
	// Windows 默认路径
	appData := os.Getenv("APPDATA")
	if appData != "" {
		defaultPath := appData + "\\shareserial\\server.yaml"
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath
		}
	}
	// 检查当前目录
	if _, err := os.Stat("./configs/server-windows.yaml"); err == nil {
		return "./configs/server-windows.yaml"
	}
	if _, err := os.Stat("./configs/server.yaml"); err == nil {
		return "./configs/server.yaml"
	}
	return "defaults"
}

// ConnectionMonitor 连接监控器（可选功能）
type ConnectionMonitor struct {
	mu       sync.Mutex
	server   *server.TCPServer
	running  bool
	stopChan chan struct{}
}

// NewConnectionMonitor 创建连接监控器
func NewConnectionMonitor(srv *server.TCPServer) *ConnectionMonitor {
	return &ConnectionMonitor{
		server:   srv,
		stopChan: make(chan struct{}),
	}
}

// Start 启动监控
func (m *ConnectionMonitor) Start() {
	m.mu.Lock()
	m.running = true
	m.mu.Unlock()

	go m.monitorLoop()
}

// monitorLoop 监控循环
func (m *ConnectionMonitor) monitorLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.mu.Lock()
			running := m.running
			m.mu.Unlock()

			if !running {
				return
			}

			// 输出状态
			clientCount := m.server.ClientCount()
			if clientCount > 0 {
				log.Printf("Status: %d clients connected, write lock: %v",
					clientCount, m.server.HasWriteLock())
			}
		}
	}
}

// Stop 停止监控
func (m *ConnectionMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.running = false
	close(m.stopChan)
}

// isPortOpen 检查端口是否可连接
func isPortOpen(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
