package detectors

import (
	"fmt"
	"net"

	"RealityChecker/internal/types"

	"github.com/oschwald/geoip2-golang"
)

// LocationStage 地理位置检测阶段
type LocationStage struct {
	geoipDB *geoip2.Reader
}

// NewLocationStage 创建地理位置检测阶段
func NewLocationStage() *LocationStage {
	stage := &LocationStage{}
	stage.loadGeoIPDatabase()
	return stage
}

// Execute 执行地理位置检测
func (ls *LocationStage) Execute(ctx *types.PipelineContext) error {

	// 复用 IP 解析阶段（优先级更高，已先执行）解析出的 IP，避免重复 DNS 查询；
	// 仅在尚未解析时才自行解析。
	var ip string
	if ctx.Result.Location != nil && ctx.Result.Location.IPAddress != "" {
		ip = ctx.Result.Location.IPAddress
	} else {
		resolved, err := ls.resolveIP(ctx.Domain)
		if err != nil {
			return fmt.Errorf("IP解析失败: %v", err)
		}
		ip = resolved
	}

	// 获取地理位置
	country, isDomestic := ls.getLocation(ip)

	ctx.Result.Location = &types.LocationResult{
		Country:    country,
		IsDomestic: isDomestic,
		IPAddress:  ip,
	}

	if isDomestic {
		ctx.EarlyExit = true
		return fmt.Errorf("国内网站（仅参考GeoIP）")
	}

	return nil
}

// resolveIP 解析IP地址
func (ls *LocationStage) resolveIP(domain string) (string, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return "", err
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("未找到IP地址")
	}

	// 优先选择IPv4地址
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String(), nil
		}
	}

	return ips[0].String(), nil
}

// getLocation 获取地理位置
func (ls *LocationStage) getLocation(ip string) (string, bool) {
	// 注意：这里传入的是IP地址，不是域名，所以不需要检查域名特征

	// 使用GeoIP数据库
	if ls.geoipDB != nil {
		record, err := ls.geoipDB.Country(net.ParseIP(ip))
		if err == nil {
			country := record.Country.Names["zh-CN"]
			if country == "" {
				country = record.Country.Names["en"]
				if country == "" {
					country = record.Country.IsoCode
				}
			}

			isDomestic := (country == "中国" || country == "CN")
			return country, isDomestic
		}
	}

	return "未知", false
}

// loadGeoIPDatabase 加载GeoIP数据库
func (ls *LocationStage) loadGeoIPDatabase() {
	db, err := geoip2.Open("data/Country.mmdb")
	if err != nil {
		return
	}
	ls.geoipDB = db
}

// CanEarlyExit 是否可以早期退出
func (ls *LocationStage) CanEarlyExit() bool {
	return true
}

// Priority 优先级
func (ls *LocationStage) Priority() int {
	return 5 // 地理位置检测（在IP解析之后）
}

// Name 阶段名称
func (ls *LocationStage) Name() string {
	return "location"
}
