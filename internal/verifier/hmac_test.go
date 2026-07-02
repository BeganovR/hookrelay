package verifier_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"hookrelay/internal/domain"
	"hookrelay/internal/verifier"
)

func makeHMACVerifier(t *testing.T, secret, header, prefix, encoding string) verifier.Verifier {
	t.Helper()
	cfg := domain.HMACVerifierConfig{
		Header:   header,
		Prefix:   prefix,
		Secret:   secret,
		Encoding: encoding,
	}
	cfgJSON := []byte(`{"header":"` + header + `","prefix":"` + prefix + `","secret":"` + secret + `","encoding":"` + encoding + `"}`)
	v, err := verifier.New(domain.VerifierHMAC, cfgJSON)
	if err != nil {
		t.Fatalf("NewHMAC: %v (cfg=%+v)", err, cfg)
	}
	return v
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestHMAC_ValidSignature(t *testing.T) {
	body := []byte(`{"event":"order.created"}`)
	sig := sign(body, "test-secret")

	r := httptest.NewRequest(http.MethodPost, "/ingest/src", nil)
	r.Header.Set("X-Signature", "sha256="+sig)

	v := makeHMACVerifier(t, "test-secret", "X-Signature", "sha256=", "hex")
	if err := v.Verify(r, body); err != nil {
		t.Errorf("expected valid signature, got error: %v", err)
	}
}

func TestHMAC_InvalidSignature(t *testing.T) {
	body := []byte(`{"event":"order.created"}`)
	r := httptest.NewRequest(http.MethodPost, "/ingest/src", nil)
	r.Header.Set("X-Signature", "sha256=deadbeef")

	v := makeHMACVerifier(t, "test-secret", "X-Signature", "sha256=", "hex")
	if err := v.Verify(r, body); err == nil {
		t.Error("expected error for invalid signature, got nil")
	}
}

func TestHMAC_MissingHeader(t *testing.T) {
	body := []byte(`{"event":"order.created"}`)
	r := httptest.NewRequest(http.MethodPost, "/ingest/src", nil)

	v := makeHMACVerifier(t, "test-secret", "X-Signature", "sha256=", "hex")
	if err := v.Verify(r, body); err == nil {
		t.Error("expected error for missing header, got nil")
	}
}

func TestNoop_AlwaysValid(t *testing.T) {
	v, err := verifier.New(domain.VerifierNoop, nil)
	if err != nil {
		t.Fatalf("New noop: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	if err := v.Verify(r, []byte("anything")); err != nil {
		t.Errorf("noop should always pass, got: %v", err)
	}
}

func TestNew_UnknownType(t *testing.T) {
	_, err := verifier.New("unknown_type", nil)
	if err == nil {
		t.Error("expected error for unknown verifier type")
	}
}
