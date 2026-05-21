package http

import (
	"encoding/json"
	"fmt"
	stdhttp "net/http"

	"garudapanel/internal/auth"
	api "garudapanel/internal/http/api"
	"garudapanel/internal/store"
	"garudapanel/internal/vendor"
	"garudapanel/internal/wallet"
)

type Server struct{ http *stdhttp.Server }
func New(addr string, jwtSecret string) *Server {
	mux:=stdhttp.NewServeMux()
	mux.HandleFunc("/health", func(w stdhttp.ResponseWriter, r *stdhttp.Request){ w.Header().Set("Content-Type","application/json"); _=json.NewEncoder(w).Encode(map[string]string{"status":"ok"}) })
	st := store.NewMemory()
	r := api.New(auth.NewService(st,jwtSecret), vendor.NewService(st), wallet.NewService(st))
	r.Register(mux)
	return &Server{http:&stdhttp.Server{Addr:fmt.Sprintf(":%s",addr), Handler:mux}}
}
func (s *Server) Start() error { return s.http.ListenAndServe() }
