package verifier

import (
	"encoding/json"
	"fmt"
	"hookrelay/internal/domain"
	"net/http"
)

type Verifier interface {
	Verify(r *http.Request, body []byte) error
}

func New(verifierType string, cfgJSON []byte) (Verifier, error) {
	switch verifierType {
	case domain.VerifierNoop, "":
		return &Noop{}, nil
	case domain.VerifierHMAC:
		var cfg domain.HMACVerifierConfig
		if err := json.Unmarshal(cfgJSON, &cfg); err != nil {
			return nil, fmt.Errorf("parse hmac config: %w", err)
		}
		return &HMAC{cfg: cfg}, nil
	default:
		return nil, fmt.Errorf("unknown verifier type: %s", verifierType)
	}
}
