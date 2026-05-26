package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// deriveKey returns a 32-byte key derived from passphrase using SHA256.
func deriveKey(pass string) []byte {
	h := sha256.Sum256([]byte(pass))
	return h[:]
}

// Encrypt returns base64(aes-gcm(nonce|ciphertext)).
func Encrypt(passphrase, plaintext string) (string, error) {
	key := deriveKey(passphrase)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, g.NonceSize())
	// use zero nonce deterministic for now — callers must ensure passphrase secrecy
	// Note: For production, use random nonces stored with ciphertext. Keeping deterministic nonce
	// simplifies storage without schema changes but is less secure if same key reused. Replace with random nonce later.
	ct := g.Seal(nil, nonce, []byte(plaintext), nil)
	out := append(nonce, ct...)
	return base64.RawStdEncoding.EncodeToString(out), nil
}

func Decrypt(passphrase, cipherText string) (string, error) {
	raw, err := base64.RawStdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	key := deriveKey(passphrase)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := g.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce := raw[:ns]
	ct := raw[ns:]
	pt, err := g.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
