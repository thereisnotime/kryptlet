package crypto_test

import (
	"bytes"
	"testing"

	"filippo.io/age"

	"github.com/thereisnotime/kryptlet/internal/crypto"
)

func encryptForTest(t *testing.T, plaintext []byte) ([]byte, string) {
	t.Helper()
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, identity.Recipient())
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if _, err := w.Write(plaintext); err != nil {
		t.Fatalf("write plaintext: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return buf.Bytes(), identity.String()
}

func TestDecrypt(t *testing.T) {
	want := []byte(`{"hello":"world"}`)
	ciphertext, key := encryptForTest(t, want)
	got, err := crypto.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	ciphertext, _ := encryptForTest(t, []byte("secret"))
	other, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate other identity: %v", err)
	}
	if _, err := crypto.Decrypt(ciphertext, other.String()); err == nil {
		t.Error("expected error with wrong key, got nil")
	}
}

func TestDecrypt_InvalidKey(t *testing.T) {
	if _, err := crypto.Decrypt([]byte("data"), "not-a-valid-key"); err == nil {
		t.Error("expected error with invalid key, got nil")
	}
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	if _, err := crypto.Decrypt([]byte("not-age-ciphertext"), identity.String()); err == nil {
		t.Error("expected error with invalid ciphertext, got nil")
	}
}

func TestDecrypt_WhitespaceAroundKey(t *testing.T) {
	want := []byte("data")
	ciphertext, key := encryptForTest(t, want)
	got, err := crypto.Decrypt(ciphertext, "  "+key+"  \n")
	if err != nil {
		t.Fatalf("Decrypt with whitespace-padded key: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
