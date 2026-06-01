package arbiter

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrLockAlreadyHeld = errors.New("lock is already held by another client")
	ErrNotLockOwner    = errors.New("client is not the lock owner")
	ErrLockNotHeld     = errors.New("lock is not currently held")
)

// Arbiter 实现写入仲裁（独占模式）
type Arbiter struct {
	mu          sync.Mutex
	owner       string
	timeout     time.Duration
	expireTimer *time.Timer
	locked      bool
}

// NewArbiter 创建一个新的仲裁器
func NewArbiter(timeout time.Duration) *Arbiter {
	return &Arbiter{
		timeout: timeout,
	}
}

// Acquire 尝试获取写锁
// 返回 true 表示成功获取，false 表示锁已被占用
func (a *Arbiter) Acquire(clientID string) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.locked {
		return false, ErrLockAlreadyHeld
	}

	a.owner = clientID
	a.locked = true

	// 设置超时定时器
	if a.expireTimer != nil {
		a.expireTimer.Stop()
	}
	a.expireTimer = time.AfterFunc(a.timeout, func() {
		a.mu.Lock()
		if a.owner == clientID && a.locked {
			a.locked = false
			a.owner = ""
		}
		a.mu.Unlock()
	})

	return true, nil
}

// Release 释放写锁（仅持有者可释放）
func (a *Arbiter) Release(clientID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.locked {
		return ErrLockNotHeld
	}

	if a.owner != clientID {
		return ErrNotLockOwner
	}

	a.locked = false
	a.owner = ""
	if a.expireTimer != nil {
		a.expireTimer.Stop()
	}

	return nil
}

// ForceRelease 强制释放锁（用于断开连接场景）
func (a *Arbiter) ForceRelease(clientID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.owner == clientID {
		a.locked = false
		a.owner = ""
		if a.expireTimer != nil {
			a.expireTimer.Stop()
		}
	}
}

// IsLocked 检查锁是否被占用
func (a *Arbiter) IsLocked() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.locked
}

// Owner 获取当前锁持有者
func (a *Arbiter) Owner() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.owner
}

// ExtendTimeout 延长锁超时时间
func (a *Arbiter) ExtendTimeout(clientID string, newTimeout time.Duration) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.locked {
		return ErrLockNotHeld
	}

	if a.owner != clientID {
		return ErrNotLockOwner
	}

	// 重新设置超时定时器
	if a.expireTimer != nil {
		a.expireTimer.Stop()
	}
	a.timeout = newTimeout
	a.expireTimer = time.AfterFunc(newTimeout, func() {
		a.mu.Lock()
		if a.owner == clientID && a.locked {
			a.locked = false
			a.owner = ""
		}
		a.mu.Unlock()
	})

	return nil
}
