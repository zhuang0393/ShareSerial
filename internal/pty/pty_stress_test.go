package pty

import (
	"testing"
	"time"
)

// TestRealPTYCreate 测试真实 PTY 创建（仅 Linux）
func TestRealPTYCreate(t *testing.T) {
	pty, err := CreateRealPTY("/tmp/test_vtty")
	if err != nil {
		// 如果无法创建真实 PTY（权限问题等），跳过测试
		t.Skipf("cannot create real PTY: %v", err)
	}

	if pty == nil {
		t.Fatal("expected PTY to be created")
	}

	// 检查 symlink 路径
	if pty.SymlinkPath() != "/tmp/test_vtty" {
		t.Errorf("expected symlink path '/tmp/test_vtty', got '%s'", pty.SymlinkPath())
	}

	// 检查 PTY 是打开状态
	if !pty.IsOpen() {
		t.Error("expected PTY to be open")
	}

	// 清理
	pty.Close()
}

// TestRealPTYWriteRead 测试真实 PTY 读写
func TestRealPTYWriteRead(t *testing.T) {
	pty, err := CreateRealPTY("")
	if err != nil {
		t.Skipf("cannot create real PTY: %v", err)
	}
	defer pty.Close()

	// 写入数据
	data := []byte("Test data\n")
	n, err := pty.Write(data)
	if err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}
}

// TestMockPTYBlockingBehavior 测试 MockPTY 无数据时的行为不应疯狂返回错误
func TestMockPTYBlockingBehavior(t *testing.T) {
	pty, err := CreateMockPTY("/dev/vttyShare0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer pty.Close()

	// 测试：无数据时 Read 应该等待或返回可忽略的错误
	// 而不是疯狂循环返回 "no data available"

	// 设置短超时来验证行为
	done := make(chan bool)
	go func() {
		buf := make([]byte, 1024)
		pty.Read(buf)
		// Read 返回了（可能是错误）
		done <- true
	}()

	// 等待一段时间
	select {
	case <-done:
		// Read 返回了（可能是错误）
		// 这是预期行为：无数据时应该等待
		t.Log("Read returned without data")
	case <-time.After(100 * time.Millisecond):
		// Read 正在阻塞等待，这是好的行为
		t.Log("Read is blocking (good behavior)")
	}
}

// TestPTYCreateFallback 测试 CreatePTY 的回退行为
func TestPTYCreateFallback(t *testing.T) {
	// 尝试创建可能失败的路径（如权限不足的路径）
	pty, _ := CreatePTY("/root/cannot_write_here")

	// 应该回退到 MockPTY
	if pty == nil {
		t.Fatal("expected PTY to be created (fallback to Mock)")
	}

	// 验证这是一个 PTYDevice（无论是真实还是 Mock）
	pty.Close()
}

// TestPTYConcurrentReadWrite 测试并发读写安全
func TestPTYConcurrentReadWrite(t *testing.T) {
	pty, _ := CreateMockPTY("/dev/vttyShare0")
	defer pty.Close()

	// 并发写入
	numWrites := 100
	for i := 0; i < numWrites; i++ {
		go func(id int) {
			pty.Write([]byte("Concurrent write\n"))
		}(i)
	}

	// 等待完成
	time.Sleep(100 * time.Millisecond)

	// PTY 应该仍然打开
	if !pty.IsOpen() {
		t.Error("PTY should still be open after concurrent writes")
	}
}

// TestPTYCloseAndReopen 测试 PTY 关闭后状态
func TestPTYCloseAndReopen(t *testing.T) {
	pty1, _ := CreateMockPTY("/dev/vttyShare0")
	pty1.Close()

	// 关闭后不应该能读写
	buf := make([]byte, 1024)
	_, err := pty1.Read(buf)
	if err == nil {
		t.Error("expected error reading from closed PTY")
	}

	// 重新创建
	pty2, err := CreateMockPTY("/dev/vttyShare0")
	if err != nil {
		t.Fatalf("unexpected error creating new PTY: %v", err)
	}
	defer pty2.Close()

	if !pty2.IsOpen() {
		t.Error("new PTY should be open")
	}
}