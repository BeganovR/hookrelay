package service_test

import (
	"testing"
	"time"

	"hookrelay/internal/domain"
	"hookrelay/internal/service"
)

func TestCalcBackoff_Exponential(t *testing.T) {
	base := 5
	tests := []struct {
		attempt int
		wantMin time.Duration
		wantMax time.Duration
	}{
		{0, 4 * time.Second, 7 * time.Second},
		{1, 8 * time.Second, 12 * time.Second},
		{2, 16 * time.Second, 24 * time.Second},
	}
	for _, tc := range tests {
		d := service.CalcBackoff(domain.RetryExponential, base, tc.attempt)
		if d < tc.wantMin || d > tc.wantMax {
			t.Errorf("attempt=%d: CalcBackoff=%v, want [%v, %v]", tc.attempt, d, tc.wantMin, tc.wantMax)
		}
	}
}

func TestCalcBackoff_Linear(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 5 * time.Second},
		{1, 10 * time.Second},
		{2, 15 * time.Second},
		{4, 25 * time.Second},
	}
	for _, tc := range tests {
		d := service.CalcBackoff(domain.RetryLinear, 5, tc.attempt)
		if d != tc.want {
			t.Errorf("attempt=%d: CalcBackoff=%v, want %v", tc.attempt, d, tc.want)
		}
	}
}

func TestCalcBackoff_Constant(t *testing.T) {
	for attempt := 0; attempt <= 5; attempt++ {
		d := service.CalcBackoff(domain.RetryConstant, 10, attempt)
		if d != 10*time.Second {
			t.Errorf("attempt=%d: constant backoff should be 10s, got %v", attempt, d)
		}
	}
}

func TestCalcBackoff_ExponentialCap(t *testing.T) {
	d := service.CalcBackoff(domain.RetryExponential, 300, 20)
	if d > 3601*time.Second {
		t.Errorf("backoff should be capped at ~3600s, got %v", d)
	}
}

func BenchmarkCalcBackoff_Exponential(b *testing.B) {
	for b.Loop() {
		service.CalcBackoff(domain.RetryExponential, 5, 3)
	}
}
