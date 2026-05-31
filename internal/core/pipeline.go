package core

import (
	"context"
	"fmt"
	"sort"
	"time"

	"RealityChecker/internal/detectors"
	"RealityChecker/internal/network"
	"RealityChecker/internal/types"
)

// Pipeline 检测流水线
type Pipeline struct {
	stages      []types.DetectionStage
	config      *types.Config
	earlyExit   bool
	connections *network.ConnectionManager
}

// NewPipeline 创建新的检测流水线
func NewPipeline(connections *network.ConnectionManager, config *types.Config) *Pipeline {
	pipeline := &Pipeline{
		config:      config,
		earlyExit:   true,
		connections: connections,
	}

	// 初始化检测阶段
	pipeline.initializeStages()

	return pipeline
}

// initializeStages 初始化检测阶段
func (p *Pipeline) initializeStages() {
	p.stages = []types.DetectionStage{
		detectors.NewBlockedStage(),          // 1. 被墙检测 (最高优先级)
		detectors.NewRedirectStage(),         // 2. 重定向检测
		detectors.NewStatusCheckStage(),      // 3. 状态码检查
		detectors.NewIPResolverStage(),       // 4. IP解析
		detectors.NewLocationStage(),         // 5. 地理位置检测
		detectors.NewLocationCheckStage(),    // 6. 地理位置检查
		detectors.NewComprehensiveTLSStage(), // 7. 综合TLS检测 (TLS1.3、X25519、H2、SNI、证书、CDN)
		detectors.NewHotWebsiteStage(),       // 8. 热门网站检测
	}

	// 按优先级排序
	sort.Slice(p.stages, func(i, j int) bool {
		return p.stages[i].Priority() < p.stages[j].Priority()
	})
}

// Execute 执行检测流水线
func (p *Pipeline) Execute(ctx context.Context, domain string) (*types.DetectionResult, error) {
	startTime := time.Now()

	// 创建流水线上下文
	pipelineCtx := &types.PipelineContext{
		Domain:      domain,
		StartTime:   startTime,
		Result:      &types.DetectionResult{Domain: domain, StartTime: startTime},
		Connections: p.connections, // 传递连接管理器给检测器
		Cache:       nil,           // 缓存管理器已移除
		Config:      p.config,
		Context:     ctx, // 传递原始context
		EarlyExit:   false,
	}

	// 并发执行检测阶段，提高检测效率
	p.executeStagesConcurrently(ctx, pipelineCtx)

	// 计算总耗时
	pipelineCtx.Result.Duration = time.Since(startTime)

	// 评估适合性
	p.evaluateSuitability(pipelineCtx.Result)

	return pipelineCtx.Result, nil
}

// executeStagesConcurrently 并发执行检测阶段
func (p *Pipeline) executeStagesConcurrently(ctx context.Context, pipelineCtx *types.PipelineContext) {
	// 将检测阶段分为两组：阻塞检测和网络检测
	var blockingStages []types.DetectionStage
	var networkStages []types.DetectionStage

	for _, stage := range p.stages {
		if stage.CanEarlyExit() {
			blockingStages = append(blockingStages, stage)
		} else {
			networkStages = append(networkStages, stage)
		}
	}

	// 先执行阻塞检测（被墙检测、地理位置检测等）
	for _, stage := range blockingStages {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := stage.Execute(pipelineCtx); err != nil {
			pipelineCtx.Result.Error = err
			if stage.CanEarlyExit() {
				pipelineCtx.EarlyExit = true
				return
			}
		}

		// 检查是否需要早期退出
		if p.earlyExit && stage.CanEarlyExit() && pipelineCtx.EarlyExit {
			pipelineCtx.Result.EarlyExit = true
			return
		}
	}

	// 如果被阻塞，直接返回
	if pipelineCtx.EarlyExit {
		return
	}

	// 顺序执行网络检测阶段
	if len(networkStages) > 0 {
		p.executeNetworkStages(ctx, pipelineCtx, networkStages)
	}
}

// executeNetworkStages 按优先级顺序串行执行网络检测阶段。
// 这些阶段（综合TLS、热门网站）会写入同一个 Result 上的共享字段（尤其是 CDN），
// 且热门网站检测依赖 TLS 阶段产出的 CDN 结果（优先级更低、应在其之后），
// 因此必须串行执行：既消除并发写 Result 造成的数据竞争，又保证依赖顺序正确。
// 网络阶段数量很少且其中仅 TLS 涉及网络 I/O，串行几乎不影响整体耗时。
func (p *Pipeline) executeNetworkStages(ctx context.Context, pipelineCtx *types.PipelineContext, stages []types.DetectionStage) {
	for _, stage := range stages {
		select {
		case <-ctx.Done():
			return
		default:
		}

		s := stage
		func() {
			defer func() {
				if r := recover(); r != nil {
					pipelineCtx.Result.Error = fmt.Errorf("检测阶段 %s panic: %v", s.Name(), r)
				}
			}()

			if err := s.Execute(pipelineCtx); err != nil {
				pipelineCtx.Result.Error = err
			}
		}()
	}
}

