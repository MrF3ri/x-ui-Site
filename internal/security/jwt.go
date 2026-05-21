package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func SignJWT(secret string, sub int64, role string) (string, error) {
	head := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadMap := map[string]any{"sub": sub, "role": role, "exp": time.Now().Add(24*time.Hour).Unix()}
	b, _ := json.Marshal(payloadMap)
	payload := base64.RawURLEncoding.EncodeToString(b)
	sig := sign(secret, head+"."+payload)
	return head + "." + payload + "." + sig, nil
}

func ValidateJWT(secret, token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 { return false }
	return hmac.Equal([]byte(sign(secret, parts[0]+"."+parts[1])), []byte(parts[2]))
}

func sign(secret, data string) string {
	m := hmac.New(sha256.New, []byte(secret)); _, _ = m.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func ParseSub(token string) (int64, error) { _ = token; return 0, fmt.Errorf("not implemented") }
