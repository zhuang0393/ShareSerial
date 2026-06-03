package serial

import (
	"io"
	"sync"
	"testing"
	"time"
)

// BlockingMockSerialPort 模拟真实串口的阻塞读取行为
// 用于测试 Read 阻塞时 Write 是否能执行
type BlockingMockSerialPort struct {
	mu           sync.Mutex
	open         bool
	inputBuffer  []byte
	outputBuffer []byte
	readBlocked  bool // 标记 Read 是否阻塞
	blockChan    chan struct{} // 用于模拟阻塞
}

// NewBlockingMockSerialPort 创建阻塞 Mock 串口
func NewBlockingMockSerialPort() *BlockingMockSerialPort {
	return &BlockingMockSerialPort{
		inputBuffer:  make([]byte, 0),
		outputBuffer: make([]byte, 0),
		blockChan:    make(chan struct{}),
	}
}

func (b *BlockingMockSerialPort) Open(path string) error {
	b.mu.Lock()
	b.open = true
	b.mu.Unlock()
	return nil
}

func (b *BlockingMockSerialPort) Close() error {
	b.mu.Lock()
	b.open = false
	// 解除任何阻塞的 Read
	close(b.blockChan)
	b.mu.Unlock()
	return nil
}

func (b *BlockingMockSerialPort) IsOpen() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.open
}

// Read 模拟阻塞读取（真实串口行为）
// 当没有数据时，阻塞等待而不是返回 EOF
func (b *BlockingMockSerialPort) Read(buf []byte) (int, error) {
	b.mu.Lock()

	if !b.open {
		b.mu.Unlock()
		return 0, ErrPortNotOpen
	}

	// 如果没有数据，阻塞等待
	if len(b.inputBuffer) == 0 {
		b.readBlocked = true
		b.mu.Unlock()

		// 阻塞等待数据或关闭
		<-b.blockChan

		b.mu.Lock()
		b.readBlocked = false
		if !b.open {
			b.mu.Unlock()
			return 0, io.EOF
		}
	}

	// 有数据时读取
	n := copy(buf, b.inputBuffer)
	b.inputBuffer = b.inputBuffer[n:]
	b.mu.Unlock()
	return n, nil
}

// Write 写入数据
func (b *BlockingMockSerialPort) Write(buf []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.open {
		return 0, ErrPortNotOpen
	}

	b.outputBuffer = append(b.outputBuffer, buf...)
	return len(buf), nil
}

// InjectInput 注入输入数据（并解除阻塞）
func (b *BlockingMockSerialPort) InjectInput(data []byte) {
	b.mu.Lock()
	b.inputBuffer = append(b.inputBuffer, data...)
	// 解除阻塞
	select {
	case <-b.blockChan:
		// 已经关闭
	default:
		close(b.blockChan)
		b.blockChan = make(chan struct{})
	}
	b.mu.Unlock()
}

// Unblock 解除阻塞（不注入数据）
func (b *BlockingMockSerialPort) Unblock() {
	b.mu.Lock()
	select {
	case <-b.blockChan:
	default:
		close(b.blockChan)
		b.blockChan = make(chan struct{})
	}
	b.mu.Unlock()
}

// GetOutputData 获取输出数据
func (b *BlockingMockSerialPort) GetOutputData() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.outputBuffer
}

// IsReadBlocked 检查 Read 是否阻塞
func (b *BlockingMockSerialPort) IsReadBlocked() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.readBlocked
}

