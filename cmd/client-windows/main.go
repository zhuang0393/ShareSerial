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
	"shareserial/internal/localproxy"
)

var (
	version = "1.0.0"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	serverAddr := flag.String("server", "", "服务器地址（覆盖配置文件）")
	localPort := flag.Int("local-port", 8888, "本地 TCP 端口")
	maxRetry := flag.Int("max-retry", 0, "最大重连次数（覆盖配置文件）")
	retryInterval := flag.Int("retry-interval", 0, "重连间隔秒数（覆盖配置文件）")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	if *showVersion {
		fmt.Printf("shareserial-client-windows v%s\n", version)
		return
	}

	// 加载配置文件
	cfg, err := config.LoadClient(*configPath)
	if err != nil {
		log.Printf("Warning: failed to load config: %v, using defaults", err)
		cfg = config.DefaultClientConfig()
	}

	// 命令行参数覆盖配置文件
	if *serverAddr != "" {
		cfg.Server.Address = *serverAddr
	}
	if *maxRetry != 0 {
		cfg.Reconnect.MaxRetry = *maxRetry
	}
	if *retryInterval != 0 {
		cfg.Reconnect.Interval = *retryInterval
	}

	log.Printf("ShareSerial Windows Client v%s", version)
	log.Printf("Server address: %s", cfg.ServerAddress())
	log.Printf("Local TCP port: %d", *localPort)
	log.Printf("Reconnect: enabled=%v, interval=%ds, max_retry=%d",
		cfg.Reconnect.Enabled, cfg.Reconnect.Interval, cfg.Reconnect.MaxRetry)

	// 创建本地代理
	proxy := localproxy.NewLocalProxy(*localPort)

	// 创建连接管理器
	connMgr := NewConnectionManager(cfg.ServerAddress(), cfg.Reconnect.MaxRetry, time.Duration(cfg.Reconnect.Interval)*time.Second)

	// 启动连接管理
	connMgr.Start()

	// 等待连接成功
	if err := connMgr.WaitForConnection(10 * time.Second); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	// 启动本地代理
	conn := connMgr.GetConnection()
	if err := proxy.Start(conn); err != nil {
		connMgr.Stop()
		log.Fatalf("Failed to start local proxy: %v", err)
	}

	log.Printf("Local TCP proxy started: localhost:%d", *localPort)
	log.Printf("")
	log.Printf("=== Connect with Putty ===")
	log.Printf("  Connection type: Raw")
	log.Printf("  Host Name: localhost")
	log.Printf("  Port: %d", *localPort)
	log.Printf("")
	log.Printf("=== Or use Python ===")
	log.Printf("  import socket")
	log.Printf("  s = socket.socket()")
	log.Printf("  s.connect(('localhost', %d))", *localPort)
	log.Printf("")

	// 数据转发监控
	go monitorLoop(connMgr, proxy)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	<-sigChan
	log.Println("Shutting down...")

	// 停止服务
	proxy.Stop()
	connMgr.Stop()

	log.Println("Client stopped")
}

// ConnectionManager 连接管理器（简化版重连）
type ConnectionManager struct {
	mu            sync.Mutex
	serverAddr    string
	conn          net.Conn
	running       bool
	maxRetry      int
	retryInterval time.Duration
	stopChan      chan struct{}
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager(serverAddr string, maxRetry int, retryInterval time.Duration) *ConnectionManager {
	return &ConnectionManager{
		serverAddr:    serverAddr,
		maxRetry:      maxRetry,
		retryInterval: retryInterval,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动连接管理
func (cm *ConnectionManager) Start() {
	cm.mu.Lock()
	cm.running = true
	cm.mu.Unlock()

	// 初始连接
	cm.connect()

	// 后台重连
	go cm.reconnectLoop()
}

// connect 尝试连接
func (cm *ConnectionManager) connect() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.conn != nil {
		return nil
	}

	conn, err := net.Dial("tcp", cm.serverAddr)
	if err != nil {
		return err
	}

	cm.conn = conn
	return nil
}

// reconnectLoop 重连循环
func (cm *ConnectionManager) reconnectLoop() {
	retryCount := 0

	for {
		select {
		case <-cm.stopChan:
			return
		default:
			cm.mu.Lock()
			conn := cm.conn
			running := cm.running
			cm.mu.Unlock()

			if !running {
				return
			}

			// 检查连接状态
			if conn == nil {
				// 尝试重连
				retryCount++
				if cm.maxRetry > 0 && retryCount > cm.maxRetry {
					log.Printf("Max retry count reached: %d", cm.maxRetry)
					return
				}

				log.Printf("Reconnecting... (attempt %d)", retryCount)
				if err := cm.connect(); err != nil {
					log.Printf("Reconnect failed: %v", err)
				} else {
					log.Printf("Reconnected to server: %s", cm.serverAddr)
					retryCount = 0
				}
			}

			time.Sleep(cm.retryInterval)
		}
	}
}

// WaitForConnection 等待连接成功
func (cm *ConnectionManager) WaitForConnection(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		cm.mu.Lock()
		conn := cm.conn
		cm.mu.Unlock()

		if conn != nil {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for connection")
}

// GetConnection 获取当前连接
func (cm *ConnectionManager) GetConnection() net.Conn {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.conn
}

// Stop 停止连接管理
func (cm *ConnectionManager) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.running = false
	close(cm.stopChan)

	if cm.conn != nil {
		cm.conn.Close()
		cm.conn = nil
	}
}

// monitorLoop 监控连接状态并更新代理
func monitorLoop(connMgr *ConnectionManager, proxy *localproxy.LocalProxy) {
	lastConn := connMgr.GetConnection()

	for {
		time.Sleep(500 * time.Millisecond)

		if !proxy.IsRunning() {
			return
		}

		currentConn := connMgr.GetConnection()

		// 检查连接是否变化
		if currentConn != lastConn && currentConn != nil {
			proxy.UpdateRemoteConnection(currentConn)
			lastConn = currentConn
			log.Printf("Proxy updated with new connection")
		}

		// 输出状态
		connCount := proxy.ConnectionCount()
		if connCount > 0 {
			log.Printf("Status: %d local connections, buffer: %d bytes",
				connCount, proxy.BufferSize())
		}
	}
}
