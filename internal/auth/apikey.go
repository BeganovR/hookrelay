package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const keyPrefix = "hr_"

func Generate() (rawKey, keyHash, prefix string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", fmt.Errorf("generate api key: %w", err)
	}
	rawKey = keyPrefix + hex.EncodeToString(b)
	keyHash = Hash(rawKey)
	prefix = rawKey[:12]
	return
}

func Hash(rawKey string) string {
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}

func GenerateSigningSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate signing secret: %w", err)
	}
	return "whsec_" + hex.EncodeToString(b), nil
}
