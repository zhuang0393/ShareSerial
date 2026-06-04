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
		cfg = config.DefaultClientConfig()
	}

	// 命令行参数覆盖配置文件
	if *serverAddr != "" {
		// 解析 server 地址（可能包含端口）
		host, port, err := net.SplitHostPort(*serverAddr)
		if err == nil {
			// 地址包含端口，分别设置
			cfg.Server.Address = host
			// 解析端口字符串为整数
			portInt := 0
			for _, c := range port {
				if c >= '0' && c <= '9' {
					portInt = portInt*10 + int(c-'0')
				}
			}
			cfg.Server.Port = portInt
		} else {
			// 地址不含端口，只设置 IP
			cfg.Server.Address = *serverAddr
		}
	}
	if *maxRetry != 0 {
		cfg.Reconnect.MaxRetry = *maxRetry
	}
	if *retryInterval != 0 {
		cfg.Reconnect.Interval = *retryInterval
	}

	// 创建本地代理
	proxy := localproxy.NewLocalProxy(*localPort)

	// 创建连接管理器
	connMgr := NewConnectionManager(cfg.ServerAddress(), cfg.Reconnect.MaxRetry, time.Duration(cfg.Reconnect.Interval)*time.Second)

	// 启动连接管理
	connMgr.Start()

	// 等待连接成功
	if err := connMgr.WaitForConnection(10 * time.Second); err != nil {
		log.Fatalf("[FAIL] 无法连接服务器: %v", err)
	}

	// 启动本地代理
	conn := connMgr.GetConnection()
	if err := proxy.Start(conn); err != nil {
		connMgr.Stop()
		log.Fatalf("[FAIL] 无法启动本地代理: %v", err)
	}

	// 简洁的启动信息
	log.Printf("========================================")
	log.Printf("ShareSerial Client 已就绪")
	log.Printf("========================================")
	log.Printf("服务器: %s", cfg.ServerAddress())
	log.Printf("本地端口: %d", *localPort)
	log.Printf("========================================")
	log.Printf("")
	log.Printf("MobaXterm 连接设置:")
	log.Printf("  类型: Raw (不是 SSH)")
	log.Printf("  主机: localhost")
	log.Printf("  端口: %d", *localPort)
	log.Printf("")

	// 数据转发监控
	go monitorLoop(connMgr, proxy)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	<-sigChan
	log.Println("[INFO] 正在关闭...")

	// 停止服务
	proxy.Stop()
	connMgr.Stop()

	log.Println("[OK] Client 已停止")
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
	if err := cm.connect(); err != nil {
		log.Printf("[WARN] 初次连接失败: %v", err)
	}

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
					log.Printf("[FAIL] 达到最大重试次数: %d", cm.maxRetry)
					return
				}

				log.Printf("[INFO] 重连中... (第 %d 次)", retryCount)
				if err := cm.connect(); err != nil {
					log.Printf("[WARN] 重连失败: %v", err)
				} else {
					log.Printf("[OK] 已重连: %s", cm.serverAddr)
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

	return fmt.Errorf("连接超时")
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
			log.Printf("[INFO] 代理已更新")
		}
	}
}