package http

import (
	"database/sql"
	"encoding/json"
	"fmt"
	stdhttp "net/http"

	"garudapanel/internal/frontend"

	"garudapanel/internal/auth"
	api "garudapanel/internal/http/api"
	"garudapanel/internal/repository"
	"garudapanel/internal/vendor"
	"garudapanel/internal/wallet"
)

type Server struct{ http *stdhttp.Server }
func New(addr string, db *sql.DB, jwtSecret string) *Server {
	mux:=stdhttp.NewServeMux()
	if afs, err := frontend.AssetsFS(); err == nil {
		mux.Handle("/assets/", stdhttp.StripPrefix("/assets/", stdhttp.FileServer(stdhttp.FS(afs))))
	}
	if fh, err := frontend.New(); err == nil {
		mux.HandleFunc("/store/", fh.Store)
		mux.HandleFunc("/store", fh.Store)
		mux.HandleFunc("/store/product", fh.Product)
		mux.HandleFunc("/store/service", fh.Service)
	}
	mux.HandleFunc("/health", func(w stdhttp.ResponseWriter, r *stdhttp.Request){ w.Header().Set("Content-Type","application/json"); _=json.NewEncoder(w).Encode(map[string]string{"status":"ok"}) })
	ur := repository.NewUserRepository(db)
	wr := repository.NewWalletRepository(db)
	vr := repository.NewVendorRepository(db)
	r := api.New(auth.NewService(ur,wr,jwtSecret), vendor.NewService(vr), wallet.NewService(wr))
	r.Register(mux)
	return &Server{http:&stdhttp.Server{Addr:fmt.Sprintf(":%s",addr), Handler:mux}}
}
func (s *Server) Start() error { return s.http.ListenAndServe() }
