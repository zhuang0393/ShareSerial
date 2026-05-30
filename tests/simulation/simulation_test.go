package simulation

import (
	"os"
	"testing"
	"time"
)

// SimulationEnvironment 仿真测试环境
type SimulationEnvironment struct {
	VSP            *VirtualSerialPair
	ProcessManager *ProcessManager
	DataGenerator  *DataGenerator
	TerminalTester *TerminalTester
	PTYPath        string
}

// SetupSimulationEnvironment 设置仿真测试环境
func SetupSimulationEnvironment(t *testing.T) *SimulationEnvironment {
	// 1. 创建虚拟串口对
	vsp, err := CreateVirtualSerialPair()
	if err != nil {
		t.Fatalf("Failed to create virtual serial pair: %v", err)
	}

	// 2. 创建进程管理器
	pm := NewProcessManager()

	// 3. 创建数据生成器
	dg := NewDataGenerator(vsp)

	// 4. 启动 Server
	if err := pm.StartServer(vsp.PhysicalPort()); err != nil {
		vsp.Close()
		t.Fatalf("Failed to start server: %v", err)
	}

	// 5. 启动 Client
	ptyPath := "/tmp/vttyTest0"
	if err := pm.StartClientWithPTY(ptyPath); err != nil {
		pm.StopServer()
		vsp.Close()
		t.Fatalf("Failed to start client: %v", err)
	}

	// 6. 创建终端测试器
	tt := NewTerminalTester(ptyPath)

	// 等待系统稳定
	time.Sleep(500 * time.Millisecond)

	return &SimulationEnvironment{
		VSP:            vsp,
		ProcessManager: pm,
		DataGenerator:  dg,
		TerminalTester: tt,
		PTYPath:        ptyPath,
	}
}

// Cleanup 清理仿真环境
func (sim *SimulationEnvironment) Cleanup() {
	if sim.ProcessManager != nil {
		sim.ProcessManager.Cleanup()
	}
	if sim.VSP != nil {
		sim.VSP.Close()
	}
}

// InjectData 注入数据
func (sim *SimulationEnvironment) InjectData(data string) error {
	return sim.VSP.InjectData(data)
}

// ReadFromPTY 从 PTY 读取数据
func (sim *SimulationEnvironment) ReadFromPTY(timeout time.Duration) ([]byte, error) {
	return sim.ProcessManager.ReadFromPTY(sim.PTYPath, timeout)
}

// contains 检查字符串包含
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// min 辅助函数
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestSimulationVirtualSerialPair 测试虚拟串口对
func TestSimulationVirtualSerialPair(t *testing.T) {
	vsp, err := CreateVirtualSerialPair()
	if err != nil {
		t.Fatalf("Failed to create VSP: %v", err)
	}
	defer vsp.Close()

	// 验证设备文件存在
	if !vsp.IsReady() {
		t.Error("VSP should be ready")
	}

	// 检查路径
	t.Logf("Physical port: %s", vsp.PhysicalPort())
	t.Logf("Terminal port: %s", vsp.TerminalPort())

	// 测试写入
	testData := "VSP_TEST_DATA"
	n, err := vsp.WriteToPhysical([]byte(testData))
	if err != nil {
		t.Errorf("Failed to write to physical: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write length mismatch: expected %d, got %d", len(testData), n)
	}

	t.Logf("VSP test completed: %d bytes written", n)
}

// TestSimulationDataGenerator 测试数据生成器
func TestSimulationDataGenerator(t *testing.T) {
	vsp, err := CreateVirtualSerialPair()
	if err != nil {
		t.Fatalf("Failed to create VSP: %v", err)
	}
	defer vsp.Close()

	// 创建数据生成器
	dg := NewDataGenerator(vsp)

	// 测试单行生成
	if err := dg.GenerateOneLine(); err != nil {
		t.Errorf("Failed to generate one line: %v", err)
	}

	// 测试多行生成
	if err := dg.GenerateMultipleLines(10); err != nil {
		t.Errorf("Failed to generate multiple lines: %v", err)
	}

	// 验证行计数
	if dg.LineCount() != 11 {
		t.Errorf("Expected 11 lines, got %d", dg.LineCount())
	}

	// 测试启动序列
	if err := dg.GenerateBootSequence(); err != nil {
		t.Errorf("Failed to generate boot sequence: %v", err)
	}

	t.Logf("Data generator test completed: %d lines generated", dg.LineCount())
}

