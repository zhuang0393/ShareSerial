package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"shareserial/internal/cli"
)

var (
	version = "1.0.0"
)

func main() {
	serverAddr := flag.String("server", "127.0.0.1:7700", "服务器地址")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	if *showVersion {
		fmt.Printf("shareserial v%s\n", version)
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"help"}
	}

	// 创建 CLI 实例
	c := cli.NewMockCLI()

	// 如果需要连接服务器，建立连接获取实时数据
	if args[0] == "log" || args[0] == "status" || args[0] == "send" {
		conn, err := net.DialTimeout("tcp", *serverAddr, 5*time.Second)
		if err != nil {
			log.Fatalf("Failed to connect to server: %v", err)
		}
		defer conn.Close()

		// 根据命令发送请求
		switch args[0] {
		case "log":
			// 发送请求获取日志
			if _, err := conn.Write([]byte("GET_LOG\n")); err != nil {
				log.Printf("Failed to send log request: %v", err)
			}
			// 读取响应（简化实现）
			buf := make([]byte, 4096)
			n, _ := conn.Read(buf)
			lines := parseLogData(string(buf[:n]))
			c.SetLogData(lines)
		case "status":
			// 发送请求获取状态
			if _, err := conn.Write([]byte("GET_STATUS\n")); err != nil {
				log.Printf("Failed to send status request: %v", err)
			}
			buf := make([]byte, 1024)
			n, _ := conn.Read(buf)
			status := parseStatusData(string(buf[:n]))
			c.SetStatus(status)
		case "send":
			// 发送命令
			command := ""
			for i := 0; i < len(args); i++ {
				if args[i] == "--command" && i+1 < len(args) {
					command = args[i+1]
					break
				}
			}
			if command != "" {
				if _, err := conn.Write([]byte("SEND:" + command + "\n")); err != nil {
					log.Printf("Failed to send command: %v", err)
				}
				buf := make([]byte, 1024)
				n, _ := conn.Read(buf)
				c.SetSendResponse(string(buf[:n]))
			}
		}
	}

	// 执行命令
	result, err := c.Execute(args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result)
}

// parseLogData 解析日志数据
func parseLogData(data string) []string {
	lines := []string{}
	for _, line := range splitLines(data) {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// parseStatusData 解析状态数据
func parseStatusData(data string) map[string]interface{} {
	status := map[string]interface{}{
		"connected": true,
		"server":    data,
	}
	return status
}

// splitLines 分割行
func splitLines(data string) []string {
	result := []string{}
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			result = append(result, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		result = append(result, data[start:])
	}
	return result
}
