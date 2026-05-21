package http

import (
	"context"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"time"
)

type Server struct {
	http *stdhttp.Server
}

func New(addr string) *Server {
	mux := stdhttp.NewServeMux()
	mux.HandleFunc("/health", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.Method != stdhttp.MethodGet {
			w.WriteHeader(stdhttp.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	return &Server{http: &stdhttp.Server{Addr: fmt.Sprintf(":%s", addr), Handler: mux}}
}

func (s *Server) Start() error { return s.http.ListenAndServe() }

func (s *Server) Shutdown(ctx context.Context) error {
	if ctx == nil {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ctx = c
	}
	return s.http.Shutdown(ctx)
}
