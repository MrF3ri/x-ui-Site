package http

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	stdhttp "net/http"
	"net"
	"strconv"

	"garudapanel/internal/auth"
	api "garudapanel/internal/http/api"
	"garudapanel/internal/repository"
	"garudapanel/internal/storefront"
	"garudapanel/internal/vendor"
	"garudapanel/internal/wallet"
)

type Server struct{ http *stdhttp.Server }
func New(addr string, db *sql.DB, jwtSecret, appEnv, redisAddr, minioEndpoint string) *Server {
	mux := stdhttp.NewServeMux()
	mux.Handle("/assets/", stdhttp.StripPrefix("/assets/", stdhttp.FileServer(stdhttp.Dir("public/assets"))))
	if sfh, err := storefront.NewHandler(db); err == nil { mux.HandleFunc("/store", sfh.Router); mux.HandleFunc("/store/", sfh.Router) }
	mux.HandleFunc("/health", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		pg := "down"
		if db != nil && db.Ping() == nil { pg = "up" }
		redis := probeTCP(redisAddr)
		minio := probeTCP(minioEndpoint)
		writeJSON(w, 200, map[string]any{"status":"ok","app":"up","postgres":pg,"redis":redis,"minio":minio,"mode":appEnv,"environment":appEnv})
	})
	ur := repository.NewUserRepository(db); wr := repository.NewWalletRepository(db); vr := repository.NewVendorRepository(db); cr := repository.NewCatalogRepository(db)
	r := api.New(auth.NewService(ur, wr, jwtSecret), vendor.NewService(vr), wallet.NewService(wr)); r.Register(mux)
	mux.HandleFunc("/api/v1/vendor/catalog/create", func(w stdhttp.ResponseWriter, r *stdhttp.Request) { if r.Header.Get("X-Role") != "vendor" { httpError(w, 403, "forbidden"); return }; var p struct{ VendorID int64 `json:"vendor_id"`; Slug,Title,Description,Protocol string; InboundID,XUINodeID int64; TrafficLimitGB,DurationDays int; PriceToman int64; IsActive,AutoProvision,RenewalEnabled bool; CountryCode,StockStatus string }; if err := json.NewDecoder(r.Body).Decode(&p); err != nil { httpError(w, 400, "bad request"); return }; if err := cr.Create(storefrontToModel(p)); err != nil { httpError(w, 400, err.Error()); return }; writeJSON(w, 201, map[string]string{"status":"created"}) })
	mux.HandleFunc("/api/v1/vendor/catalog/list", func(w stdhttp.ResponseWriter, r *stdhttp.Request) { if r.Header.Get("X-Role") != "vendor" { httpError(w, 403, "forbidden"); return }; vid,_:=strconv.ParseInt(r.URL.Query().Get("vendor_id"),10,64); rows,err:=db.Query(`SELECT id,vendor_id,slug,title,description,protocol,inbound_id,xui_node_id,traffic_limit_gb,duration_days,price_toman,is_active,auto_provision,renewal_enabled,country_code,stock_status FROM catalog_items WHERE vendor_id=$1 AND deleted_at IS NULL`,vid); if err!=nil{httpError(w,400,err.Error()); return}; defer rows.Close(); out:=[]map[string]any{}; for rows.Next(){ var id,vendorID,inboundID,nodeID int64; var slug,title,description,protocol,country,stock string; var tgb,days int; var price int64; var active,ap,ren bool; _=rows.Scan(&id,&vendorID,&slug,&title,&description,&protocol,&inboundID,&nodeID,&tgb,&days,&price,&active,&ap,&ren,&country,&stock); out=append(out,map[string]any{"id":id,"vendor_id":vendorID,"slug":slug,"title":title,"description":description,"protocol":protocol,"inbound_id":inboundID,"xui_node_id":nodeID,"traffic_limit_gb":tgb,"duration_days":days,"price_toman":price,"is_active":active,"auto_provision":ap,"renewal_enabled":ren,"country_code":country,"stock_status":stock}) }; writeJSON(w,200,out) })

	h := recovery(mux)
	return &Server{http: &stdhttp.Server{Addr: fmt.Sprintf(":%s", addr), Handler: h}}
}
func (s *Server) Start() error { return s.http.ListenAndServe() }
func writeJSON(w stdhttp.ResponseWriter, code int, v any){ w.Header().Set("Content-Type","application/json"); w.WriteHeader(code); _=json.NewEncoder(w).Encode(v)}
func httpError(w stdhttp.ResponseWriter, code int, msg string){ writeJSON(w,code,map[string]string{"error":msg}) }
func probeTCP(addr string) string { if addr=="" { return "down" }; c, err := net.DialTimeout("tcp", addr, 800000000); if err!=nil { return "down" }; _ = c.Close(); return "up" }
func recovery(next stdhttp.Handler) stdhttp.Handler { return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request){ defer func(){ if rec:=recover(); rec!=nil { log.Printf("panic recovered path=%s err=%v", r.URL.Path, rec); httpError(w,500,"internal server error") } }(); next.ServeHTTP(w,r) }) }

func storefrontToModel(p struct{VendorID int64 `json:"vendor_id"`; Slug,Title,Description,Protocol string; InboundID,XUINodeID int64; TrafficLimitGB,DurationDays int; PriceToman int64; IsActive,AutoProvision,RenewalEnabled bool; CountryCode,StockStatus string}) repository.CatalogItemInput { return repository.CatalogItemInput{VendorID:p.VendorID,Slug:p.Slug,Title:p.Title,Description:p.Description,Protocol:p.Protocol,InboundID:p.InboundID,XUINodeID:p.XUINodeID,TrafficLimitGB:p.TrafficLimitGB,DurationDays:p.DurationDays,PriceToman:p.PriceToman,IsActive:p.IsActive,AutoProvision:p.AutoProvision,RenewalEnabled:p.RenewalEnabled,CountryCode:p.CountryCode,StockStatus:p.StockStatus} }
