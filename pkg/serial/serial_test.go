package serial

import (
	"testing"
)

// TestSerialConfig 测试串口配置
func TestSerialConfig(t *testing.T) {
	config := DefaultConfig()

	if config.BaudRate != 115200 {
		t.Errorf("expected baudrate 115200, got %d", config.BaudRate)
	}
	if config.DataBits != 8 {
		t.Errorf("expected databits 8, got %d", config.DataBits)
	}
	if config.StopBits != 1 {
		t.Errorf("expected stopbits 1, got %d", config.StopBits)
	}
	if config.Parity != ParityNone {
		t.Errorf("expected parity None, got %d", config.Parity)
	}
}

// TestMockSerialOpenClose 测试 Mock 串口打开和关闭
func TestMockSerialOpenClose(t *testing.T) {
	mock := NewMockSerialPort()

	err := mock.Open("/dev/ttyMock0")
	if err != nil {
		t.Fatalf("unexpected error opening mock serial: %v", err)
	}

	if !mock.IsOpen() {
		t.Error("expected mock serial to be open")
	}

	err = mock.Close()
	if err != nil {
		t.Fatalf("unexpected error closing mock serial: %v", err)
	}

	if mock.IsOpen() {
		t.Error("expected mock serial to be closed")
	}
}

// TestMockSerialReadWrite 测试 Mock 串口读写
func TestMockSerialReadWrite(t *testing.T) {
	mock := NewMockSerialPort()
	mock.Open("/dev/ttyMock0")

	// 模拟数据输入
	mock.InjectInput([]byte("Hello from serial\n"))

	// 读取数据
	buf := make([]byte, 1024)
	n, err := mock.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error reading: %v", err)
	}

	expected := "Hello from serial\n"
	if string(buf[:n]) != expected {
		t.Errorf("expected '%s', got '%s'", expected, string(buf[:n]))
	}

	// 写入数据
	data := []byte("Command sent\n")
	n, err = mock.Write(data)
	if err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}

	// 检查写入的数据被记录
	written := mock.GetWrittenData()
	if string(written) != "Command sent\n" {
		t.Errorf("expected written data 'Command sent\\n', got '%s'", string(written))
	}
}

// TestSerialScanner 测试串口扫描
func TestSerialScanner(t *testing.T) {
	scanner := NewScanner()

	// 在没有真实串口的环境下，扫描应该返回空列表或错误
	ports, err := scanner.Scan()

	// 期望：要么返回空列表，要么返回错误（取决于环境）
	// 我们不强制要求特定结果，只测试接口可用
	if err != nil && ports != nil {
		t.Errorf("if error returned, ports should be nil")
	}
}

// TestSerialHandlerWithMock 测试使用 Mock 的串口处理器
func TestSerialHandlerWithMock(t *testing.T) {
	mock := NewMockSerialPort()
	handler := NewHandlerWithPort(mock)

	// 模拟数据输入（一次注入，一次性读取）
	mock.InjectInput([]byte("Log line 1\nLog line 2\n"))

	// 通过处理器读取
	buf := make([]byte, 1024)
	n, _ := handler.Read(buf)

	expected := "Log line 1\nLog line 2\n"
	if string(buf[:n]) != expected {
		t.Errorf("expected '%s', got '%s'", expected, string(buf[:n]))
	}
}

// TestSerialHandlerWriteWithLock 测试带写锁的写入
func TestSerialHandlerWriteWithLock(t *testing.T) {
	mock := NewMockSerialPort()
	handler := NewHandlerWithPort(mock)

	// 模拟获取写锁
	handler.SetWriteLockOwner("client1")

	// Client1 可以写入
	data := []byte("reboot\n")
	n, err := handler.WriteWithLock("client1", data)
	if err != nil {
		t.Fatalf("unexpected error writing with lock: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}

	// Client2 不能写入
	_, err = handler.WriteWithLock("client2", []byte("test\n"))
	if err == nil {
		t.Error("expected error when writing without lock")
	}
}

// TestSerialBuffer 测试串口数据缓冲
func TestSerialBuffer(t *testing.T) {
	mock := NewMockSerialPort()
	mock.Open("/dev/ttyMock0")

	// 快速注入大量数据
	for i := 0; i < 10; i++ {
		mock.InjectInput([]byte("Data line\n"))
	}

	// 读取所有数据
	totalRead := 0
	buf := make([]byte, 1024)
	for {
		n, err := mock.Read(buf)
		if err != nil {
			break
		}
		totalRead += n
		if totalRead >= 90 { // 10 lines * 9 bytes
			break
		}
	}

	if totalRead < 90 {
		t.Errorf("expected at least 90 bytes read, got %d", totalRead)
	}
}
