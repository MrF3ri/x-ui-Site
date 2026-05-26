package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password required")
	}
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	h := sha256.Sum256(append(salt, []byte(password)...))
	return base64.RawStdEncoding.EncodeToString(salt) + "." + base64.RawStdEncoding.EncodeToString(h[:]), nil
}
func ComparePassword(hash, password string) bool {
	var a, b string
	for i, c := range hash {
		if c == '.' {
			a = hash[:i]
			b = hash[i+1:]
			break
		}
	}
	s, err := base64.RawStdEncoding.DecodeString(a)
	if err != nil {
		return false
	}
	d, err := base64.RawStdEncoding.DecodeString(b)
	if err != nil {
		return false
	}
	h := sha256.Sum256(append(s, []byte(password)...))
	return string(h[:]) == string(d)
}
