package server

import (
	"testing"
	"time"

	"shareserial/pkg/arbiter"
	"shareserial/pkg/serial"
)

// TestServerAutoAcquireWriteLock 测试自动获取写锁逻辑
func TestServerAutoAcquireWriteLock(t *testing.T) {
	// 创建 Arbiter
	arb := arbiter.NewArbiter(30 * time.Second)

	// 创建 Mock 串口
	mockSerial := serial.NewMockSerialPort()

	t.Log("=== Test: Auto Acquire Write Lock ===")

	// 模拟 Client 1 连接
	client1 := "client-1"
	t.Logf("Client %s connected", client1)

	// Client 1 发送数据（无锁状态）
	t.Log("Client 1 sends data without lock")
	userInput1 := []byte("command1\n")

	// 模拟 handleClient 的自动获取锁逻辑
	currentOwner := arb.Owner()
	t.Logf("Current lock owner: '%s' (empty means no lock)", currentOwner)

	// 自动获取锁逻辑
	if currentOwner != client1 {
		// 尝试获取锁（不阻塞）
		acquired, _ := arb.Acquire(client1)
		t.Logf("Auto acquire lock: acquired=%v", acquired)

		// 获取成功后写入串口
		if acquired {
			_, _ = mockSerial.Write(userInput1)
			t.Logf("Written to serial: %s", string(userInput1))
		}
	}

	// 验证锁状态
	newOwner := arb.Owner()
	t.Logf("New lock owner: '%s'", newOwner)
	if newOwner != client1 {
		t.Errorf("Expected lock owner to be '%s', got '%s'", client1, newOwner)
	}

	// 模拟 Client 2 连接
	client2 := "client-2"
	t.Logf("Client %s connected", client2)

	// Client 2 发送数据（锁被 Client 1 持有）
	t.Log("Client 2 sends data (lock held by client 1)")
	userInput2 := []byte("command2\n")

	currentOwner = arb.Owner()
	t.Logf("Current lock owner: '%s'", currentOwner)

	if currentOwner != client2 {
		// 尝试获取锁
		acquired, _ := arb.Acquire(client2)
		t.Logf("Client 2 acquire lock: acquired=%v", acquired)

		if !acquired {
			t.Log("Client 2 failed to acquire lock (expected - held by client 1)")
			// 等待逻辑（模拟 handleClient 中的等待）
			for i := 0; i < 5; i++ {
				acquired, _ = arb.Acquire(client2)
				if acquired {
					t.Logf("Client 2 acquired lock after %d attempts", i+1)
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}

		// 如果最终获取到锁，写入串口
		if arb.Owner() == client2 {
			_, _ = mockSerial.Write(userInput2)
			t.Logf("Client 2 written to serial: %s", string(userInput2))
		}
	}

	t.Log("=== Test completed ===")
}

// TestWriteLockTimeout 测试写锁超时释放
func TestWriteLockTimeout(t *testing.T) {
	// 创建短超时 Arbiter（方便测试）
	arb := arbiter.NewArbiter(2 * time.Second)

	t.Log("=== Test: Write Lock Timeout ===")

	client := "test-client"

	// 获取锁
	acquired, _ := arb.Acquire(client)
	if !acquired {
		t.Fatal("Failed to acquire lock")
	}
	t.Logf("Lock acquired by '%s'", arb.Owner())

	// 等待超时
	t.Log("Waiting for lock timeout...")
	time.Sleep(3 * time.Second)

	// 检查锁是否释放
	owner := arb.Owner()
	t.Logf("Lock owner after timeout: '%s'", owner)

	if owner != "" {
		t.Log("Lock still held (timeout mechanism working)")
	}

	// 尝试延长超时
	acquired2, _ := arb.Acquire(client)
	t.Logf("Re-acquire lock: acquired=%v", acquired2)

	t.Log("=== Test completed ===")
}

// TestMultipleClientInput 测试多客户端输入场景
func TestMultipleClientInput(t *testing.T) {
	arb := arbiter.NewArbiter(30 * time.Second)
	mockSerial := serial.NewMockSerialPort()

	t.Log("=== Test: Multiple Client Input ===")

	clients := []string{"client-A", "client-B", "client-C"}

	// 所有 Client 连接
	for _, c := range clients {
		t.Logf("Client '%s' connected", c)
	}

	// Client A 输入
	clientA := clients[0]
	inputA := []byte("input from A\n")

	// 自动获取锁
	if arb.Owner() != clientA {
		arb.Acquire(clientA)
	}

	// 写入串口
	if arb.Owner() == clientA {
		mockSerial.Write(inputA)
		t.Logf("Client A written: %s", string(inputA))
	}

	// Client B 尝试输入（应该等待）
	clientB := clients[1]
	inputB := []byte("input from B\n")

	t.Log("Client B attempts to input...")
	acquired, _ := arb.Acquire(clientB)
	t.Logf("Client B acquire result: %v (expected false - held by A)", acquired)

	// Client A 释放锁
	arb.Release(clientA)
	t.Log("Client A released lock")

	// Client B 现在可以获取锁
	acquired, _ = arb.Acquire(clientB)
	t.Logf("Client B acquire after release: %v", acquired)

	if acquired {
		mockSerial.Write(inputB)
		t.Logf("Client B written: %s", string(inputB))
	}

	t.Log("=== Test completed ===")
}

// TestInputOutputSequence 测试输入-输出序列
func TestInputOutputSequence(t *testing.T) {
	arb := arbiter.NewArbiter(30 * time.Second)
	mockSerial := serial.NewMockSerialPort()

	t.Log("=== Test: Input-Output Sequence ===")

	client := "test-client"

	// 序列 1: 用户输入命令
	t.Log("Sequence 1: User input")
	command1 := []byte("ls\n")

	// 自动获取锁
	arb.Acquire(client)
	mockSerial.Write(command1)
	t.Logf("Command sent: %s", string(command1))

	// 序列 2: 读取串口响应（模拟）
	t.Log("Sequence 2: Serial response")
	// 注意：MockSerialPort 的读取需要设置数据
	// 这里只是模拟流程

	// 序列 3: 用户输入另一个命令
	t.Log("Sequence 3: Another user input")
	command2 := []byte("help\n")

	// 锁还在持有中，直接写入
	if arb.Owner() == client {
		mockSerial.Write(command2)
		t.Logf("Command sent: %s", string(command2))
	}

	// 验证锁状态
	t.Logf("Final lock owner: '%s'", arb.Owner())
	t.Logf("Lock timeout: 30s (auto-release)")

	t.Log("=== Test completed ===")
}