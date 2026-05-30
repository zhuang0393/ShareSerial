package main

import (
	"flag"
	"fmt"
	"io"
	"log"
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
		cfg.Server.Address = *serverAddr
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

	// 数据转发 goroutine：服务器 -> PTY
	go func() {
		for {
			conn := reconn.GetConnection()
			if conn == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from server: %v", err)
				}
				reconn.Disconnect()
				continue
			}
			device.Write(buf[:n])
		}
	}()

	// PTY -> 服务器转发
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := device.Read(buf)
			if err != nil {
				// Read 正常阻塞，不会频繁返回错误
				time.Sleep(10 * time.Millisecond)
				continue
			}

			conn := reconn.GetConnection()
			if conn != nil && reconn.IsConnected() {
				conn.Write(buf[:n])
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

	// 关闭连接
	reconn.Stop()
	reconn.Disconnect()
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