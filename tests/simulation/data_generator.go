package simulation

import (
	"fmt"
	"sync"
	"time"
)

// DataGenerator 模拟开发板数据生成器
type DataGenerator struct {
	mu           sync.Mutex
	vsp          *VirtualSerialPair
	running      bool
	stopChan     chan struct{}
	lineCount    int
	patterns     []LogPattern
	customData   []string
}

// LogPattern 日志模式配置
type LogPattern struct {
	Level   string
	Message string
	Weight  int // 出现权重
}

// DefaultLogPatterns 返回默认日志模式
func DefaultLogPatterns() []LogPattern {
	return []LogPattern{
		{Level: "INFO", Message: "System starting", Weight: 5},
		{Level: "INFO", Message: "Loading kernel modules", Weight: 3},
		{Level: "INFO", Message: "Mounting filesystems", Weight: 2},
		{Level: "DEBUG", Message: "Initializing hardware", Weight: 10},
		{Level: "WARN", Message: "Low memory warning", Weight: 1},
		{Level: "ERROR", Message: "Failed to mount /data", Weight: 1},
		{Level: "INFO", Message: "Network interface up", Weight: 3},
		{Level: "DEBUG", Message: "UART receive: 0x%02X", Weight: 20},
		{Level: "INFO", Message: "Application started", Weight: 2},
	}
}

// AndroidBootPatterns 返回 Android 启动日志模式
func AndroidBootPatterns() []LogPattern {
	return []LogPattern{
		{Level: "INFO", Message: "init: Starting service 'ueventd'", Weight: 5},
		{Level: "INFO", Message: "init: Starting service 'console'", Weight: 3},
		{Level: "DEBUG", Message: "SELinux: Initializing", Weight: 2},
		{Level: "INFO", Message: "init: Starting service 'logd'", Weight: 2},
		{Level: "DEBUG", Message: "lowmemorykiller: start", Weight: 1},
		{Level: "INFO", Message: "ActivityManager: START u0 {com.android.launcher}", Weight: 5},
		{Level: "WARN", Message: "PackageManager: Unknown permission", Weight: 3},
		{Level: "ERROR", Message: "System.err: java.lang.NullPointerException", Weight: 1},
		{Level: "INFO", Message: "Boot completed in %d ms", Weight: 1},
	}
}

// NewDataGenerator 创建数据生成器
func NewDataGenerator(vsp *VirtualSerialPair) *DataGenerator {
	return &DataGenerator{
		vsp:      vsp,
		patterns: DefaultLogPatterns(),
		stopChan: make(chan struct{}),
	}
}

// NewDataGeneratorWithPatterns 使用自定义模式创建生成器
func NewDataGeneratorWithPatterns(vsp *VirtualSerialPair, patterns []LogPattern) *DataGenerator {
	return &DataGenerator{
		vsp:      vsp,
		patterns: patterns,
		stopChan: make(chan struct{}),
	}
}

// SetPatterns 设置日志模式
func (dg *DataGenerator) SetPatterns(patterns []LogPattern) {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	dg.patterns = patterns
}

// SetCustomData 设置自定义数据
func (dg *DataGenerator) SetCustomData(data []string) {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	dg.customData = data
}

// StartContinuous 启动持续数据生成（后台运行）
func (dg *DataGenerator) StartContinuous(interval time.Duration) {
	dg.mu.Lock()
	dg.running = true
	dg.mu.Unlock()

	go func() {
		for {
			select {
			case <-dg.stopChan:
				return
			default:
				dg.GenerateOneLine()
				time.Sleep(interval)
			}
		}
	}()
}

// Stop 停止数据生成
func (dg *DataGenerator) Stop() {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	if dg.running {
		close(dg.stopChan)
		dg.running = false
	}
}

// GenerateOneLine 生成一行日志
func (dg *DataGenerator) GenerateOneLine() error {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	// 选择一个模式（随机权重）
	pattern := dg.selectPattern()
	timestamp := time.Now().Format("15:04:05.000")

	line := fmt.Sprintf("[%s] %s: %s\n", timestamp, pattern.Level, pattern.Message)

	if err := dg.vsp.InjectData(line); err != nil {
		return err
	}

	dg.lineCount++
	return nil
}

