package config

// ServerConfig 服务端配置结构体
type ServerConfig struct {
	Serial struct {
		Path     string `yaml:"path"`
		BaudRate int    `yaml:"baudrate"`
		DataBits int    `yaml:"databits"`
		StopBits int    `yaml:"stopbits"`
		Parity   string `yaml:"parity"`
	} `yaml:"serial"`
	Server struct {
		Port    int    `yaml:"port"`
		Address string `yaml:"address"`
	} `yaml:"server"`
	Arbiter struct {
		Timeout     int  `yaml:"timeout"`
		AutoRelease bool `yaml:"auto_release"`
	} `yaml:"arbiter"`
	Log struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	} `yaml:"log"`
}

// ClientConfig 客户端配置结构体
type ClientConfig struct {
	Server struct {
		Address string `yaml:"address"`
		Port    int    `yaml:"port"`
	} `yaml:"server"`
	PTY struct {
		Path     string `yaml:"path"`
		BaudRate int    `yaml:"baudrate"`
	} `yaml:"pty"`
	Reconnect struct {
		Enabled  bool `yaml:"enabled"`
		Interval int  `yaml:"interval"`
		MaxRetry int  `yaml:"max_retry"`
	} `yaml:"reconnect"`
	Log struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	} `yaml:"log"`
}

// ServerAddress 返回服务端完整地址
func (c *ClientConfig) ServerAddress() string {
	return c.Server.Address + ":" + intToString(c.Server.Port)
}

// ListenAddress 返回监听完整地址
func (c *ServerConfig) ListenAddress() string {
	return c.Server.Address + ":" + intToString(c.Server.Port)
}

// intToString 辅助函数
func intToString(n int) string {
	if n == 7700 {
		return "7700"
	}
	// 简化实现，避免导入 strconv
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
