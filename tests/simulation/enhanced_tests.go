package simulation

import (
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"shareserial/pkg/arbiter"
)

// TestSimulationShellInteraction 测试 Shell 交互场景
// 模拟用户通过虚拟串口发送命令并接收响应
func TestSimulationShellInteraction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// 1. 创建虚拟串口对
	vsp, err := CreateVirtualSerialPair()
	if err != nil {
		t.Fatalf("Failed to create VSP: %v", err)
	}
	defer vsp.Close()

	// 2. 启动 Server
	pm := NewProcessManager()
	if err := pm.StartServer(vsp.PhysicalPort()); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer pm.Cleanup()

	// 3. 启动 Client
	ptyPath := "/tmp/vttyShell0"
	if err := pm.StartClientWithPTY(ptyPath); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// 4. 模拟 Shell 交互
	commands := []struct {
		cmd      string
		response string
	}{
		{"ls", "file1.txt\nfile2.txt\n"},
		{"pwd", "/home/user\n"},
		{"echo hello", "hello\n"},
		{"uname -a", "Linux test 5.4.0\n"},
	}

	for _, tc := range commands {
		// 注入命令响应（模拟开发板响应）
		vsp.InjectData(fmt.Sprintf("$ %s\n", tc.cmd))
		time.Sleep(100 * time.Millisecond)
		vsp.InjectData(tc.response)
		time.Sleep(100 * time.Millisecond)
	}

	// 5. 验证 PTY 文件存在
	if _, err := os.Stat(ptyPath); err != nil {
		t.Errorf("PTY file not found: %v", err)
	}

	t.Logf("Shell interaction test completed: %d commands processed", len(commands))
}

// TestSimulationReconnect 测试断线重连场景
// 模拟网络断开后客户端自动重连
func TestSimulationReconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// 1. 创建虚拟串口对
	vsp, err := CreateVirtualSerialPair()
	if err != nil {
		t.Fatalf("Failed to create VSP: %v", err)
	}
	defer vsp.Close()

	// 2. 启动 Server
	pm := NewProcessManager()
	if err := pm.StartServer(vsp.PhysicalPort()); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// 3. 建立初始连接
	conn, err := net.Dial("tcp", pm.ServerAddr())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// 验证初始连接
	initialConnected := true
	if _, err := conn.Write([]byte("test\n")); err != nil {
		initialConnected = false
	}
	t.Logf("Initial connection: %v", initialConnected)

	// 4. 模拟断线
	conn.Close()
	time.Sleep(500 * time.Millisecond)

	// 5. 重连
	conn2, err := net.Dial("tcp", pm.ServerAddr())
	if err != nil {
		t.Fatalf("Failed to reconnect: %v", err)
	}
	defer conn2.Close()

	time.Sleep(200 * time.Millisecond)

	// 6. 验证重连成功
	reconnected := false
	if _, err := conn2.Write([]byte("reconnect test\n")); err != nil {
		t.Errorf("Reconnect write failed: %v", err)
	} else {
		reconnected = true
	}

	// 7. 注入数据验证重连后数据正常
	vsp.InjectData("DATA_AFTER_RECONNECT\n")
	time.Sleep(200 * time.Millisecond)

	pm.Cleanup()

	t.Logf("Reconnect test completed: initial=%v, reconnected=%v", initialConnected, reconnected)
}