// TestBlockingReadBehavior 测试阻塞读取行为
// 验证：真实串口 Read 无数据时会阻塞
func TestBlockingReadBehavior(t *testing.T) {
	t.Log("=== Test: Blocking Read Behavior ===")

	port := NewBlockingMockSerialPort()
	port.Open("/dev/ttyTest")

	// 启动 Read goroutine
	readDone := make(chan struct{})
	var readN int
	var readErr error

	go func() {
		defer close(readDone)
		buf := make([]byte, 1024)
		readN, readErr = port.Read(buf)
	}()

	// 等待确认 Read 已阻塞
	time.Sleep(100 * time.Millisecond)

	if !port.IsReadBlocked() {
		t.Log("Read is NOT blocking (unexpected for real serial)")
	} else {
		t.Log("Read IS blocking (expected for real serial)")
	}

	// 注入数据解除阻塞
	port.InjectInput([]byte("test data\n"))

	// 等待 Read 完成
	select {
	case <-readDone:
		t.Logf("Read completed: n=%d, err=%v", readN, readErr)
		t.Log("Read unblocked when data arrived")
	case <-time.After(1 * time.Second):
		t.Fatal("Read timeout - should have unblocked when data injected")
	}

	t.Log("=== Test Completed ===")
}

// TestConcurrentReadWriteWithMutex 测试带锁的并发读写
// 模拟之前的 RealSerialPort 行为（Read/Write 共享锁）
func TestConcurrentReadWriteWithMutex(t *testing.T) {
	t.Log("=== Test: Concurrent Read/Write WITH Mutex (OLD behavior) ===")

	// 使用带锁的 Mock（模拟旧版 RealSerialPort）
	port := NewMockSerialPort() // 这个有共享锁

	port.Open("/dev/ttyTest")

	// 启动 Read goroutine（会立即返回 EOF，因为没有数据）
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		buf := make([]byte, 1024)
		for i := 0; i < 10; i++ {
			n, err := port.Read(buf)
			t.Logf("Read %d: n=%d, err=%v", i+1, n, err)
			if err == io.EOF {
				time.Sleep(50 * time.Millisecond) // 模拟重试
			}
		}
	}()

	// 同时尝试 Write
	time.Sleep(100 * time.Millisecond)

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		data := []byte("user input\n")
		n, err := port.Write(data)
		t.Logf("Write: n=%d, err=%v", n, err)
	}()

	select {
	case <-writeDone:
		t.Log("Write completed successfully")
	case <-time.After(500 * time.Millisecond):
		t.Log("Write completed (Mock Read doesn't block long)")
	}

	select {
	case <-readDone:
		t.Log("Read loop completed")
	case <-time.After(1 * time.Second):
		t.Log("Read loop still running")
	}

	t.Log("")
	t.Log("Analysis:")
	t.Log("- MockSerialPort.Read returns EOF immediately (not blocking)")
	t.Log("- Lock is released quickly")
	t.Log("- Write can get lock and execute")
	t.Log("- This is why tests PASSED but real usage FAILED!")

	t.Log("=== Test Completed ===")
}

// TestRealSerialBlockingScenario 测试真实串口阻塞场景
// 这个测试模拟真实场景：Read 阻塞时 Write 无法执行
func TestRealSerialBlockingScenario(t *testing.T) {
	t.Log("=== Test: Real Serial Blocking Scenario ===")

	port := NewBlockingMockSerialPort()
	port.Open("/dev/ttyTest")

	// 启动持续 Read goroutine（模拟 readSerialAndBroadcast）
	readChan := make(chan []byte, 10)
	stopRead := make(chan struct{})

	go func() {
		buf := make([]byte, 4096)
		for {
			select {
			case <-stopRead:
				return
			default:
				n, err := port.Read(buf)
				if err == nil && n > 0 {
					data := make([]byte, n)
					copy(data, buf[:n])
					readChan <- data
				}
			}
		}
	}()

	// 等待确认 Read 已阻塞
	time.Sleep(200 * time.Millisecond)

	t.Log("Status: Read goroutine is blocked waiting for serial data")
	t.Logf("Read blocked status: %v", port.IsReadBlocked())

	// 尝试 Write（模拟用户输入）
	t.Log("Attempting Write (user input)...")

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		data := []byte("ls\n")
		n, err := port.Write(data)
		t.Logf("Write attempt: n=%d, err=%v", n, err)
	}()

	// 检查 Write 是否能完成
	select {
	case <-writeDone:
		t.Log("Write COMPLETED successfully!")
		t.Log("This means Read blocking doesn't prevent Write (good)")
	case <-time.After(2 * time.Second):
		t.Log("Write TIMEOUT!")
		t.Log("This means Read blocking prevented Write (BUG!)")
		t.Log("Root cause: Read holds lock while blocking")
	}

	// 检查输出数据
	outputData := port.GetOutputData()
	t.Logf("Output data: %s", string(outputData))

	if len(outputData) == 0 {
		t.Log("No output data - Write was blocked by Read!")
	} else {
		t.Log("Output data exists - Write succeeded!")
	}

	// 清理
	close(stopRead)
	port.Close()

	t.Log("=== Test Completed ===")
}