// selectPattern 选择日志模式
func (dg *DataGenerator) selectPattern() LogPattern {
	// 简化实现：循环选择
	totalWeight := 0
	for _, p := range dg.patterns {
		totalWeight += p.Weight
	}

	// 使用计数器轮换
	index := dg.lineCount % len(dg.patterns)
	return dg.patterns[index]
}

// GenerateMultipleLines 生成多行日志
func (dg *DataGenerator) GenerateMultipleLines(count int) error {
	for i := 0; i < count; i++ {
		if err := dg.GenerateOneLine(); err != nil {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

// GenerateCustomData 生成自定义数据
func (dg *DataGenerator) GenerateCustomData() error {
	dg.mu.Lock()
	customData := dg.customData
	dg.mu.Unlock()

	for _, line := range customData {
		if err := dg.vsp.InjectData(line + "\n"); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// GenerateBootSequence 生成启动序列日志
func (dg *DataGenerator) GenerateBootSequence() error {
	// 模拟系统启动过程
	sequences := []struct {
		delay   time.Duration
		level   string
		message string
	}{
		{0 * time.Millisecond, "INFO", "System starting..."},
		{100 * time.Millisecond, "DEBUG", "Initializing hardware..."},
		{200 * time.Millisecond, "INFO", "Loading kernel modules..."},
		{300 * time.Millisecond, "INFO", "Mounting filesystems..."},
		{400 * time.Millisecond, "DEBUG", "Configuring network..."},
		{500 * time.Millisecond, "INFO", "Network interface eth0 up"},
		{600 * time.Millisecond, "WARN", "Low memory: 256MB available"},
		{700 * time.Millisecond, "INFO", "Starting services..."},
		{800 * time.Millisecond, "ERROR", "Failed to start service 'adb'"},
		{900 * time.Millisecond, "INFO", "System ready"},
	}

	for _, seq := range sequences {
		time.Sleep(seq.delay)
		if err := dg.vsp.InjectLogLine(seq.level, seq.message); err != nil {
			return err
		}
	}

	return nil
}

// GenerateKernelLog 生成内核日志格式
func (dg *DataGenerator) GenerateKernelLog(message string) error {
	timestamp := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] kernel: %s\n", timestamp, message)
	return dg.vsp.InjectData(line)
}

// GenerateAndroidLog 生成 Android 日志格式
func (dg *DataGenerator) GenerateAndroidLog(tag, level, message string) error {
	timestamp := time.Now().Format("15:04:05.000")
	line := fmt.Sprintf("[%s] %s/%s: %s\n", timestamp, level, tag, message)
	return dg.vsp.InjectData(line)
}

// GenerateInteractivePrompt 生成交互式提示符
func (dg *DataGenerator) GenerateInteractivePrompt(prompt string) error {
	return dg.vsp.InjectData(prompt)
}

// GenerateCommandResponse 生成命令响应
func (dg *DataGenerator) GenerateCommandResponse(command, response string) error {
	// 先输出命令回显
	dg.vsp.InjectData(command + "\n")
	time.Sleep(50 * time.Millisecond)
	// 输出响应
	return dg.vsp.InjectData(response + "\n")
}

// GenerateHighFrequencyData 高频数据生成（测试性能）
func (dg *DataGenerator) GenerateHighFrequencyData(count int, intervalMs int) error {
	for i := 0; i < count; i++ {
		line := fmt.Sprintf("LINE_%05d: %s\n", i, time.Now().Format("15:04:05.000"))
		if err := dg.vsp.InjectData(line); err != nil {
			return err
		}
		time.Sleep(time.Duration(intervalMs) * time.Millisecond)
	}
	return nil
}

// LineCount 返回已生成的行数
func (dg *DataGenerator) LineCount() int {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	return dg.lineCount
}

// ResetLineCount 重置行计数
func (dg *DataGenerator) ResetLineCount() {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	dg.lineCount = 0
}