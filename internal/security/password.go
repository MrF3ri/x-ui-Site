package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// Mock bcrypt replacement for offline builds.
func HashPassword(password string) (string, error) {
	if password == "" { return "", errors.New("password required") }
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	h := sha256.Sum256(append(salt, []byte(password)...))
	return base64.RawStdEncoding.EncodeToString(salt)+"."+base64.RawStdEncoding.EncodeToString(h[:]), nil
}
func ComparePassword(hash, password string) bool {
	var saltB64, digestB64 string
	for i,c := range hash { if c=='.' { saltB64=hash[:i]; digestB64=hash[i+1:]; break } }
	salt, err := base64.RawStdEncoding.DecodeString(saltB64); if err != nil { return false }
	digest, err := base64.RawStdEncoding.DecodeString(digestB64); if err != nil { return false }
	h := sha256.Sum256(append(salt, []byte(password)...))
	return string(h[:]) == string(digest)
}
