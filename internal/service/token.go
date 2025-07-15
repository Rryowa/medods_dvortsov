package service

import (
	"crypto/rand"
	"encoding/base64"
)

// FIXME: Investigate libs
func generateRandomToken() (string, error) {
	b := make([]byte, 32) // 256-bit entropy
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