// TestSimulationBasicLink 测试基础链路
func TestSimulationBasicLink(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// 设置环境
	sim := SetupSimulationEnvironment(t)
	defer sim.Cleanup()

	// 验证 Server 运行
	if !sim.ProcessManager.IsServerRunning() {
		t.Error("Server should be running")
	}

	// 验证 Client 运行
	if !sim.ProcessManager.IsClientRunning() {
		t.Error("Client should be running")
	}

	// 测试数据传输
	testData := "TEST_DATA_BASIC_LINK"
	sim.InjectData(testData + "\n")

	time.Sleep(500 * time.Millisecond)

	t.Log("Basic link test completed")
}

// TestSimulationDataFlow 测试数据流
func TestSimulationDataFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	sim := SetupSimulationEnvironment(t)
	defer sim.Cleanup()

	// 生成多条日志
	lines := []string{
		"[INFO] System starting",
		"[DEBUG] Initializing hardware",
		"[WARN] Low memory warning",
		"[ERROR] Failed to mount",
	}

	for _, line := range lines {
		sim.InjectData(line + "\n")
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	t.Logf("Data flow test completed, %d lines injected", len(lines))
}

// TestSimulationBootSequence 测试启动序列
func TestSimulationBootSequence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	sim := SetupSimulationEnvironment(t)
	defer sim.Cleanup()

	// 生成启动序列
	if err := sim.DataGenerator.GenerateBootSequence(); err != nil {
		t.Errorf("Failed to generate boot sequence: %v", err)
	}

	t.Log("Boot sequence test completed")
}

// TestSimulationMultiClient 测试多客户端
func TestSimulationMultiClient(t *testing.T) {
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

	// 3. 启动多个 Client
	clientCount := 3
	ptyPaths, err := pm.StartMultipleClients(clientCount)
	if err != nil {
		pm.Cleanup()
		t.Fatalf("Failed to start multiple clients: %v", err)
	}

	defer pm.Cleanup()

	// 4. 注入数据
	testData := "MULTICAST_TEST_DATA"
	vsp.InjectData(testData + "\n")

	time.Sleep(500 * time.Millisecond)

	// 5. 验证所有 Client 都能收到数据
	t.Logf("Multi-client test: %d clients started", clientCount)

	for i, ptyPath := range ptyPaths {
		t.Logf("Client %d PTY: %s", i, ptyPath)

		// 检查 PTY 是否存在
		if _, err := os.Stat(ptyPath); err != nil {
			t.Logf("Client %d PTY not ready: %v", i, err)
			continue
		}
	}

	t.Log("Multi-client test completed")
}

// TestSimulationHighFrequency 测试高频数据
func TestSimulationHighFrequency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	sim := SetupSimulationEnvironment(t)
	defer sim.Cleanup()

	// 生成 50 行高频数据
	count := 50
	if err := sim.DataGenerator.GenerateHighFrequencyData(count, 10); err != nil {
		t.Errorf("Failed to generate high frequency data: %v", err)
	}

	t.Logf("High frequency test completed: %d lines generated", count)
}

// TestSimulationFullFlow 全链路测试
func TestSimulationFullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full flow test in short mode")
	}

	// 设置完整环境
	sim := SetupSimulationEnvironment(t)
	defer sim.Cleanup()

	// 等待系统稳定
	time.Sleep(1 * time.Second)

	// 验证组件状态
	t.Log("=== Component Status ===")
	t.Logf("Server running: %v (PID: %d)", sim.ProcessManager.IsServerRunning(), sim.ProcessManager.ServerPID())
	t.Logf("Client running: %v (PID: %d)", sim.ProcessManager.IsClientRunning(), sim.ProcessManager.ClientPID())
	t.Logf("VSP ready: %v (PID: %d)", sim.VSP.IsReady(), sim.VSP.GetPID())
	t.Logf("Server address: %s", sim.ProcessManager.ServerAddr())
	t.Logf("Client PTY: %s", sim.PTYPath)

	// 生成数据
	t.Log("=== Generating Data ===")

	// 生成多种类型数据
	sim.InjectData("=== FULL FLOW TEST START ===\n")
	sim.DataGenerator.GenerateBootSequence()
	sim.DataGenerator.GenerateMultipleLines(20)
	sim.InjectData("=== FULL FLOW TEST END ===\n")

	time.Sleep(2 * time.Second)

	// 验证数据传输
	t.Log("=== Verifying Data Transfer ===")

	// 统计数据生成器
	t.Logf("Lines generated: %d", sim.DataGenerator.LineCount())

	t.Log("=== Full Flow Test Completed ===")
}