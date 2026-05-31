package config

import (
	"os"
	"path/filepath"
	"testing"
)

// 配置文件只设置部分字段时，未出现的字段必须保留默认值，
// 不能被零值覆盖（旧 mergeConfig 会把 color/dns_enabled 等强制设为 false）。
func TestLoadConfigKeepsDefaultsForUnsetFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// 仅设置 network.retries，其余字段一概不出现
	if err := os.WriteFile(path, []byte("network:\n  retries: 5\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	// 文件中出现的字段应被应用
	if cfg.Network.Retries != 5 {
		t.Errorf("Network.Retries = %d, want 5", cfg.Network.Retries)
	}

	// 以下默认值不应被未出现的键覆盖
	if !cfg.Output.Color {
		t.Error("Output.Color should remain default true when absent from file")
	}
	if !cfg.Cache.DNSEnabled {
		t.Error("Cache.DNSEnabled should remain default true when absent from file")
	}
	if len(cfg.Network.DNSServers) == 0 {
		t.Error("Network.DNSServers should remain default when absent from file")
	}
	if cfg.Output.Format != "table" {
		t.Errorf("Output.Format = %q, want default \"table\"", cfg.Output.Format)
	}
}
