package batch

import (
	"testing"
	"time"
)

func TestBatchTimeoutScalesWithDomainCount(t *testing.T) {
	// 单域名（任意并发）应至少为缓冲 + 一个批次的预算
	small := batchTimeout(1, 8)
	if small != 12*time.Second+30*time.Second {
		t.Fatalf("batchTimeout(1, 8) = %v, want 42s", small)
	}

	// 100 个域名、并发 8 → 13 批 → 远大于原先固定的 15s
	large := batchTimeout(100, 8)
	if large <= 15*time.Second {
		t.Fatalf("batchTimeout(100, 8) = %v, want >> 15s", large)
	}
	if large <= small {
		t.Fatalf("expected timeout to grow with domain count: small=%v large=%v", small, large)
	}

	// 并发为非法值时回退到默认并发，不应 panic 或返回 0
	if got := batchTimeout(50, 0); got <= 0 {
		t.Fatalf("batchTimeout(50, 0) = %v, want > 0", got)
	}

	// 域名数为 0 时至少返回一个批次的时间
	if got := batchTimeout(0, 8); got < 30*time.Second {
		t.Fatalf("batchTimeout(0, 8) = %v, want >= baseBuffer", got)
	}
}
