package crypto

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
)

// Decrypt decrypts age-encrypted ciphertext using the given X25519 private key.
// Returns the raw plaintext bytes, or an error if the key is invalid or decryption fails.
func Decrypt(ciphertext []byte, privateKey string) ([]byte, error) {
	identity, err := age.ParseX25519Identity(strings.TrimSpace(privateKey))
	if err != nil {
		return nil, fmt.Errorf("invalid key: %w", err)
	}
	r, err := age.Decrypt(bytes.NewReader(ciphertext), identity)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	return io.ReadAll(r)
}
