package detectors

import "testing"

func TestCheckBlockedMatchesSubdomains(t *testing.T) {
	bs := &BlockedStage{gfwlist: map[string]bool{
		"google.com":      true,
		"sub.example.org": true,
	}}

	cases := []struct {
		domain string
		want   bool
	}{
		{"google.com", true},         // 精确匹配
		{"www.google.com", true},     // 子域名（修复前漏判）
		{"a.b.google.com", true},     // 多级子域名
		{"GOOGLE.COM", true},         // 大小写不敏感
		{"www.google.com.", true},    // 末尾点
		{"sub.example.org", true},    // 精确（多标签条目）
		{"x.sub.example.org", true},  // 多标签条目的子域名
		{"notgoogle.com", false},     // 非子域名
		{"google.com.evil.com", false}, // 后缀欺骗不应命中
		{"example.org", false},       // 仅 sub.example.org 在表中
		{"bing.com", false},
	}

	for _, tc := range cases {
		if got, _ := bs.checkBlocked(tc.domain); got != tc.want {
			t.Errorf("checkBlocked(%q) = %v, want %v", tc.domain, got, tc.want)
		}
	}
}
