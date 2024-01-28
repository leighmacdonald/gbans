package auth

import (
	"crypto/sha256"
	"fmt"
)

func FingerprintHash(fingerprint string) string {
	hasher := sha256.New()

	return fmt.Sprintf("%x", hasher.Sum([]byte(fingerprint)))
}
