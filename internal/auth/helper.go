package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

func FingerprintHash(fingerprint string) string {
	hasher := sha256.New()

	return hex.EncodeToString(hasher.Sum([]byte(fingerprint)))
}
