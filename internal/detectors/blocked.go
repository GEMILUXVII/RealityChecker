package detectors

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"RealityChecker/internal/types"
)

// BlockedStage 被墙检测阶段
type BlockedStage struct {
	gfwlist map[string]bool
}

// NewBlockedStage 创建被墙检测阶段
func NewBlockedStage() *BlockedStage {
	stage := &BlockedStage{
		gfwlist: make(map[string]bool),
	}
	stage.loadGFWList()
	return stage
}

// Execute 执行被墙检测
func (bs *BlockedStage) Execute(ctx *types.PipelineContext) error {
	// 检查是否被墙
	isBlocked, reason := bs.checkBlocked(ctx.Domain)

	ctx.Result.Blocked = &types.BlockedResult{
		IsBlocked:      isBlocked,
		BlockedReasons: []string{reason},
		MatchType:      "gfwlist",
	}

	if isBlocked {
		ctx.EarlyExit = true
		return fmt.Errorf("域名被墙（%s）", reason)
	}
	return nil
}

// checkBlocked 检查是否被墙
func (bs *BlockedStage) checkBlocked(domain string) (bool, string) {
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))

	// 精确匹配
	if bs.gfwlist[domain] {
		return true, "仅参考黑名单"
	}

	// 父域匹配：黑名单条目以裸域名形式存储（loadGFWList 已剥离 "+." 前缀），
	// 因此应逐级用父域名查找，使 www.google.com 能命中条目 google.com。
	// 从去掉最左标签开始，到最后一个单标签（TLD）之前为止，避免按裸 TLD 误匹配。
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts)-1; i++ {
		parent := strings.Join(parts[i:], ".")
		if bs.gfwlist[parent] {
			return true, fmt.Sprintf("父域匹配: %s", parent)
		}
	}

	return false, ""
}

// loadGFWList 加载GFWList
func (bs *BlockedStage) loadGFWList() {
	file, err := os.Open("data/gfwlist.conf")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inPayload := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "payload:" {
			inPayload = true
			continue
		}

		if !inPayload {
			continue
		}

		if strings.HasPrefix(line, "- '") && strings.HasSuffix(line, "'") {
			domain := strings.TrimPrefix(line, "- '")
			domain = strings.TrimSuffix(domain, "'")

			domain = strings.TrimPrefix(domain, "+.")

			if domain != "" {
				bs.gfwlist[domain] = true
			}
		}
	}
}

// CanEarlyExit 是否可以早期退出
func (bs *BlockedStage) CanEarlyExit() bool {
	return true
}

// Priority 优先级
func (bs *BlockedStage) Priority() int {
	return 1 // 被墙检测优先级最高
}

// Name 阶段名称
func (bs *BlockedStage) Name() string {
	return "blocked"
}