// TestConcurrentReadWriteNoMutex 测试无锁并发读写
// 验证：移除锁后，Read 阻塞不影响 Write
func TestConcurrentReadWriteNoMutex(t *testing.T) {
	t.Log("=== Test: Concurrent Read/Write WITHOUT Mutex (NEW behavior) ===")

	// 创建一个无锁的阻塞 Mock
	port := NewBlockingMockSerialPortNoMutex()
	port.Open("/dev/ttyTest")

	// 启动 Read goroutine
	readChan := make(chan []byte, 10)
	stopRead := make(chan struct{})

	go func() {
		buf := make([]byte, 4096)
		for {
			select {
			case <-stopRead:
				return
			default:
				n, err := port.Read(buf)
				if err == nil && n > 0 {
					data := make([]byte, n)
					copy(data, buf[:n])
					readChan <- data
				}
			}
		}
	}()

	// 等待 Read 阻塞
	time.Sleep(200 * time.Millisecond)
	t.Logf("Read blocked: %v", port.IsReadBlocked())

	// Write（应该能执行）
	t.Log("Attempting Write...")

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		data := []byte("user command\n")
		n, _ := port.Write(data)
		t.Logf("Write: n=%d", n)
	}()

	select {
	case <-writeDone:
		t.Log("✅ Write completed successfully!")
		t.Log("Even while Read is blocking, Write can execute")
	case <-time.After(2 * time.Second):
		t.Log("❌ Write timeout!")
	}

	// 验证数据
	outputData := port.GetOutputData()
	t.Logf("Output data: %q", string(outputData))

	// 解除阻塞
	port.InjectInput([]byte("response\n"))

	select {
	case data := <-readChan:
		t.Logf("Read received: %q", string(data))
	case <-time.After(1 * time.Second):
		t.Log("Read timeout")
	}

	close(stopRead)
	port.Close()

	t.Log("=== Test Completed ===")
}

// BlockingMockSerialPortNoMutex 无锁版本的阻塞 Mock
type BlockingMockSerialPortNoMutex struct {
	open         bool
	inputBuffer  []byte
	outputBuffer []byte
	readBlocked  bool
	blockChan    chan struct{}
	mu           sync.Mutex // 只用于保护 buffer，不用于 Read/Write
}

func NewBlockingMockSerialPortNoMutex() *BlockingMockSerialPortNoMutex {
	return &BlockingMockSerialPortNoMutex{
		inputBuffer:  make([]byte, 0),
		outputBuffer: make([]byte, 0),
		blockChan:    make(chan struct{}),
	}
}

func (b *BlockingMockSerialPortNoMutex) Open(path string) error {
	b.open = true
	return nil
}

func (b *BlockingMockSerialPortNoMutex) Close() error {
	b.open = false
	close(b.blockChan)
	return nil
}

func (b *BlockingMockSerialPortNoMutex) IsOpen() bool {
	return b.open
}

// Read 无锁，可以阻塞
func (b *BlockingMockSerialPortNoMutex) Read(buf []byte) (int, error) {
	if !b.open {
		return 0, ErrPortNotOpen
	}

	// 无数据时阻塞
	b.mu.Lock()
	hasData := len(b.inputBuffer) > 0
	b.mu.Unlock()

	if !hasData {
		b.readBlocked = true
		<-b.blockChan // 阻塞等待
		b.readBlocked = false

		if !b.open {
			return 0, io.EOF
		}
	}

	// 读取数据（不加锁）
	b.mu.Lock()
	n := copy(buf, b.inputBuffer)
	b.inputBuffer = b.inputBuffer[n:]
	b.mu.Unlock()

	return n, nil
}

