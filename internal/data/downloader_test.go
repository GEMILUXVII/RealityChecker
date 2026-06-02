package data

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 文件已存在时不再触发任何下载（数据刷新交由 CI 的 data-latest release），
// 即使文件较旧也保持原样、不被覆盖。
func TestEnsureFileSkipsExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	// 即使把修改时间设为 30 天前，已存在的文件也不应被重新下载
	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}

	d := &Downloader{timeout: 2 * time.Second, retries: 1, retryDelay: 0}
	// URL 指向会立即失败的地址：若被错误地触发下载会报错，但文件已存在就不该触发
	file := DataFile{Name: "data.txt", URL: "http://127.0.0.1:1/nope", LocalPath: path}

	if err := d.ensureFile(file); err != nil {
		t.Fatalf("已存在的文件不应触发下载，得到错误: %v", err)
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
