package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey string

const (
	CtxUserID   contextKey = "user_id"
	CtxVendorID contextKey = "vendor_id"
	CtxRole     contextKey = "role"
)

// Claims holds decoded JWT payload fields we care about.
type Claims struct {
	Sub      int64  `json:"sub"`
	Role     string `json:"role"`
	VendorID int64  `json:"vendor_id"`
}

// JWT validates the Authorization: Bearer <token> header.
// On success it injects Claims into the request context.
func JWT(secret string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				jsonError(w, http.StatusUnauthorized, "missing token")
				return
			}
			claims, ok := parseJWT(secret, token)
			if !ok {
				jsonError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), CtxUserID, claims.Sub)
			ctx = context.WithValue(ctx, CtxRole, claims.Role)
			ctx = context.WithValue(ctx, CtxVendorID, claims.VendorID)
			next(w, r.WithContext(ctx))
		}
	}
}

// RequireRole returns 403 if the authenticated role is not in the allowed list.
func RequireRole(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(CtxRole).(string)
			if _, ok := allowed[role]; !ok {
				jsonError(w, http.StatusForbidden, "forbidden")
				return
			}
			next(w, r)
		}
	}
}

// Chain applies middlewares right-to-left so the first listed runs first.
func Chain(h http.HandlerFunc, mws ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// SecureHeaders adds security-related HTTP headers to every response.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		next.ServeHTTP(w, r)
	})
}

// UserIDFromCtx extracts user ID from context (set by JWT middleware).
func UserIDFromCtx(ctx context.Context) int64 {
	v, _ := ctx.Value(CtxUserID).(int64)
	return v
}

// RoleFromCtx extracts role from context.
func RoleFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(CtxRole).(string)
	return v
}

// VendorIDFromCtx extracts vendor ID from context.
func VendorIDFromCtx(ctx context.Context) int64 {
	v, _ := ctx.Value(CtxVendorID).(int64)
	return v
}

// --- internal helpers ---

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

func parseJWT(secret, token string) (Claims, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, false
	}
	if !verifyHMAC(secret, parts[0]+"."+parts[1], parts[2]) {
		return Claims{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, false
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return Claims{}, false
	}
	return c, true
}

func verifyHMAC(secret, data, sig string) bool {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(data))
	expected := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}

// ParseToken exposes JWT parsing for non-middleware usage.
func ParseToken(secret, token string) (Claims, bool) { return parseJWT(secret, token) }

// ExtractToken extracts a bearer token from the request without returning errors.
func ExtractToken(r *http.Request) string { return extractBearer(r) }
