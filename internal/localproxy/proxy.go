package localproxy

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// LocalProxy 本地 TCP 代理
// 将远程 Server 的数据转发到本地 TCP 端口
// 用户可通过连接本地端口访问远程串口
type LocalProxy struct {
	mu            sync.Mutex
	remoteConn    net.Conn     // 远程 Server 连接
	localListener net.Listener // 本地监听
	localAddr     string
	localPort     int
	running       bool
	stopChan      chan struct{}
	localConns    map[net.Conn]bool // 活动的本地连接
	remoteBuffer  []byte            // 远程数据缓冲（用于新连接）
}

// NewLocalProxy 创建本地代理
func NewLocalProxy(localPort int) *LocalProxy {
	return &LocalProxy{
		localAddr:    fmt.Sprintf("127.0.0.1:%d", localPort),
		localPort:    localPort,
		stopChan:     make(chan struct{}),
		localConns:   make(map[net.Conn]bool),
		remoteBuffer: make([]byte, 0, 1024*1024), // 1MB 缓冲
	}
}

// Start 启动本地代理
func (p *LocalProxy) Start(remoteConn net.Conn) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.remoteConn = remoteConn

	// 启动本地监听
	listener, err := net.Listen("tcp", p.localAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", p.localAddr, err)
	}

	p.localListener = listener
	p.running = true

	// 接受本地连接
	go p.acceptLoop()

	// 从远程读取数据并广播到所有本地连接
	go p.remoteReadLoop()

	return nil
}

// acceptLoop 接受本地连接
func (p *LocalProxy) acceptLoop() {
	for {
		select {
		case <-p.stopChan:
			return
		default:
			localConn, err := p.localListener.Accept()
			if err != nil {
				return // 监听器已关闭
			}

			p.mu.Lock()
			p.localConns[localConn] = true
			p.mu.Unlock()

			// 发送已有的缓冲数据到新连接
			p.sendBufferedData(localConn)

			// 启动本地连接处理
			go p.handleLocalConnection(localConn)
		}
	}
}

// handleLocalConnection 处理本地连接
func (p *LocalProxy) handleLocalConnection(localConn net.Conn) {
	defer func() {
		p.mu.Lock()
		delete(p.localConns, localConn)
		p.mu.Unlock()
		localConn.Close()
	}()

	// 从本地连接读取数据并发送到远程
	buf := make([]byte, 1024)
	for {
		select {
		case <-p.stopChan:
			return
		default:
			n, err := localConn.Read(buf)
			if err != nil {
				return // 连接关闭
			}

			// 发送到远程
			p.mu.Lock()
			if p.remoteConn != nil {
				_, _ = p.remoteConn.Write(buf[:n])
			}
			p.mu.Unlock()
		}
	}
}

// remoteReadLoop 从远程读取数据并广播到所有本地连接
func (p *LocalProxy) remoteReadLoop() {
	buf := make([]byte, 4096)

	for {
		select {
		case <-p.stopChan:
			return
		default:
			p.mu.Lock()
			remoteConn := p.remoteConn
			p.mu.Unlock()

			if remoteConn == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			n, err := remoteConn.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Remote connection error: %v", err)
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if n > 0 {
				// 缓存数据（用于新连接）
				data := make([]byte, n)
				copy(data, buf[:n])

				p.mu.Lock()
				// 限制缓冲大小
				if len(p.remoteBuffer) < 1024*1024 {
					p.remoteBuffer = append(p.remoteBuffer, data...)
				}
				// 广播到所有本地连接
				for localConn := range p.localConns {
					go func(conn net.Conn, d []byte) {
						_, _ = conn.Write(d)
					}(localConn, data)
				}
				p.mu.Unlock()
			}
		}
	}
}

// sendBufferedData 发送已缓冲的数据到新连接
func (p *LocalProxy) sendBufferedData(localConn net.Conn) {
	p.mu.Lock()
	buffer := p.remoteBuffer
	p.mu.Unlock()

	if len(buffer) > 0 {
		// 发送最近的数据（最多 64KB）
		start := 0
		if len(buffer) > 64*1024 {
			start = len(buffer) - 64*1024
		}
		_, _ = localConn.Write(buffer[start:])
	}
}

// UpdateRemoteConnection 更新远程连接（重连后）
func (p *LocalProxy) UpdateRemoteConnection(newConn net.Conn) {
	p.mu.Lock()
	oldConn := p.remoteConn
	p.remoteConn = newConn
	p.mu.Unlock()

	if oldConn != nil {
		oldConn.Close()
	}
}

// Stop 停止本地代理
func (p *LocalProxy) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	p.running = false
	close(p.stopChan)

	// 关闭所有本地连接
	for localConn := range p.localConns {
		localConn.Close()
	}
	p.localConns = make(map[net.Conn]bool)

	// 关闭监听器
	if p.localListener != nil {
		p.localListener.Close()
	}

	// 关闭远程连接
	if p.remoteConn != nil {
		p.remoteConn.Close()
	}
}

// IsRunning 检查是否运行
func (p *LocalProxy) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// LocalAddr 返回本地地址
func (p *LocalProxy) LocalAddr() string {
	return p.localAddr
}

// LocalPort 返回本地端口
func (p *LocalProxy) LocalPort() int {
	return p.localPort
}

// ConnectionCount 返回本地连接数
func (p *LocalProxy) ConnectionCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.localConns)
}

// BufferSize 返回缓冲数据大小
func (p *LocalProxy) BufferSize() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.remoteBuffer)
}

// ClearBuffer 清空缓冲
func (p *LocalProxy) ClearBuffer() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.remoteBuffer = make([]byte, 0, 1024*1024)
}
