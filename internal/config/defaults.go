package config

// DefaultServerConfig 返回服务端默认配置
func DefaultServerConfig() *ServerConfig {
	cfg := &ServerConfig{}

	// Serial 默认值
	cfg.Serial.Path = "/dev/ttyUSB0"
	cfg.Serial.BaudRate = 115200
	cfg.Serial.DataBits = 8
	cfg.Serial.StopBits = 1
	cfg.Serial.Parity = "none"

	// Server 默认值
	cfg.Server.Port = 7700
	cfg.Server.Address = "0.0.0.0"

	// Arbiter 默认值
	cfg.Arbiter.Timeout = 30
	cfg.Arbiter.AutoRelease = true

	// Log 默认值
	cfg.Log.Level = "info"
	cfg.Log.File = ""

	return cfg
}

// DefaultClientConfig 返回客户端默认配置
func DefaultClientConfig() *ClientConfig {
	cfg := &ClientConfig{}

	// Server 默认值
	cfg.Server.Address = "127.0.0.1"
	cfg.Server.Port = 7700

	// PTY 默认值
	cfg.PTY.Path = "/tmp/vttyShare0"
	cfg.PTY.BaudRate = 115200

	// Reconnect 默认值
	cfg.Reconnect.Enabled = true
	cfg.Reconnect.Interval = 5
	cfg.Reconnect.MaxRetry = 10

	// Log 默认值
	cfg.Log.Level = "info"
	cfg.Log.File = ""

	return cfg
}

// ApplyDefaults 为空字段应用默认值
func ApplyDefaults(cfg *ServerConfig) *ServerConfig {
	defaults := DefaultServerConfig()

	if cfg.Serial.Path == "" {
		cfg.Serial.Path = defaults.Serial.Path
	}
	if cfg.Serial.BaudRate == 0 {
		cfg.Serial.BaudRate = defaults.Serial.BaudRate
	}
	if cfg.Serial.DataBits == 0 {
		cfg.Serial.DataBits = defaults.Serial.DataBits
	}
	if cfg.Serial.StopBits == 0 {
		cfg.Serial.StopBits = defaults.Serial.StopBits
	}
	if cfg.Serial.Parity == "" {
		cfg.Serial.Parity = defaults.Serial.Parity
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = defaults.Server.Port
	}
	if cfg.Server.Address == "" {
		cfg.Server.Address = defaults.Server.Address
	}
	if cfg.Arbiter.Timeout == 0 {
		cfg.Arbiter.Timeout = defaults.Arbiter.Timeout
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = defaults.Log.Level
	}

	return cfg
}

// ApplyClientDefaults 为空字段应用默认值
func ApplyClientDefaults(cfg *ClientConfig) *ClientConfig {
	defaults := DefaultClientConfig()

	if cfg.Server.Address == "" {
		cfg.Server.Address = defaults.Server.Address
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = defaults.Server.Port
	}
	if cfg.PTY.Path == "" {
		cfg.PTY.Path = defaults.PTY.Path
	}
	if cfg.PTY.BaudRate == 0 {
		cfg.PTY.BaudRate = defaults.PTY.BaudRate
	}
	if cfg.Reconnect.Interval == 0 {
		cfg.Reconnect.Interval = defaults.Reconnect.Interval
	}
	if cfg.Reconnect.MaxRetry == 0 {
		cfg.Reconnect.MaxRetry = defaults.Reconnect.MaxRetry
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = defaults.Log.Level
	}

	return cfg
}
