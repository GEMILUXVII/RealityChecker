package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"RealityChecker/internal/report"
	"RealityChecker/internal/ui"
)

// executeCheck 执行单域名检测
func (r *RootCmd) executeCheck(domain string) {
	// 验证域名格式
	domain = strings.TrimSpace(domain)
	if !isValidDomain(domain) {
		ui.PrintErrorWithDetails(
			fmt.Sprintf("错误：域名格式无效 '%s'", domain),
			"提示：请检查域名格式，例如：apple.com, google.com",
			"域名要求：",
			"   - 只能包含字母、数字、连字符和点",
			"   - 不能以点开头或结尾",
			"   - 不能包含连续的点",
			"   - 长度不超过253个字符",
		)
		return
	}

	ui.PrintTimestampedMessage("开始检测域名: %s", domain)

	result, err := r.engine.CheckDomain(r.ctx, domain)
	if err != nil {
		fmt.Printf("检测失败: %v\n", err)
		return
	}

	// 使用格式化器输出结果（复用已加载的配置）
	formatter := report.NewFormatter(r.config)
	fmt.Printf("\n%s", formatter.FormatSingleResult(result))

	// 显示广告
	ui.PrintAdvertisement()
}

// isValidDomain 验证域名格式是否有效
func isValidDomain(domain string) bool {
	// 基本长度检查
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}

	// 检查是否包含非法字符
	if strings.ContainsAny(domain, " \t\n\r") {
		return false
	}

	// 检查是否以点开头或结尾
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	// 检查是否包含连续的点
	if strings.Contains(domain, "..") {
		return false
	}

	// 使用正则表达式验证域名格式
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	if !domainRegex.MatchString(domain) {
		return false
	}

	// 格式校验通过即认为有效。
	// 不再做 DNS 解析：之前的 net.LookupHost 结果被直接丢弃（两个分支都 return true），
	// 只会为每个域名平添一次阻塞 DNS 查询；真正的解析在后续检测阶段进行。
	return true
}
