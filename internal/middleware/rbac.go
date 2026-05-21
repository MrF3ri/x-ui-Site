package middleware

import (
	"net/http"
	"strings"

	"garudapanel/internal/rbac"
)

func RBAC(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			role := strings.TrimSpace(r.Header.Get("X-Role"))
			if err := rbac.Enforce(role, roles...); err != nil { http.Error(w, "forbidden", http.StatusForbidden); return }
			next(w,r)
		}
	}
}
