package detectors

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"
)

func makeLeafCert(t *testing.T, commonName string, dnsNames []string, notBefore, notAfter time.Time) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		Issuer:       pkix.Name{CommonName: "Test Self-Signed CA"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		DNSNames:     dnsNames,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func TestEvaluateCertificate(t *testing.T) {
	now := time.Now()
	cert := makeLeafCert(t, "example.com", []string{"example.com"}, now.Add(-time.Hour), now.Add(90*24*time.Hour))

	t.Run("hostname match reported, self-signed is invalid", func(t *testing.T) {
		res, sniMatch := evaluateCertificate([]*x509.Certificate{cert}, "example.com", now)
		if res == nil {
			t.Fatal("expected non-nil certificate result")
		}
		if !sniMatch {
			t.Error("expected hostname to match SAN example.com")
		}
		// 自签名证书不被系统根信任 → Valid 必为 false
		if res.Valid {
			t.Error("self-signed certificate must not be valid")
		}
		if res.Subject == "" || res.NotAfter.IsZero() {
			t.Error("expected issuer/subject/validity fields to be populated")
		}
	})

	t.Run("hostname mismatch", func(t *testing.T) {
		_, sniMatch := evaluateCertificate([]*x509.Certificate{cert}, "other.com", now)
		if sniMatch {
			t.Error("expected hostname mismatch for other.com")
		}
	})

	t.Run("empty chain", func(t *testing.T) {
		res, sniMatch := evaluateCertificate(nil, "example.com", now)
		if res != nil || sniMatch {
			t.Error("expected nil result and no match for empty chain")
		}
	})
}
