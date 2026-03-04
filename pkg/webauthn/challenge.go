package webauthn

import "crypto/rand"

// NewChallenge generates a 32-byte random challenge.
func NewChallenge() ([]byte, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return b, err
}
