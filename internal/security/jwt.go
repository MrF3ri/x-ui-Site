package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

func SignJWT(secret string, sub int64, role string) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	b,_ := json.Marshal(map[string]any{"sub":sub,"role":role,"exp":time.Now().Add(24*time.Hour).Unix()})
	payload := base64.RawURLEncoding.EncodeToString(b)
	sig := sign(secret, header+"."+payload)
	return header+"."+payload+"."+sig, nil
}
func ValidateJWT(secret, token string) bool { p:=strings.Split(token,"."); if len(p)!=3{return false}; return hmac.Equal([]byte(sign(secret,p[0]+"."+p[1])),[]byte(p[2])) }
func sign(secret,data string) string { h:=hmac.New(sha256.New,[]byte(secret)); _,_=h.Write([]byte(data)); return base64.RawURLEncoding.EncodeToString(h.Sum(nil)) }
