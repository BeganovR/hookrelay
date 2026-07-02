package auth_test

import (
	"strings"
	"testing"

	"hookrelay/internal/auth"
)

func TestGenerate(t *testing.T) {
	raw, hash, prefix, err := auth.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.HasPrefix(raw, "hr_") {
		t.Errorf("raw key should start with hr_, got %q", raw[:min(10, len(raw))])
	}
	if len(hash) != 64 {
		t.Errorf("hash length should be 64, got %d", len(hash))
	}
	if prefix != raw[:12] {
		t.Errorf("prefix mismatch: %q vs raw[:12]=%q", prefix, raw[:12])
	}
}

func TestHash_Deterministic(t *testing.T) {
	h1 := auth.Hash("hr_testkey")
	h2 := auth.Hash("hr_testkey")
	if h1 != h2 {
		t.Error("Hash should be deterministic")
	}
}

func TestHash_Different(t *testing.T) {
	h1 := auth.Hash("hr_key1")
	h2 := auth.Hash("hr_key2")
	if h1 == h2 {
		t.Error("Different keys should produce different hashes")
	}
}

func TestGenerate_Unique(t *testing.T) {
	raw1, _, _, _ := auth.Generate()
	raw2, _, _, _ := auth.Generate()
	if raw1 == raw2 {
		t.Error("Generated keys should be unique")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
