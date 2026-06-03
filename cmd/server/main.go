package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
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
	serialPath := flag.String("serial", "", "串口路径（覆盖配置文件）")
	port := flag.Int("port", 0, "监听端口（覆盖配置文件，0 表示使用配置文件值）")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	if *showVersion {
		fmt.Printf("shareserial-server v%s\n", version)
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
		_ = serialPort.Open(cfg.Serial.Path)
	}
	srv.SetSerial(serialPort)

	// 启动服务器
	addr := cfg.ListenAddress()
	if err := srv.Start(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Printf("ShareSerial Server started on %s", srv.Addr())
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
	if err := srv.Stop(); err != nil {
		log.Printf("Failed to stop server: %v", err)
	}
	log.Println("Server stopped")
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
	// 检查默认路径
	if _, err := os.Stat("/etc/shareserial/server.yaml"); err == nil {
		return "/etc/shareserial/server.yaml"
	}
	if _, err := os.Stat("./configs/server.yaml"); err == nil {
		return "./configs/server.yaml"
	}
	return "defaults"
}
