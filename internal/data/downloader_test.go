package data

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 下载源不可达时：已存在但过期的文件应被保留并继续使用，而不是导致启动失败。
func TestEnsureFileKeepsStaleFileWhenDownloadFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	// 把修改时间设为 4 天前 → 视为过期
	old := time.Now().Add(-4 * 24 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}

	d := &Downloader{timeout: 2 * time.Second, retries: 1, retryDelay: 0}
	// 指向一个会立即连接失败的地址（本地保留端口），不触发外部网络
	file := DataFile{Name: "data.txt", URL: "http://127.0.0.1:1/nope", LocalPath: path}

	if err := d.ensureFile(file); err != nil {
		t.Fatalf("过期文件 + 下载失败不应返回错误，得到: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("现有文件应被保留: %v", err)
	}
	if string(b) != "existing" {
		t.Fatalf("现有文件内容应保持不变，得到 %q", string(b))
	}
}

// 文件完全缺失且下载失败时仍应报错（缺少数据无法工作）。
func TestEnsureFileMissingAndDownloadFailsReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.txt")

	d := &Downloader{timeout: 2 * time.Second, retries: 1, retryDelay: 0}
	file := DataFile{Name: "missing.txt", URL: "http://127.0.0.1:1/nope", LocalPath: path}

	if err := d.ensureFile(file); err == nil {
		t.Fatal("文件缺失且下载失败时应返回错误")
	}
}
