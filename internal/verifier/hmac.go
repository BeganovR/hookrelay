package verifier

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hookrelay/internal/domain"
	"net/http"
	"strings"
)

type HMAC struct {
	cfg domain.HMACVerifierConfig
}

func (h *HMAC) Verify(r *http.Request, body []byte) error {
	sigHeader := r.Header.Get(h.cfg.Header)
	if sigHeader == "" {
		return fmt.Errorf("missing signature header %q", h.cfg.Header)
	}
	sigHeader = strings.TrimPrefix(sigHeader, h.cfg.Prefix)

	var sigBytes []byte
	var err error
	switch h.cfg.Encoding {
	case "base64":
		sigBytes, err = base64.StdEncoding.DecodeString(sigHeader)
	default:
		sigBytes, err = hex.DecodeString(sigHeader)
	}
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(h.cfg.Secret))
	mac.Write(body)
	expected := mac.Sum(nil)

	if !hmac.Equal(sigBytes, expected) {
		return errors.New("signature mismatch")
	}
	return nil
}
