package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()

	// 验证默认值
	if cfg.Serial.Path != "/dev/ttyUSB0" {
		t.Errorf("Expected Serial.Path=/dev/ttyUSB0, got %s", cfg.Serial.Path)
	}
	if cfg.Serial.BaudRate != 115200 {
		t.Errorf("Expected Serial.BaudRate=115200, got %d", cfg.Serial.BaudRate)
	}
	if cfg.Server.Port != 7700 {
		t.Errorf("Expected Server.Port=7700, got %d", cfg.Server.Port)
	}
	if cfg.Server.Address != "0.0.0.0" {
		t.Errorf("Expected Server.Address=0.0.0.0, got %s", cfg.Server.Address)
	}
	if cfg.Arbiter.Timeout != 30 {
		t.Errorf("Expected Arbiter.Timeout=30, got %d", cfg.Arbiter.Timeout)
	}
}

func TestDefaultClientConfig(t *testing.T) {
	cfg := DefaultClientConfig()

	// 验证默认值
	if cfg.Server.Address != "127.0.0.1" {
		t.Errorf("Expected Server.Address=127.0.0.1, got %s", cfg.Server.Address)
	}
	if cfg.Server.Port != 7700 {
		t.Errorf("Expected Server.Port=7700, got %d", cfg.Server.Port)
	}
	if cfg.PTY.Path != "/tmp/vttyShare0" {
		t.Errorf("Expected PTY.Path=/tmp/vttyShare0, got %s", cfg.PTY.Path)
	}
	if cfg.Reconnect.Interval != 5 {
		t.Errorf("Expected Reconnect.Interval=5, got %d", cfg.Reconnect.Interval)
	}
}

func TestLoadServerFromFile(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "server.yaml")

	configContent := `
serial:
  path: "/dev/ttyACM0"
  baudrate: 921600
server:
  port: 8000
  address: "127.0.0.1"
arbiter:
  timeout: 60
log:
  level: "debug"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 加载配置
	cfg, err := LoadServerFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 验证加载的值
	if cfg.Serial.Path != "/dev/ttyACM0" {
		t.Errorf("Expected Serial.Path=/dev/ttyACM0, got %s", cfg.Serial.Path)
	}
	if cfg.Serial.BaudRate != 921600 {
		t.Errorf("Expected Serial.BaudRate=921600, got %d", cfg.Serial.BaudRate)
	}
	if cfg.Server.Port != 8000 {
		t.Errorf("Expected Server.Port=8000, got %d", cfg.Server.Port)
	}
	if cfg.Arbiter.Timeout != 60 {
		t.Errorf("Expected Arbiter.Timeout=60, got %d", cfg.Arbiter.Timeout)
	}
}

func TestLoadClientFromFile(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "client.yaml")

	configContent := `
server:
  address: "192.168.1.100"
  port: 9000
pty:
  path: "/dev/vttyShare1"
reconnect:
  enabled: false
  interval: 10
  max_retry: 5
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 加载配置
	cfg, err := LoadClientFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 验证加载的值
	if cfg.Server.Address != "192.168.1.100" {
		t.Errorf("Expected Server.Address=192.168.1.100, got %s", cfg.Server.Address)
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("Expected Server.Port=9000, got %d", cfg.Server.Port)
	}
	if cfg.PTY.Path != "/dev/vttyShare1" {
		t.Errorf("Expected PTY.Path=/dev/vttyShare1, got %s", cfg.PTY.Path)
	}
	if cfg.Reconnect.Enabled != false {
		t.Errorf("Expected Reconnect.Enabled=false, got %v", cfg.Reconnect.Enabled)
	}
}

func TestLoadServerNotFound(t *testing.T) {
	// 加载不存在的文件
	cfg, err := LoadServer("/nonexistent/path/server.yaml")

	// 应返回默认配置（不报错）
	if err != nil {
		t.Errorf("Expected no error for missing file, got %v", err)
	}
	if cfg == nil {
		t.Error("Expected default config, got nil")
	}
	if cfg.Server.Port != 7700 {
		t.Errorf("Expected default port 7700, got %d", cfg.Server.Port)
	}
}