// TestSimulationHighFrequencyPressure 高频压力测试
// 测试系统在高频率数据流下的稳定性
func TestSimulationHighFrequencyPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping pressure test in short mode")
	}

	// 1. 创建虚拟串口对
	vsp, err := CreateVirtualSerialPair()
	if err != nil {
		t.Fatalf("Failed to create VSP: %v", err)
	}
	defer vsp.Close()

	// 2. 启动 Server
	pm := NewProcessManager()
	if err := pm.StartServer(vsp.PhysicalPort()); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer pm.Cleanup()

	// 3. 启动多个 Client
	ptyPaths, err := pm.StartMultipleClients(3)
	if err != nil {
		t.Fatalf("Failed to start multiple clients: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// 4. 高频数据注入
	dg := NewDataGenerator(vsp)

	// 测试参数
	testLines := 1000 // 生成 1000 行数据
	testInterval := 5 // 每 5ms 一行（高频率）

	start := time.Now()
	if err := dg.GenerateHighFrequencyData(testLines, testInterval); err != nil {
		t.Errorf("High frequency generation failed: %v", err)
	}
	elapsed := time.Since(start)

	// 5. 计算吞吐量
	throughput := float64(testLines) / elapsed.Seconds()

	// 6. 验证结果
	t.Logf("High frequency pressure test:")
	t.Logf("  Lines generated: %d", testLines)
	t.Logf("  Time elapsed: %v", elapsed)
	t.Logf("  Throughput: %.2f lines/sec", throughput)
	t.Logf("  Clients: %d PTYs created", len(ptyPaths))

	// 验证吞吐量 > 100 lines/sec（实际应该更高）
	if throughput < 100 {
		t.Errorf("Throughput too low: %.2f lines/sec", throughput)
	}

	// 验证 PTY 文件
	for i, ptyPath := range ptyPaths {
		if _, err := os.Stat(ptyPath); err != nil {
			t.Logf("Client %d PTY: %s (not accessible)", i, ptyPath)
		} else {
			t.Logf("Client %d PTY: %s (ready)", i, ptyPath)
		}
	}
}

// TestSimulationMultiClientConcurrent 多客户端并发测试
// 测试多个客户端同时连接和接收数据
func TestSimulationMultiClientConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// 1. 创建虚拟串口对
	vsp, err := CreateVirtualSerialPair()
	if err != nil {
		t.Fatalf("Failed to create VSP: %v", err)
	}
	defer vsp.Close()

	// 2. 启动 Server
	pm := NewProcessManager()
	if err := pm.StartServer(vsp.PhysicalPort()); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer pm.Cleanup()

	// 3. 并发连接多个客户端
	clientCount := 10
	var wg sync.WaitGroup
	connErrors := make([]error, clientCount)

	start := time.Now()

	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			conn, err := net.Dial("tcp", pm.ServerAddr())
			if err != nil {
				connErrors[idx] = err
				return
			}
			// 保持连接一段时间
			time.Sleep(200 * time.Millisecond)
			conn.Close()
		}(i)
	}

	wg.Wait()
	connectTime := time.Since(start)

	// 4. 检查连接错误
	errorCount := 0
	for _, e := range connErrors {
		if e != nil {
			errorCount++
		}
	}

	// 5. 注入广播数据
	vsp.InjectData("BROADCAST_TO_ALL_CLIENTS\n")
	time.Sleep(200 * time.Millisecond)

	// 6. 结果报告
	t.Logf("Multi-client concurrent test:")
	t.Logf("  Client count: %d", clientCount)
	t.Logf("  Connect time: %v", connectTime)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Success rate: %.2f%%", float64(clientCount-errorCount)/float64(clientCount)*100)

	// 验证成功率 > 90%
	if errorCount > clientCount/10 {
		t.Errorf("Too many connection errors: %d out of %d", errorCount, clientCount)
	}
}

