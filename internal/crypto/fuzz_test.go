package crypto_test

import (
	"testing"

	"github.com/thereisnotime/kryptlet/internal/crypto"
)

func FuzzDecrypt(f *testing.F) {
	// Seed with cases that exercise each error path.
	f.Add([]byte("not-age-ciphertext"), "AGE-SECRET-KEY-1QQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQR")
	f.Add([]byte(""), "not-a-key")
	f.Add([]byte("age-encryption.org/v1\n"), "AGE-SECRET-KEY-1")
	f.Add([]byte{0x61, 0x67, 0x65}, "  AGE-SECRET-KEY-1QQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQR  ")

	f.Fuzz(func(t *testing.T, ciphertext []byte, key string) {
		// Must never panic — only return an error.
		_, _ = crypto.Decrypt(ciphertext, key)
	})
}
