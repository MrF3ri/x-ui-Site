package audit

import (
	"context"
	"database/sql"
	"log"
	"net/http"
)

// Entry represents one audit log record.
type Entry struct {
	VendorID   *int64
	UserID     *int64
	ActorRole  string
	Action     string
	Resource   string
	ResourceID *int64
	IP         string
	UserAgent  string
	Status     string // ok | denied | error
	Detail     string
}

// Logger writes audit entries to the database.
type Logger struct{ db *sql.DB }

func NewLogger(db *sql.DB) *Logger { return &Logger{db: db} }

// Log writes an audit entry asynchronously (fire-and-forget).
func (l *Logger) Log(ctx context.Context, e Entry) {
	go func() {
		_, err := l.db.ExecContext(ctx,
			`INSERT INTO audit_logs
			 (vendor_id, user_id, actor_role, action, resource, resource_id, ip, user_agent, status, detail)
			 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			e.VendorID, e.UserID, e.ActorRole, e.Action,
			e.Resource, e.ResourceID, e.IP, e.UserAgent, e.Status, e.Detail,
		)
		if err != nil {
			log.Printf("audit: write failed: %v", err)
		}
	}()
}

// FromRequest builds a partial Entry from an HTTP request.
func FromRequest(r *http.Request, action, resource, status string) Entry {
	return Entry{
		Action:    action,
		Resource:  resource,
		IP:        realIP(r),
		UserAgent: r.Header.Get("User-Agent"),
		Status:    status,
	}
}

func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i, c := range xff {
			if c == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
