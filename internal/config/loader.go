package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

var (
	ErrConfigNotFound = errors.New("config file not found")
	ErrConfigParse    = errors.New("failed to parse config file")
)

// LoadServer 加载服务端配置文件
// 查找顺序：指定路径 -> /etc/shareserial/server.yaml -> ./configs/server.yaml
func LoadServer(path string) (*ServerConfig, error) {
	// 尝试多个路径
	paths := []string{
		path,
		"/etc/shareserial/server.yaml",
		"./configs/server.yaml",
	}

	for _, p := range paths {
		if p == "" {
			continue
		}

		data, err := os.ReadFile(p)
		if err != nil {
			continue // 文件不存在，尝试下一个
		}

		cfg := &ServerConfig{}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, ErrConfigParse
		}

		// 应用默认值
		return ApplyDefaults(cfg), nil
	}

	// 所有路径都找不到，返回默认配置
	return DefaultServerConfig(), nil
}

// LoadClient 加载客户端配置文件
// 查找顺序：指定路径 -> /etc/shareserial/client.yaml -> ./configs/client.yaml
func LoadClient(path string) (*ClientConfig, error) {
	// 尝试多个路径
	paths := []string{
		path,
		"/etc/shareserial/client.yaml",
		"./configs/client.yaml",
	}

	for _, p := range paths {
		if p == "" {
			continue
		}

		data, err := os.ReadFile(p)
		if err != nil {
			continue // 文件不存在，尝试下一个
		}

		cfg := &ClientConfig{}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, ErrConfigParse
		}

		// 应用默认值
		return ApplyClientDefaults(cfg), nil
	}

	// 所有路径都找不到，返回默认配置
	return DefaultClientConfig(), nil
}

// LoadServerFromFile 从指定文件加载服务端配置（用于测试）
func LoadServerFromFile(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &ServerConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, ErrConfigParse
	}

	return ApplyDefaults(cfg), nil
}

// LoadClientFromFile 从指定文件加载客户端配置（用于测试）
func LoadClientFromFile(path string) (*ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &ClientConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, ErrConfigParse
	}

	return ApplyClientDefaults(cfg), nil
}

// SaveServer 保存服务端配置到文件（用于生成默认配置）
func SaveServer(cfg *ServerConfig, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// SaveClient 保存客户端配置到文件
func SaveClient(cfg *ClientConfig, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
