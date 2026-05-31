package core

import (
	"testing"

	"RealityChecker/internal/types"
)

func fullySuitableResult() *types.DetectionResult {
	return &types.DetectionResult{
		Domain:  "example.com",
		Network: &types.NetworkResult{Accessible: true, StatusCode: 200},
		TLS:     &types.TLSResult{SupportsTLS13: true, SupportsX25519: true, SupportsHTTP2: true},
		Certificate: &types.CertificateResult{Valid: true, DaysUntilExpiry: 90},
		SNI:     &types.SNIResult{SupportsSNI: true, SNIMatch: true},
	}
}

func TestEvaluateSuitability(t *testing.T) {
	p := NewPipeline(nil, &types.Config{})

	t.Run("fully valid is suitable and clears error", func(t *testing.T) {
		r := fullySuitableResult()
		r.Error = nil
		p.evaluateSuitability(r)
		if !r.Suitable || !r.HardRequirementsMet || r.Error != nil {
			t.Fatalf("expected suitable with no error, got suitable=%v hard=%v err=%v", r.Suitable, r.HardRequirementsMet, r.Error)
		}
	})

	t.Run("nil TLS after reachable redirect is NOT suitable", func(t *testing.T) {
		// 复现 bug：网络可达、状态码安全，但 TLS 检测未执行（早退）
		r := &types.DetectionResult{
			Domain:  "example.com",
			Network: &types.NetworkResult{Accessible: true, StatusCode: 200},
			// TLS / Certificate / SNI 均为 nil
		}
		p.evaluateSuitability(r)
		if r.Suitable {
			t.Fatal("domain with no TLS result must not be marked suitable")
		}
	})

	t.Run("blocked is not suitable", func(t *testing.T) {
		r := fullySuitableResult()
		r.Blocked = &types.BlockedResult{IsBlocked: true}
		p.evaluateSuitability(r)
		if r.Suitable {
			t.Fatal("blocked domain must not be suitable")
		}
	})

	t.Run("domestic is not suitable", func(t *testing.T) {
		r := fullySuitableResult()
		r.Location = &types.LocationResult{IsDomestic: true}
		p.evaluateSuitability(r)
		if r.Suitable {
			t.Fatal("domestic domain must not be suitable")
		}
	})

	t.Run("missing certificate is not suitable", func(t *testing.T) {
		r := fullySuitableResult()
		r.Certificate = nil
		p.evaluateSuitability(r)
		if r.Suitable {
			t.Fatal("domain without certificate result must not be suitable")
		}
	})

	t.Run("SNI mismatch is not suitable", func(t *testing.T) {
		r := fullySuitableResult()
		r.SNI = &types.SNIResult{SupportsSNI: true, SNIMatch: false}
		p.evaluateSuitability(r)
		if r.Suitable {
			t.Fatal("SNI mismatch must not be suitable")
		}
	})
}