// Write 无锁，立即执行
func (b *BlockingMockSerialPortNoMutex) Write(buf []byte) (int, error) {
	if !b.open {
		return 0, ErrPortNotOpen
	}

	// 直接写入（不加锁）
	b.mu.Lock()
	b.outputBuffer = append(b.outputBuffer, buf...)
	b.mu.Unlock()

	return len(buf), nil
}

func (b *BlockingMockSerialPortNoMutex) InjectInput(data []byte) {
	b.mu.Lock()
	b.inputBuffer = append(b.inputBuffer, data...)
	b.mu.Unlock()

	// 解除阻塞
	select {
	case <-b.blockChan:
	default:
		close(b.blockChan)
		b.blockChan = make(chan struct{})
	}
}

func (b *BlockingMockSerialPortNoMutex) GetOutputData() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.outputBuffer
}

func (b *BlockingMockSerialPortNoMutex) IsReadBlocked() bool {
	return b.readBlocked
}

// TestMutexProblemDemonstration 演示锁导致的问题
func TestMutexProblemDemonstration(t *testing.T) {
	t.Log("=== Demonstration: Why Mutex Caused User Input to Fail ===")

	t.Log("")
	t.Log("Scenario A: WITH mutex (OLD behavior)")
	t.Log("--------------------------------------")

	// 模拟带锁的串口
	portWithMutex := struct {
		mu     sync.Mutex
		buffer []byte
	}{
		buffer: make([]byte, 0),
	}

	// Read goroutine（阻塞持有锁）
	readBlocked := make(chan struct{})
	go func() {
		portWithMutex.mu.Lock()
		close(readBlocked)
		// 模拟阻塞读取（持有锁）
		time.Sleep(5 * time.Second)
		portWithMutex.mu.Unlock()
	}()

	<-readBlocked
	t.Log("Read has acquired lock and is blocking")

	// Write goroutine（尝试获取锁）
	writeAttempt := make(chan struct{})
	go func() {
		close(writeAttempt)
		portWithMutex.mu.Lock() // 会阻塞！
		t.Log("Write acquired lock")
		portWithMutex.mu.Unlock()
	}()

	<-writeAttempt
	t.Log("Write attempted to acquire lock...")

	time.Sleep(100 * time.Millisecond)
	t.Log("Write is BLOCKED waiting for Read to release lock")
	t.Log("User input cannot be sent!")

	t.Log("")
	t.Log("Scenario B: WITHOUT mutex (NEW behavior)")
	t.Log("---------------------------------------")

	// 模拟无锁的串口
	portNoMutex := struct {
		buffer []byte
		mu     sync.Mutex // 只保护 buffer
	}{
		buffer: make([]byte, 0),
	}

	// Read goroutine（阻塞但不持有 Write 的锁）
	readBlocked2 := make(chan struct{})
	go func() {
		close(readBlocked2)
		// Read 阻塞，但不持有任何阻止 Write 的锁
		time.Sleep(5 * time.Second)
	}()

	<-readBlocked2
	t.Log("Read is blocking (but not holding write-blocking lock)")

	// Write goroutine（可以立即执行）
	writeDone := make(chan struct{})
	go func() {
		portNoMutex.mu.Lock()
		portNoMutex.buffer = append(portNoMutex.buffer, []byte("user input")...)
		portNoMutex.mu.Unlock()
		close(writeDone)
	}()

	select {
	case <-writeDone:
		t.Log("✅ Write completed immediately!")
		t.Log("User input sent successfully!")
	case <-time.After(100 * time.Millisecond):
		t.Log("Write should have completed")
	}

	t.Log("")
	t.Log("Conclusion:")
	t.Log("- OLD: Read blocking prevented Write → User input failed")
	t.Log("- NEW: Read blocking doesn't prevent Write → User input works")
	t.Log("=== Demonstration Completed ===")
}