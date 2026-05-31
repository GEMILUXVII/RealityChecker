package detectors

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"RealityChecker/internal/types"
)

// CDNStage CDN检测阶段
// 实际 CDN 判定已合并进 ComprehensiveTLSStage 与 RedirectStage；
// 这里仅保留基于 HTTP 响应头的检测方法，供 RedirectStage 复用。
type CDNStage struct {
	httpStrongHeader    map[string]bool
	httpMediumHeader    map[string]bool
	httpValueCdnDomains map[string]bool
}

// NewCDNStage 创建CDN检测阶段
func NewCDNStage() *CDNStage {
	stage := &CDNStage{
		httpStrongHeader:    make(map[string]bool),
		httpMediumHeader:    make(map[string]bool),
		httpValueCdnDomains: make(map[string]bool),
	}
	stage.loadCDNKeywords()
	return stage
}

// Execute CDN检测已合并到 ComprehensiveTLSStage，此处为空实现以满足 DetectionStage 接口
func (cs *CDNStage) Execute(ctx *types.PipelineContext) error {
	return nil
}

// checkHTTPStrongHeader 检查HTTP强响应头特征
func (cs *CDNStage) checkHTTPStrongHeader(networkResult *types.NetworkResult) (string, string) {
	if networkResult == nil || networkResult.Headers == nil {
		return "", ""
	}

	// 检查HTTP强响应头
	for headerName := range cs.httpStrongHeader {
		// 检查header名称
		for respHeaderName, respHeaderValue := range networkResult.Headers {
			if strings.EqualFold(respHeaderName, headerName) {
				provider := cs.getProviderFromHeader(headerName)
				return provider, fmt.Sprintf("HTTP强响应头特征: %s=%s", respHeaderName, respHeaderValue)
			}
		}

		// 检查server头（特殊处理）
		if strings.HasPrefix(headerName, "server: ") {
			serverValue := strings.TrimPrefix(headerName, "server: ")
			if serverHeader, exists := networkResult.Headers["Server"]; exists {
				if strings.Contains(strings.ToLower(serverHeader), strings.ToLower(serverValue)) {
					provider := cs.getProviderFromHeader(headerName)
					return provider, fmt.Sprintf("HTTP强响应头特征: Server=%s", serverHeader)
				}
			}
		}
	}

	return "", ""
}

// checkHTTPValueCdnDomains 检查HTTP头值中的CDN域名
func (cs *CDNStage) checkHTTPValueCdnDomains(networkResult *types.NetworkResult) (string, string) {
	if networkResult == nil || networkResult.Headers == nil {
		return "", ""
	}

	// 检查HTTP响应头值中是否包含CDN域名
	for headerName, respHeaderValue := range networkResult.Headers {
		respHeaderValueLower := strings.ToLower(respHeaderValue)

		for cdnDomain := range cs.httpValueCdnDomains {
			// 移除注释部分
			cleanDomain := strings.Split(cdnDomain, "#")[0]
			cleanDomain = strings.TrimSpace(cleanDomain)

			if strings.Contains(respHeaderValueLower, strings.ToLower(cleanDomain)) {
				provider := cs.getProviderFromHeader(cdnDomain)
				return provider, fmt.Sprintf("HTTP头值CDN域名特征: %s包含%s", headerName, cleanDomain)
			}
		}
	}

	return "", ""
}

// checkHTTPMediumHeader 检查HTTP中等响应头
func (cs *CDNStage) checkHTTPMediumHeader(networkResult *types.NetworkResult) (string, string) {
	if networkResult == nil || networkResult.Headers == nil {
		return "", ""
	}

	// 检查HTTP中等响应头
	for headerName := range cs.httpMediumHeader {
		for respHeaderName, respHeaderValue := range networkResult.Headers {
			if strings.EqualFold(respHeaderName, headerName) {
				provider := cs.getProviderFromHeader(headerName)
				return provider, fmt.Sprintf("HTTP中等响应头特征: %s=%s", respHeaderName, respHeaderValue)
			}
		}
	}

	return "", ""
}

// getProviderFromHeader 根据HTTP头获取CDN提供商
func (cs *CDNStage) getProviderFromHeader(header string) string {
	// 不使用硬编码，具体的CDN提供商信息由关键字库提供
	return "CDN"
}

// loadCDNKeywords 加载CDN关键词（仅加载 HTTP 响应头相关的小节）
func (cs *CDNStage) loadCDNKeywords() {
	file, err := os.Open("data/cdn_keywords.txt")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查是否是节标题
		if strings.HasSuffix(line, ":") {
			currentSection = line
			continue
		}

		// 根据节标题分类加载
		switch currentSection {
		case "http_strong_header:":
			cs.httpStrongHeader[line] = true
		case "http_medium_header:":
			cs.httpMediumHeader[line] = true
		case "http_value_cdn_domains:":
			cs.httpValueCdnDomains[line] = true
		}
	}
}

// CanEarlyExit 是否可以早期退出
func (cs *CDNStage) CanEarlyExit() bool {
	return false
}

// Priority 优先级
func (cs *CDNStage) Priority() int {
	return 9 // CDN检测（已合并进综合TLS阶段，未单独加入流水线）
}

// Name 阶段名称
func (cs *CDNStage) Name() string {
	return "cdn"
}
