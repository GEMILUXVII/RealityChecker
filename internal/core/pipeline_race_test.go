package core

import (
	"context"
	"testing"

	"RealityChecker/internal/types"
)

// TestExecuteNetworkStagesNoRace 验证网络检测阶段串行执行：
// 这两个阶段（综合TLS、热门网站）会写入同一个 Result.CDN。旧实现并发执行会触发
// 数据竞争（go test -race 必报），改为串行后应无竞争，且热门网站阶段写入的 CDN 结果
// 不会丢失。Connections 为 nil 时 TLS 检测走无网络 I/O 的回退分支。
func TestExecuteNetworkStagesNoRace(t *testing.T) {
	p := NewPipeline(nil, &types.Config{})

	var networkStages []types.DetectionStage
	for _, s := range p.GetStages() {
		if !s.CanEarlyExit() {
			networkStages = append(networkStages, s)
		}
	}
	if len(networkStages) < 2 {
		t.Fatalf("expected at least 2 network stages, got %d", len(networkStages))
	}

	for i := 0; i < 100; i++ {
		pctx := &types.PipelineContext{
			Domain:      "example.com",
			Result:      &types.DetectionResult{Domain: "example.com"},
			Connections: nil,
			Config:      &types.Config{},
		}
		p.executeNetworkStages(context.Background(), pctx, networkStages)

		// 热门网站阶段一定会写入 CDN（用于承载 IsHotWebsite），串行后不应丢失
		if pctx.Result.CDN == nil {
			t.Fatal("expected Result.CDN to be set after network stages")
		}
	}
}