func TestLoadClientNotFound(t *testing.T) {
	// 加载不存在的文件
	cfg, err := LoadClient("/nonexistent/path/client.yaml")

	// 应返回默认配置（不报错）
	if err != nil {
		t.Errorf("Expected no error for missing file, got %v", err)
	}
	if cfg == nil {
		t.Error("Expected default config, got nil")
	}
	if cfg.Server.Port != 7700 {
		t.Errorf("Expected default port 7700, got %d", cfg.Server.Port)
	}
}

func TestApplyDefaults(t *testing.T) {
	// 创建部分配置
	cfg := &ServerConfig{}
	cfg.Serial.Path = "/dev/ttyACM0" // 只设置部分值

	// 应用默认值
	cfg = ApplyDefaults(cfg)

	// 验证设置的值保留
	if cfg.Serial.Path != "/dev/ttyACM0" {
		t.Errorf("Expected Serial.Path=/dev/ttyACM0, got %s", cfg.Serial.Path)
	}

	// 验证未设置的值使用默认值
	if cfg.Serial.BaudRate != 115200 {
		t.Errorf("Expected default BaudRate 115200, got %d", cfg.Serial.BaudRate)
	}
	if cfg.Server.Port != 7700 {
		t.Errorf("Expected default Port 7700, got %d", cfg.Server.Port)
	}
}

func TestApplyClientDefaults(t *testing.T) {
	// 创建部分配置
	cfg := &ClientConfig{}
	cfg.Server.Address = "192.168.1.50" // 只设置部分值

	// 应用默认值
	cfg = ApplyClientDefaults(cfg)

	// 验证设置的值保留
	if cfg.Server.Address != "192.168.1.50" {
		t.Errorf("Expected Server.Address=192.168.1.50, got %s", cfg.Server.Address)
	}

	// 验证未设置的值使用默认值
	if cfg.Server.Port != 7700 {
		t.Errorf("Expected default Port 7700, got %d", cfg.Server.Port)
	}
}

func TestServerAddress(t *testing.T) {
	cfg := &ClientConfig{}
	cfg.Server.Address = "192.168.1.100"
	cfg.Server.Port = 7700

	expected := "192.168.1.100:7700"
	if cfg.ServerAddress() != expected {
		t.Errorf("Expected ServerAddress=%s, got %s", expected, cfg.ServerAddress())
	}
}

func TestListenAddress(t *testing.T) {
	cfg := &ServerConfig{}
	cfg.Server.Address = "0.0.0.0"
	cfg.Server.Port = 7700

	expected := "0.0.0.0:7700"
	if cfg.ListenAddress() != expected {
		t.Errorf("Expected ListenAddress=%s, got %s", expected, cfg.ListenAddress())
	}
}

func TestLoadExistingConfig(t *testing.T) {
	// 测试加载项目中已有的配置文件
	cfg, err := LoadServerFromFile("./configs/server.yaml")
	if err != nil {
		t.Logf("Skipping test: configs/server.yaml not found")
		return
	}

	// 验证配置文件中的值
	if cfg.Serial.Path != "/dev/ttyUSB0" {
		t.Errorf("Expected Serial.Path=/dev/ttyUSB0, got %s", cfg.Serial.Path)
	}
	if cfg.Server.Port != 7700 {
		t.Errorf("Expected Server.Port=7700, got %d", cfg.Server.Port)
	}
}

func TestSaveAndLoad(t *testing.T) {
	// 创建配置
	cfg := DefaultServerConfig()
	cfg.Server.Port = 8888

	// 保存到临时文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	err := SaveServer(cfg, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// 重新加载
	loaded, err := LoadServerFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	// 验证值一致
	if loaded.Server.Port != 8888 {
		t.Errorf("Expected Port=8888, got %d", loaded.Server.Port)
	}
}