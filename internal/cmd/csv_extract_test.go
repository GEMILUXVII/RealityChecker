package cmd

import (
	"reflect"
	"testing"
)

func TestExtractDomainsFromCSV(t *testing.T) {
	cases := []struct {
		name    string
		records [][]string
		want    []string
	}{
		{
			name: "old format (CERT_DOMAIN at index 2)",
			records: [][]string{
				{"IP", "ORIGIN", "CERT_DOMAIN", "CERT_ISSUER", "GEO_CODE"},
				{"1.1.1.1", "1.1.1.1", "example.com", "Let's Encrypt", "HK"},
			},
			want: []string{"example.com"},
		},
		{
			name: "new format (CERT_DOMAIN at index 8)",
			records: [][]string{
				{"IP", "ORIGIN", "TLS", "ALPN", "CURVE", "CERT_LENGTH", "CERT_SIGNATURE", "CERT_PUBLICKEY", "CERT_DOMAIN", "CERT_ISSUER", "GEO_CODE"},
				{"1.1.1.1", "1.1.1.1", "TLS 1.3", "h2", "X25519", "4096(certs count: 2)", "SHA256-RSA", "RSA", "example.com", "Let's Encrypt", "HK"},
			},
			want: []string{"example.com"},
		},
		{
			name: "no recognizable header falls back to index 2",
			records: [][]string{
				{"a", "b", "c"},
				{"1.1.1.1", "1.1.1.1", "fallback.com"},
			},
			want: []string{"fallback.com"},
		},
		{
			name: "new format dedups and skips wildcards",
			records: [][]string{
				{"IP", "ORIGIN", "TLS", "ALPN", "CURVE", "CERT_LENGTH", "CERT_SIGNATURE", "CERT_PUBLICKEY", "CERT_DOMAIN", "CERT_ISSUER", "GEO_CODE"},
				{"1.1.1.1", "1.1.1.1", "TLS 1.3", "h2", "X25519", "x", "y", "z", "dup.com", "CA", "HK"},
				{"2.2.2.2", "2.2.2.2", "TLS 1.3", "h2", "X25519", "x", "y", "z", "dup.com", "CA", "HK"},
				{"3.3.3.3", "3.3.3.3", "TLS 1.3", "h2", "X25519", "x", "y", "z", "*.wildcard.com", "CA", "HK"},
			},
			want: []string{"dup.com"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractDomainsFromCSV(tc.records)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("extractDomainsFromCSV() = %v, want %v", got, tc.want)
			}
		})
	}
}
