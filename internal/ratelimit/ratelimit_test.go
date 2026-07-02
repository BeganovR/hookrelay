package ratelimit

import "testing"

func TestLimiter_AllowWithinBurst(t *testing.T) {
	l := New(1, 3)

	for i := 0; i < 3; i++ {
		if !l.Allow("src-1") {
			t.Fatalf("request %d: expected allow within burst", i)
		}
	}
	if l.Allow("src-1") {
		t.Fatal("expected deny after burst exhausted")
	}
}

func TestLimiter_PerKeyIsolation(t *testing.T) {
	l := New(1, 1)

	if !l.Allow("src-1") {
		t.Fatal("expected first request for src-1 to be allowed")
	}
	if l.Allow("src-1") {
		t.Fatal("expected second request for src-1 to be denied")
	}
	if !l.Allow("src-2") {
		t.Fatal("expected first request for src-2 to be allowed independently of src-1")
	}
}
