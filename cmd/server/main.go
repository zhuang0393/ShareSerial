package main

import (
	"flag"
	"fmt"
	"log"
	"net"
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
		cfg = config.DefaultServerConfig()
	}

	// 命令行参数覆盖配置文件
	if *serialPath != "" {
		cfg.Serial.Path = *serialPath
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}

	// 创建服务器
	srv := server.NewTCPServer()

	// 尝试打开真实串口
	serialPort, err := serial.NewPort(cfg.Serial.Path)
	if err != nil {
		log.Printf("[WARN] 串口 %s 未找到，使用模拟串口", cfg.Serial.Path)
		serialPort = serial.NewMockSerialPort()
		_ = serialPort.Open(cfg.Serial.Path)
	}
	srv.SetSerial(serialPort)

	// 启动服务器
	addr := cfg.ListenAddress()
	if err := srv.Start(addr); err != nil {
		log.Fatalf("[FAIL] 无法启动服务器: %v", err)
	}

	// 简洁的启动信息
	localIP := getLocalIP()
	log.Printf("========================================")
	log.Printf("ShareSerial Server 已就绪")
	log.Printf("========================================")
	log.Printf("监听: %s:%d", localIP, cfg.Server.Port)
	log.Printf("串口: %s @ %d baud", cfg.Serial.Path, cfg.Serial.BaudRate)
	log.Printf("========================================")
	log.Printf("")
	log.Printf("Windows Client 连接命令:")
	log.Printf("  shareserial-client-windows.exe --server %s:%d", localIP, cfg.Server.Port)
	log.Printf("")

	// 停止信号
	stopChan := make(chan struct{})

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 从串口读取数据并广播
	go readSerialAndBroadcast(srv, serialPort, stopChan)

	// 等待信号
	<-sigChan
	log.Println("[INFO] 正在关闭服务器...")

	// 先停止数据读取
	close(stopChan)

	// 停止服务器
	if err := srv.Stop(); err != nil {
		log.Printf("[WARN] 停止服务器失败: %v", err)
	}
	log.Println("[OK] 服务器已停止")
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

// getLocalIP 获取本地 IP 地址
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "127.0.0.1"
}