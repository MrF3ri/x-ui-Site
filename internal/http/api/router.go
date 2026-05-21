package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"garudapanel/internal/auth"
	"garudapanel/internal/vendor"
	"garudapanel/internal/wallet"
)

type Router struct { auth *auth.Service; vendor *vendor.Service; wallet *wallet.Service }
func New(a *auth.Service, v *vendor.Service, w *wallet.Service) *Router { return &Router{a,v,w} }
func (r *Router) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/auth/register", r.register)
	mux.HandleFunc("/api/v1/auth/login", r.login)
	mux.HandleFunc("/api/v1/vendor/create", r.vendorCreate)
	mux.HandleFunc("/api/v1/vendor/list", r.vendorList)
	mux.HandleFunc("/api/v1/wallet/balance", r.walletBalance)
}
func decode(req *http.Request, dst any) error { return json.NewDecoder(req.Body).Decode(dst) }
func write(w http.ResponseWriter, c int, v any){ w.Header().Set("Content-Type","application/json"); w.WriteHeader(c); _ = json.NewEncoder(w).Encode(v)}

func (r *Router) register(w http.ResponseWriter, req *http.Request){ var b struct{Email,Password string}; if decode(req,&b)!=nil {write(w,400,map[string]string{"error":"bad request"}); return}; if err:=r.auth.Register(b.Email,b.Password); err!=nil{write(w,400,map[string]string{"error":err.Error()});return}; write(w,201,map[string]string{"status":"registered"}) }
func (r *Router) login(w http.ResponseWriter, req *http.Request){ var b struct{Email,Password string}; if decode(req,&b)!=nil {write(w,400,map[string]string{"error":"bad request"}); return}; t,err:=r.auth.Login(b.Email,b.Password); if err!=nil{write(w,401,map[string]string{"error":"unauthorized"});return}; write(w,200,map[string]string{"token":t}) }
func (r *Router) vendorCreate(w http.ResponseWriter, req *http.Request){ var b struct{OwnerID int64 `json:"owner_id"`; Name,Slug string}; if decode(req,&b)!=nil {write(w,400,map[string]string{"error":"bad request"}); return}; if err:=r.vendor.Create(b.OwnerID,b.Name,b.Slug); err!=nil{write(w,400,map[string]string{"error":err.Error()});return}; write(w,201,map[string]string{"status":"created"}) }
func (r *Router) vendorList(w http.ResponseWriter, req *http.Request){ ownerID,_:=strconv.ParseInt(req.URL.Query().Get("owner_id"),10,64); data,err:=r.vendor.ListByOwner(ownerID); if err!=nil{write(w,400,map[string]string{"error":err.Error()});return}; write(w,200,data) }
func (r *Router) walletBalance(w http.ResponseWriter, req *http.Request){ uid,_:=strconv.ParseInt(req.URL.Query().Get("user_id"),10,64); bal,err:=r.wallet.Balance(uid); if err!=nil{write(w,404,map[string]string{"error":"wallet not found"});return}; write(w,200,map[string]int64{"balance":bal}) }
