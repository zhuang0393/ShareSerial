package arbiter

import (
	"sync"
	"testing"
	"time"
)

// TestArbiterAcquireLock 测试获取写锁成功
func TestArbiterAcquireLock(t *testing.T) {
	arbiter := NewArbiter(30 * time.Second)

	// Client1 获取锁
	ok, err := arbiter.Acquire("client1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected lock to be acquired")
	}
	if !arbiter.IsLocked() {
		t.Error("expected arbiter to be locked")
	}
	if arbiter.Owner() != "client1" {
		t.Errorf("expected owner 'client1', got '%s'", arbiter.Owner())
	}
}

// TestArbiterAcquireLockAlreadyLocked 测试锁已被占用时获取失败
func TestArbiterAcquireLockAlreadyLocked(t *testing.T) {
	arbiter := NewArbiter(30 * time.Second)

	// Client1 获取锁
	_, _ = arbiter.Acquire("client1")

	// Client2 尝试获取锁
	ok, err := arbiter.Acquire("client2")
	if err == nil {
		t.Error("expected error when lock already held")
	}
	if ok {
		t.Error("expected lock acquisition to fail")
	}
	if arbiter.Owner() != "client1" {
		t.Errorf("expected owner 'client1', got '%s'", arbiter.Owner())
	}
}

// TestArbiterReleaseLock 测试释放写锁
func TestArbiterReleaseLock(t *testing.T) {
	arbiter := NewArbiter(30 * time.Second)

	// Client1 获取锁
	_, _ = arbiter.Acquire("client1")

	// Client1 释放锁
	err := arbiter.Release("client1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if arbiter.IsLocked() {
		t.Error("expected arbiter to be unlocked")
	}
	if arbiter.Owner() != "" {
		t.Errorf("expected empty owner, got '%s'", arbiter.Owner())
	}
}

// TestArbiterReleaseLockNotOwner 测试非持有者释放锁失败
func TestArbiterReleaseLockNotOwner(t *testing.T) {
	arbiter := NewArbiter(30 * time.Second)

	// Client1 获取锁
	_, _ = arbiter.Acquire("client1")

	// Client2 尝试释放锁
	err := arbiter.Release("client2")
	if err == nil {
		t.Error("expected error when releasing lock not owned")
	}
	if !arbiter.IsLocked() {
		t.Error("expected arbiter to remain locked")
	}
	if arbiter.Owner() != "client1" {
		t.Errorf("expected owner 'client1', got '%s'", arbiter.Owner())
	}
}

// TestArbiterLockTimeout 测试锁超时自动释放
func TestArbiterLockTimeout(t *testing.T) {
	// 使用短超时测试
	arbiter := NewArbiter(100 * time.Millisecond)

	// Client1 获取锁
	arbiter.Acquire("client1")
	if !arbiter.IsLocked() {
		t.Fatal("expected arbiter to be locked initially")
	}

	// 等待超时
	time.Sleep(150 * time.Millisecond)

	// 检查锁已释放
	if arbiter.IsLocked() {
		t.Error("expected lock to be released after timeout")
	}
	if arbiter.Owner() != "" {
		t.Errorf("expected empty owner after timeout, got '%s'", arbiter.Owner())
	}
}

// TestArbiterConcurrentAcquire 测试并发获取锁
func TestArbiterConcurrentAcquire(t *testing.T) {
	arbiter := NewArbiter(30 * time.Second)

	var wg sync.WaitGroup
	winners := make([]string, 0)
	losers := make([]string, 0)
	var mu sync.Mutex

	// 10 个客户端同时尝试获取锁
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			ok, _ := arbiter.Acquire(id)
			mu.Lock()
			if ok {
				winners = append(winners, id)
			} else {
				losers = append(losers, id)
			}
			mu.Unlock()
		}(string(rune('A' + i)))
	}

	wg.Wait()

	// 只有一个赢家
	if len(winners) != 1 {
		t.Errorf("expected 1 winner, got %d", len(winners))
	}
	if len(losers) != 9 {
		t.Errorf("expected 9 losers, got %d", len(losers))
	}
}

// TestArbiterOwnerDisconnect 测试持有者断开后锁释放
func TestArbiterOwnerDisconnect(t *testing.T) {
	arbiter := NewArbiter(30 * time.Second)

	// Client1 获取锁
	arbiter.Acquire("client1")

	// 模拟断开（强制释放）
	arbiter.ForceRelease("client1")

	// 检查锁已释放
	if arbiter.IsLocked() {
		t.Error("expected lock to be released after disconnect")
	}

	// Client2 可以获取锁
	ok, _ := arbiter.Acquire("client2")
	if !ok {
		t.Error("expected client2 to acquire lock after disconnect")
	}
}

// TestArbiterExtendTimeout 测试延长锁超时
func TestArbiterExtendTimeout(t *testing.T) {
	arbiter := NewArbiter(100 * time.Millisecond)

	// Client1 获取锁
	_, _ = arbiter.Acquire("client1")

	// 在超时前延长
	time.Sleep(50 * time.Millisecond)
	_ = arbiter.ExtendTimeout("client1", 200*time.Millisecond)

	// 原超时时间后，锁仍存在
	time.Sleep(100 * time.Millisecond)
	if !arbiter.IsLocked() {
		t.Error("expected lock to still be held after extending timeout")
	}

	// 新超时时间后，锁释放
	time.Sleep(150 * time.Millisecond)
	if arbiter.IsLocked() {
		t.Error("expected lock to be released after new timeout")
	}
}