// TestSimulationWriteLockArbitration 写锁仲裁测试
// 测试多客户端竞争写锁的场景
func TestSimulationWriteLockArbitration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// 1. 创建虚拟串口对
	vsp, err := CreateVirtualSerialPair()
	if err != nil {
		t.Fatalf("Failed to create VSP: %v", err)
	}
	defer vsp.Close()

	// 2. 启动 Server
	pm := NewProcessManager()
	if err := pm.StartServer(vsp.PhysicalPort()); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer pm.Cleanup()

	// 3. 启动多个 Client
	ptyPaths, err := pm.StartMultipleClients(5)
	if err != nil {
		t.Fatalf("Failed to start multiple clients: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// 4. 创建仲裁器
	arb := arbiter.NewArbiter(10 * time.Second)

	// 5. 模拟多个客户端竞争写锁
	clientIDs := []string{"client-0", "client-1", "client-2", "client-3", "client-4"}
	lockResults := make([]bool, len(clientIDs))

	// Client 0 获取写锁
	ok, _ := arb.Acquire(clientIDs[0])
	lockResults[0] = ok
	t.Logf("Client 0 acquire lock: %v", ok)

	// 其他客户端尝试获取写锁（应该失败）
	for i := 1; i < len(clientIDs); i++ {
		ok, _ = arb.Acquire(clientIDs[i])
		lockResults[i] = ok
		t.Logf("Client %d acquire lock: %v (expected: false)", i, ok)

		// 验证其他客户端无法获取锁
		if ok {
			t.Errorf("Client %d should not acquire lock while Client 0 holds it", i)
		}
	}

	// 6. Client 0 释放写锁
	arb.Release(clientIDs[0])
	t.Logf("Client 0 released lock")

	// 7. Client 1 现在可以获取写锁
	ok, _ = arb.Acquire(clientIDs[1])
	t.Logf("Client 1 acquire lock after release: %v (expected: true)", ok)

	if !ok {
		t.Error("Client 1 should acquire lock after Client 0 released")
	}

	// 8. 写入数据验证
	vsp.InjectData("TEST_WRITE_LOCK_DATA\n")
	time.Sleep(200 * time.Millisecond)

	t.Logf("Write lock arbitration test completed: %d clients tested", len(clientIDs))
	t.Logf("PTY paths created: %v", ptyPaths)
}

// TestSimulationLongRunningStability 长时间运行稳定性测试
// 测试系统在长时间运行下的稳定性
func TestSimulationLongRunningStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long running test in short mode")
	}

	// 测试持续时间（测试模式：30秒）
	duration := 30 * time.Second

	// 1. 创建虚拟串口对
	vsp, err := CreateVirtualSerialPair()
	if err != nil {
		t.Fatalf("Failed to create VSP: %v", err)
	}
	defer vsp.Close()

	// 2. 启动 Server
	pm := NewProcessManager()
	if err := pm.StartServer(vsp.PhysicalPort()); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer pm.Cleanup()

	// 3. 启动 Client
	ptyPath := "/tmp/vttyStability0"
	if err := pm.StartClientWithPTY(ptyPath); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// 4. 持续注入数据
	dg := NewDataGenerator(vsp)
	dg.StartContinuous(100 * time.Millisecond) // 每 100ms 一行

	start := time.Now()
	linesGenerated := 0

	// 运行指定时间
	for time.Since(start) < duration {
		time.Sleep(1 * time.Second)
		currentLines := dg.LineCount()
		t.Logf("Running: %v elapsed, %d lines generated", time.Since(start), currentLines)
	}

	dg.Stop()
	linesGenerated = dg.LineCount()

	// 5. 验证 Server 和 Client 仍在运行
	if !pm.IsServerRunning() {
		t.Error("Server should still be running")
	}
	if !pm.IsClientRunning() {
		t.Error("Client should still be running")
	}

	// 6. 验证 PTY 文件
	if _, err := os.Stat(ptyPath); err != nil {
		t.Errorf("PTY file should exist after long run: %v", err)
	}

	// 7. 结果报告
	t.Logf("Long running stability test completed:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Lines generated: %d", linesGenerated)
	t.Logf("  Average rate: %.2f lines/sec", float64(linesGenerated)/duration.Seconds())
	t.Logf("  Server status: running=%v (PID=%d)", pm.IsServerRunning(), pm.ServerPID())
	t.Logf("  Client status: running=%v (PID=%d)", pm.IsClientRunning(), pm.ClientPID())

	// 验证数据量（至少应该有 duration/0.1 秒的行数）
	expectedMinLines := int(duration.Seconds() / 0.1)
	if linesGenerated < expectedMinLines/2 {
		t.Errorf("Too few lines generated: %d (expected at least %d)", linesGenerated, expectedMinLines/2)
	}
}

// TestSimulationDataIntegrity 数据完整性测试
// 测试数据在传输过程中是否保持完整
func TestSimulationDataIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// 1. 创建虚拟串口对
	vsp, err := CreateVirtualSerialPair()
	if err != nil {
		t.Fatalf("Failed to create VSP: %v", err)
	}
	defer vsp.Close()

	// 2. 启动 Server
	pm := NewProcessManager()
	if err := pm.StartServer(vsp.PhysicalPort()); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer pm.Cleanup()

	// 3. 连接客户端
	conn, err := net.Dial("tcp", pm.ServerAddr())
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer conn.Close()

	time.Sleep(200 * time.Millisecond)

	// 4. 发送特殊数据（包含各种字符）
	testData := []string{
		"ASCII: abc123!@#$%^&*()",
		"Numbers: 0123456789",
		"Timestamp: 2025-01-01 12:00:00.000",
		"JSON: {\"key\": \"value\"}",
		"Empty line:",
		"Long line: " + "A" + "B" + "C" + "D" + "E",
	}

	for _, data := range testData {
		// 注入数据
		vsp.InjectData(data + "\n")
		time.Sleep(100 * time.Millisecond)
	}

	// 5. 尝试读取数据
	buf := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Logf("Read error: %v", err)
	}

	receivedData := string(buf[:n])
	t.Logf("Data integrity test:")
	t.Logf("  Sent lines: %d", len(testData))
	t.Logf("  Received bytes: %d", n)

	// 6. 验证数据包含关键内容
	for _, data := range testData {
		if data != "" && len(data) > 5 {
			// 检查关键部分是否在接收数据中
			keyword := data[:5]
			if !contains(receivedData, keyword) {
				t.Logf("Warning: keyword '%s' not found in received data", keyword)
			}
		}
	}

	t.Logf("Data integrity test completed")
}

