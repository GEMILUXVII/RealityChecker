package report

import (
	"strings"

	"RealityChecker/internal/types"
)

// Formatter 报告格式化器
type Formatter struct {
	config *types.Config
}

// NewFormatter 创建格式化器
func NewFormatter(config *types.Config) *Formatter {
	return &Formatter{
		config: config,
	}
}

// FormatSingleResult 格式化单个检测结果
func (f *Formatter) FormatSingleResult(result *types.DetectionResult) string {
	var output strings.Builder

	// 使用表格格式化器，和批量检测保持一致的格式
	tableFormatter := NewTableFormatter(f.config)

	// 显示域名检测结果表格（无论适合与否）
	output.WriteString("检测结果:\n\n")
	output.WriteString(tableFormatter.FormatSuitableTable([]*types.DetectionResult{result}))
	output.WriteString("\n")

	// 如果不适合，显示不适合的原因
	if !result.Suitable || result.Error != nil {
		var unsuitableResults []*types.DetectionResult
		unsuitableResults = append(unsuitableResults, result)
		output.WriteString(tableFormatter.FormatUnsuitableSummary(unsuitableResults))
	}

	return output.String()
}
