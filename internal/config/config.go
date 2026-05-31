package config

import (
	"fmt"
	"os"
	"time"

	"RealityChecker/internal/types"

	"gopkg.in/yaml.v3"
)

// LoadConfig 加载配置
func LoadConfig(configPath string) (*types.Config, error) {
	// 获取默认配置
	config := getDefaultConfig()

	// 如果提供了配置文件路径，尝试加载
	if configPath != "" {
		if err := loadConfigFromFile(config, configPath); err != nil {
			return nil, fmt.Errorf("加载配置文件失败: %v", err)
		}
	} else {
		// 尝试从默认位置加载配置文件
		defaultPaths := []string{
			"config.yaml",
			"config.yml",
			"./config.yaml",
			"./config.yml",
		}

		for _, path := range defaultPaths {
			if _, err := os.Stat(path); err == nil {
				if err := loadConfigFromFile(config, path); err == nil {
					break // 成功加载，跳出循环
				}
			}
		}
	}

	// 验证并设置默认值
	validateAndSetDefaults(config)
	return config, nil
}

// loadConfigFromFile 从文件加载配置
func loadConfigFromFile(config *types.Config, filePath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在: %s", filePath)
	}

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 直接把文件内容反序列化到已填入默认值的配置上。
	// yaml.v3 只会覆盖文件中实际出现的字段，未出现的字段保留默认值，
	// 从而正确区分"未设置"与"显式零值"，避免布尔项、Retries 等默认值被零值覆盖。
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	return nil
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() *types.Config {
	return &types.Config{
		Network: types.NetworkConfig{
			Timeout:    3 * time.Second, // 减少到3秒
			Retries:    1,
			DNSServers: []string{"8.8.8.8", "1.1.1.1"},
		},
		TLS: types.TLSConfig{
			MinVersion: 771, // TLS 1.2
			MaxVersion: 772, // TLS 1.3
		},
		Concurrency: types.ConcurrencyConfig{
			MaxConcurrent: 8,
			CheckTimeout:  3 * time.Second, // 减少到3秒
			CacheTTL:      5 * time.Minute,
		},
		Output: types.OutputConfig{
			Color:   true,
			Verbose: false,
			Format:  "table",
		},
		Cache: types.CacheConfig{
			DNSEnabled:    true,
			ResultEnabled: true,
			TTL:           5 * time.Minute,
			MaxSize:       1000,
		},
		Batch: types.BatchConfig{
			StreamOutput: false,
			ProgressBar:  true,
			ReportFormat: "text",
			Timeout:      30 * time.Second,
		},
	}
}

// validateAndSetDefaults 验证配置并设置默认值
func validateAndSetDefaults(config *types.Config) {
	// 网络配置验证
	if config.Network.Timeout <= 0 {
		config.Network.Timeout = 30 * time.Second
	}
	if config.Network.Retries < 0 {
		config.Network.Retries = 3
	}
	if len(config.Network.DNSServers) == 0 {
		config.Network.DNSServers = []string{"8.8.8.8", "1.1.1.1"}
	}

	// TLS配置验证
	if config.TLS.MinVersion == 0 {
		config.TLS.MinVersion = 771 // TLS 1.2
	}
	if config.TLS.MaxVersion == 0 {
		config.TLS.MaxVersion = 772 // TLS 1.3
	}

	// 并发配置验证
	if config.Concurrency.MaxConcurrent <= 0 {
		config.Concurrency.MaxConcurrent = 8
	}
	if config.Concurrency.CheckTimeout <= 0 {
		config.Concurrency.CheckTimeout = 30 * time.Second
	}
	if config.Concurrency.CacheTTL <= 0 {
		config.Concurrency.CacheTTL = 5 * time.Minute
	}

	// 输出配置验证
	if config.Output.Format == "" {
		config.Output.Format = "table"
	}

	// 缓存配置验证
	if config.Cache.TTL <= 0 {
		config.Cache.TTL = 5 * time.Minute
	}
	if config.Cache.MaxSize <= 0 {
		config.Cache.MaxSize = 1000
	}

	// 批量配置验证
	if config.Batch.ReportFormat == "" {
		config.Batch.ReportFormat = "text"
	}
	if config.Batch.Timeout <= 0 {
		config.Batch.Timeout = 60 * time.Second
	}
}