// evaluateSuitability 评估适合性
func (p *Pipeline) evaluateSuitability(result *types.DetectionResult) {
	// 检查硬性条件
	if result.Blocked != nil && result.Blocked.IsBlocked {
		result.Suitable = false
		result.Error = fmt.Errorf("域名被墙")
		return
	}

	if result.Location != nil && result.Location.IsDomestic {
		result.Suitable = false
		result.Error = fmt.Errorf("国内网站")
		return
	}

	if result.Network != nil && !result.Network.Accessible {
		result.Suitable = false
		result.Error = fmt.Errorf("网络不可达")
		result.StatusCodeCategory = types.StatusCodeCategoryNetwork
		return
	}

	// 检查状态码是否安全
	if result.Network != nil && result.Network.Accessible {
		statusCodeCategory := types.ClassifyStatusCode(result.Network.StatusCode, true)
		result.StatusCodeCategory = statusCodeCategory

		// 如果状态码不安全，标记为不适合
		if statusCodeCategory == types.StatusCodeCategoryExcluded {
			result.Suitable = false
			result.Error = fmt.Errorf("状态码不自然: %d", result.Network.StatusCode)
			return
		}
	}

	// TLS 相关硬性条件必须有实际检测结果才能判定为适合。
	// 若因早期退出或网络阶段异常导致未执行 TLS 检测（result.TLS 为 nil），
	// 不能仅因为"没有检测到失败"就判为适合，否则会产生假阳性。
	if result.TLS == nil {
		result.Suitable = false
		if result.Error == nil {
			result.Error = fmt.Errorf("未完成TLS检测")
		}
		return
	}
	if !result.TLS.SupportsTLS13 {
		result.Suitable = false
		result.Error = fmt.Errorf("不支持TLS 1.3")
		return
	}
	if !result.TLS.SupportsHTTP2 {
		result.Suitable = false
		result.Error = fmt.Errorf("不支持HTTP/2")
		return
	}

	if result.Certificate == nil || !result.Certificate.Valid {
		result.Suitable = false
		result.Error = fmt.Errorf("证书无效")
		return
	}
	// 只有真正过期的证书才标记为不适合（天数小于等于0）
	if result.Certificate.DaysUntilExpiry <= 0 {
		result.Suitable = false
		result.Error = fmt.Errorf("证书已过期（%d天）", result.Certificate.DaysUntilExpiry)
		return
	}

	if result.SNI == nil || !result.SNI.SupportsSNI || !result.SNI.SNIMatch {
		result.Suitable = false
		result.Error = fmt.Errorf("SNI不匹配")
		return
	}

	// X25519 探测仅在上述硬性条件均满足时才会真正执行（否则被跳过、值保持 false）。
	// 因此放在最后判断，避免在证书/SNI 等先失败时被误报为"不支持X25519"。
	if !result.TLS.SupportsX25519 {
		result.Suitable = false
		result.Error = fmt.Errorf("不支持X25519密钥交换")
		return
	}

	// 所有硬性条件都符合，清除早期阶段可能残留的非致命错误
	result.Suitable = true
	result.HardRequirementsMet = true
	result.Error = nil
}

// SetEarlyExit 设置是否早期退出
func (p *Pipeline) SetEarlyExit(earlyExit bool) {
	p.earlyExit = earlyExit
}

// GetStages 获取检测阶段
func (p *Pipeline) GetStages() []types.DetectionStage {
	return p.stages
}

// AddStage 添加检测阶段
func (p *Pipeline) AddStage(stage types.DetectionStage) {
	p.stages = append(p.stages, stage)
	sort.Slice(p.stages, func(i, j int) bool {
		return p.stages[i].Priority() < p.stages[j].Priority()
	})
}

// RemoveStage 移除检测阶段
func (p *Pipeline) RemoveStage(name string) {
	var newStages []types.DetectionStage
	for _, stage := range p.stages {
		if stage.Name() != name {
			newStages = append(newStages, stage)
		}
	}
	p.stages = newStages
}