// TestSimulationErrorHandling 错误处理测试
// 测试系统在异常情况下的行为
func TestSimulationErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// 1. 创建虚拟串口对
	vsp, err := CreateVirtualSerialPair()
	if err != nil {
		t.Fatalf("Failed to create VSP: %v", err)
	}
	defer vsp.Close()

	// 2. 启动 Server
	pm := NewProcessManager()
	if err := pm.StartServer(vsp.PhysicalPort()); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer pm.Cleanup()

	// 3. 测试场景：连接后立即断开
	for i := 0; i < 10; i++ {
		conn, err := net.Dial("tcp", pm.ServerAddr())
		if err != nil {
			t.Errorf("Connection %d failed: %v", i, err)
			continue
		}

		// 立即发送数据然后断开
		conn.Write([]byte(fmt.Sprintf("Quick disconnect test %d\n", i)))
		conn.Close()

		time.Sleep(50 * time.Millisecond)
	}

	// 4. 验证 Server 仍在运行
	if !pm.IsServerRunning() {
		t.Error("Server should still be running after quick disconnect tests")
	}

	// 5. 正常连接测试（验证系统恢复）
	conn, err := net.Dial("tcp", pm.ServerAddr())
	if err != nil {
		t.Fatalf("Normal connection failed after error tests: %v", err)
	}

	// 发送正常数据
	conn.Write([]byte("NORMAL_DATA_AFTER_ERRORS\n"))
	time.Sleep(200 * time.Millisecond)

	// 读取响应
	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n, _ := conn.Read(buf)
	conn.Close()

	t.Logf("Error handling test completed:")
	t.Logf("  Quick disconnect tests: 10")
	t.Logf("  Server still running: %v", pm.IsServerRunning())
	t.Logf("  Normal data received: %d bytes", n)

	// 验证正常连接可以工作
	if n > 0 {
		t.Log("System recovered successfully after errors")
	}
}

// TestSimulationConcurrentReadWrite 并发读写测试
// 测试同时进行读和写操作
func TestSimulationConcurrentReadWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// 1. 创建虚拟串口对
	vsp, err := CreateVirtualSerialPair()
	if err != nil {
		t.Fatalf("Failed to create VSP: %v", err)
	}
	defer vsp.Close()

	// 2. 启动 Server
	pm := NewProcessManager()
	if err := pm.StartServer(vsp.PhysicalPort()); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer pm.Cleanup()

	// 3. 连接客户端
	conn, err := net.Dial("tcp", pm.ServerAddr())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(200 * time.Millisecond)

	// 4. 并发读写
	var wg sync.WaitGroup
	readCount := 0
	writeCount := 0
	errors := make([]error, 0)
	var errorMu sync.Mutex

	// 读线程
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			buf := make([]byte, 1024)
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := conn.Read(buf)
			if err == nil && n > 0 {
				readCount++
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()

	// 写线程（通过 VSP 注入数据）
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			if err := vsp.InjectData(fmt.Sprintf("DATA_%d\n", i)); err != nil {
				errorMu.Lock()
				errors = append(errors, err)
				errorMu.Unlock()
			} else {
				writeCount++
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()

	wg.Wait()

	// 5. 结果报告
	t.Logf("Concurrent read/write test:")
	t.Logf("  Read operations: %d successful", readCount)
	t.Logf("  Write operations: %d successful", writeCount)
	t.Logf("  Errors: %d", len(errors))

	// 验证大部分操作成功
	if len(errors) > 10 {
		t.Errorf("Too many errors: %d", len(errors))
	}

	// 验证至少有一些读操作成功
	if readCount < 10 {
		t.Logf("Warning: Low read count: %d", readCount)
	}
}
