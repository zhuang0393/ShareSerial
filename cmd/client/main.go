package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"shareserial/internal/config"
	"shareserial/internal/pty"
	"shareserial/internal/reconnect"
)

var (
	version = "1.0.0"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	serverAddr := flag.String("server", "", "服务器地址（覆盖配置文件）")
	ptyPath := flag.String("pty", "", "虚拟串口路径（覆盖配置文件）")
	maxRetry := flag.Int("max-retry", 0, "最大重连次数（覆盖配置文件，0 表示使用配置文件值）")
	retryInterval := flag.Int("retry-interval", 0, "重连间隔秒数（覆盖配置文件，0 表示使用配置文件值）")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	if *showVersion {
		fmt.Printf("shareserial-client v%s\n", version)
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
		// 解析 server 地址（可能包含端口）
		// 如果地址包含端口（如 127.0.0.1:7700），分离 IP 和端口
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
	if *ptyPath != "" {
		cfg.PTY.Path = *ptyPath
	}
	if *maxRetry != 0 {
		cfg.Reconnect.MaxRetry = *maxRetry
	}
	if *retryInterval != 0 {
		cfg.Reconnect.Interval = *retryInterval
	}

	log.Printf("Configuration loaded from: %s", getConfigSource(*configPath))
	log.Printf("Server address: %s", cfg.ServerAddress())
	log.Printf("PTY path: %s", cfg.PTY.Path)
	log.Printf("Reconnect: enabled=%v, interval=%ds, max_retry=%d",
		cfg.Reconnect.Enabled, cfg.Reconnect.Interval, cfg.Reconnect.MaxRetry)

	// 创建虚拟串口
	device, err := pty.CreatePTY(cfg.PTY.Path)
	if err != nil {
		log.Fatalf("Failed to create PTY: %v", err)
	}
	log.Printf("Virtual serial port created: %s", device.SymlinkPath())

	// 创建重连管理器
	serverAddress := cfg.ServerAddress()
	reconn := reconnect.NewReconnectManager(
		serverAddress,
		cfg.Reconnect.MaxRetry,
		time.Duration(cfg.Reconnect.Interval)*time.Second,
	)

	// 连接服务器
	err = reconn.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	log.Printf("Connected to server: %s", serverAddress)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动自动重连（后台）
	reconnErrChan := reconn.AutoReconnect()

	// 创建停止信号 channel，用于通知所有 goroutine 停止
	stopChan := make(chan struct{})

	// 数据转发 goroutine：服务器 -> PTY
	serverToPtyDone := make(chan struct{})
	go func() {
		defer close(serverToPtyDone)
		errorCount := 0 // 限制错误日志打印次数
		for {
			select {
			case <-stopChan:
				return // 收到停止信号，退出循环
			default:
			}

			conn := reconn.GetConnection()
			if conn == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				_ = reconn.Disconnect()
				// 限制错误日志打印次数，避免疯狂打印
				if err != io.EOF && errorCount < 3 {
					log.Printf("Error reading from server: %v", err)
					errorCount++
				}
				// 连接断开后等待一下，避免频繁循环
				time.Sleep(500 * time.Millisecond)
				continue
			}
			errorCount = 0 // 成功读取后重置计数
			_, _ = device.Write(buf[:n])
		}
	}()

	// PTY -> 服务器转发（使用 goroutine 读取避免阻塞）
	ptyToServerDone := make(chan struct{})
	go func() {
		defer close(ptyToServerDone)
		buf := make([]byte, 1024)
		for {
			// 使用 goroutine 来避免 Read 阻塞无法响应 stopChan
			readDone := make(chan struct{})
			var n int
			var err error
			go func() {
				defer close(readDone)
				n, err = device.Read(buf)
			}()

			// 等待读取完成或停止信号
			select {
			case <-stopChan:
				// 强制关闭 device 来解除 Read 阻塞
				device.Close()
				return
			case <-readDone:
				// 读取完成，处理数据
				if err != nil {
					// PTY 关闭或出错，退出
					return
				}
				conn := reconn.GetConnection()
				if conn != nil && reconn.IsConnected() {
					_, _ = conn.Write(buf[:n])
				}
			}
		}
	}()

	// 等待信号或重连失败
	select {
	case <-sigChan:
		log.Println("Shutting down client...")
	case err := <-reconnErrChan:
		log.Printf("Reconnect failed: %v", err)
	}

	// 通知所有 goroutine 停止
	close(stopChan)

	// 等待 goroutine 完成（最多等待 2 秒）
	select {
	case <-serverToPtyDone:
	case <-time.After(2 * time.Second):
	}
	select {
	case <-ptyToServerDone:
	case <-time.After(2 * time.Second):
	}

	// 关闭连接
	reconn.Stop()
	_ = reconn.Disconnect()
	device.Close()
	log.Println("Client stopped")
}

// getConfigSource 返回配置来源描述
func getConfigSource(path string) string {
	if path != "" {
		return path
	}
	// 检查默认路径
	if _, err := os.Stat("/etc/shareserial/client.yaml"); err == nil {
		return "/etc/shareserial/client.yaml"
	}
	if _, err := os.Stat("./configs/client.yaml"); err == nil {
		return "./configs/client.yaml"
	}
	return "defaults"
}
