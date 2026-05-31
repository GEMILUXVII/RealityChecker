package cmd

import (
	"strings"
	"testing"
)

func TestIsValidDomain(t *testing.T) {
	cases := []struct {
		domain string
		want   bool
	}{
		{"apple.com", true},
		{"a.b.example.co.uk", true},
		{"xn--abc.com", true},
		{"example123.io", true},
		{"", false},
		{"exa mple.com", false},     // 空格
		{".apple.com", false},       // 以点开头
		{"apple.com.", false},       // 以点结尾
		{"ap..ple.com", false},      // 连续点
		{"-apple.com", false},       // 标签以连字符开头
		{"apple-.com", false},       // 标签以连字符结尾
		{strings.Repeat("a", 254), false}, // 超长
	}

	for _, tc := range cases {
		if got := isValidDomain(tc.domain); got != tc.want {
			t.Errorf("isValidDomain(%q) = %v, want %v", tc.domain, got, tc.want)
		}
	}
}
