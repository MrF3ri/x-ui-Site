package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	stdhttp "net/http"
	"net"
	"strconv"
	"time"

	"garudapanel/internal/auth"
	"garudapanel/internal/eventbus"
	api "garudapanel/internal/http/api"
	"garudapanel/internal/notification"
	"garudapanel/internal/order"
	"garudapanel/internal/proxy"
	"garudapanel/internal/repository"
	"garudapanel/internal/storefront"
	"garudapanel/internal/vendor"
	"garudapanel/internal/wallet"
)

type Server struct{ http *stdhttp.Server }

func New(addr string, db *sql.DB, jwtSecret, appEnv, redisAddr, minioEndpoint string) *Server {
	mux := stdhttp.NewServeMux()

	// ── Static assets ─────────────────────────────────────────────────────
	mux.Handle("/assets/", stdhttp.StripPrefix("/assets/", stdhttp.FileServer(stdhttp.Dir("public/assets"))))

	// ── Storefront ────────────────────────────────────────────────────────
	if sfh, err := storefront.NewHandler(db); err == nil {
		mux.HandleFunc("/store", sfh.Router)
		mux.HandleFunc("/store/", sfh.Router)
	} else {
		log.Printf("storefront init failed: %v", err)
	}

	// ── Health ────────────────────────────────────────────────────────────
	mux.HandleFunc("/health", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		pg := "down"
		if db != nil && db.Ping() == nil {
			pg = "up"
		}
		writeJSON(w, 200, map[string]any{
			"status": "ok", "app": "up",
			"postgres": pg,
			"redis":    probeTCP(redisAddr),
			"minio":    probeTCP(minioEndpoint),
			"mode":     appEnv,
		})
	})

	// ── Repositories ──────────────────────────────────────────────────────
	ur  := repository.NewUserRepository(db)
	wr  := repository.NewWalletRepository(db)
	vr  := repository.NewVendorRepository(db)
	cr  := repository.NewCatalogRepository(db)
	or_ := repository.NewOrderRepository(db)
	ir  := repository.NewIdempotencyRepository(db)
	jr  := repository.NewProvisioningJobRepository(db)
	wt  := repository.NewWalletTxRepository(db)
	cp  := repository.NewCatalogPriceRepository(db)
	psr := repository.NewProxyServiceRepository(db)
	xpr := repository.NewXUIPanelRepository(db)

	// ── Notification hub ─────────────────────────────────────────────────
	bus := eventbus.New()
	hub := notification.NewHub(bus)

	// ── Existing auth/vendor/wallet API ──────────────────────────────────
	r := api.New(auth.NewService(ur, wr, jwtSecret), vendor.NewService(vr), wallet.NewService(wr))
	r.Register(mux)

	// ── Order engine ──────────────────────────────────────────────────────
	orderSvc := order.NewService(db, or_, ir, jr, wt, cp, hub)
	order.NewHandler(orderSvc).Register(mux)

	// ── Proxy service engine ──────────────────────────────────────────────
	proxySvc := proxy.NewService(db, psr, xpr, jr, or_, hub)
	proxy.NewHandler(proxySvc).Register(mux)

	// ── Catalog CRUD ──────────────────────────────────────────────────────
	mux.HandleFunc("/api/v1/vendor/catalog/create", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.Header.Get("X-Role") != "vendor" {
			httpError(w, 403, "forbidden")
			return
		}
		var p struct {
			VendorID       int64  `json:"vendor_id"`
			Slug           string
			Title          string
			Description    string
			Protocol       string
			InboundID      int64  `json:"inbound_id"`
			XUINodeID      int64  `json:"xui_node_id"`
			TrafficLimitGB int    `json:"traffic_limit_gb"`
			DurationDays   int    `json:"duration_days"`
			PriceToman     int64  `json:"price_toman"`
			IsActive       bool   `json:"is_active"`
			AutoProvision  bool   `json:"auto_provision"`
			RenewalEnabled bool   `json:"renewal_enabled"`
			CountryCode    string `json:"country_code"`
			StockStatus    string `json:"stock_status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			httpError(w, 400, "bad request")
			return
		}
		if err := cr.Create(repository.CatalogItemInput{
			VendorID: p.VendorID, Slug: p.Slug, Title: p.Title,
			Description: p.Description, Protocol: p.Protocol,
			InboundID: p.InboundID, XUINodeID: p.XUINodeID,
			TrafficLimitGB: p.TrafficLimitGB, DurationDays: p.DurationDays,
			PriceToman: p.PriceToman, IsActive: p.IsActive,
			AutoProvision: p.AutoProvision, RenewalEnabled: p.RenewalEnabled,
			CountryCode: p.CountryCode, StockStatus: p.StockStatus,
		}); err != nil {
			httpError(w, 400, err.Error())
			return
		}
		writeJSON(w, 201, map[string]string{"status": "created"})
	})

	mux.HandleFunc("/api/v1/vendor/catalog/list", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.Header.Get("X-Role") != "vendor" {
			httpError(w, 403, "forbidden")
			return
		}
		vid, _ := strconv.ParseInt(r.URL.Query().Get("vendor_id"), 10, 64)
		rows, err := db.Query(
			`SELECT id,vendor_id,slug,title,description,protocol,inbound_id,xui_node_id,
			        traffic_limit_gb,duration_days,price_toman,is_active,auto_provision,
			        renewal_enabled,country_code,stock_status
			 FROM catalog_items WHERE vendor_id=$1 AND deleted_at IS NULL`, vid)
		if err != nil {
			httpError(w, 400, err.Error())
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, vendorID, inboundID, nodeID int64
			var slug, title, description, protocol, country, stock string
			var tgb, days int
			var price int64
			var active, ap, ren bool
			_ = rows.Scan(&id, &vendorID, &slug, &title, &description, &protocol,
				&inboundID, &nodeID, &tgb, &days, &price, &active, &ap, &ren, &country, &stock)
			out = append(out, map[string]any{
				"id": id, "vendor_id": vendorID, "slug": slug,
				"title": title, "description": description, "protocol": protocol,
				"inbound_id": inboundID, "xui_node_id": nodeID,
				"traffic_limit_gb": tgb, "duration_days": days, "price_toman": price,
				"is_active": active, "auto_provision": ap, "renewal_enabled": ren,
				"country_code": country, "stock_status": stock,
			})
		}
		writeJSON(w, 200, out)
	})

	// ── Panel management ──────────────────────────────────────────────────
	mux.HandleFunc("/api/v1/vendor/panel/add", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.Header.Get("X-Role") != "vendor" {
			httpError(w, 403, "forbidden")
			return
		}
		var body struct {
			VendorID  int64  `json:"vendor_id"`
			Name      string `json:"name"`
			URL       string `json:"url"`
			Token     string `json:"token"`
			InboundID int64  `json:"inbound_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, 400, "bad request")
			return
		}
		id, err := xpr.Create(context.Background(), repository.XUIPanelRecord{
			VendorID:  body.VendorID,
			Name:      body.Name,
			URL:       body.URL,
			Token:     body.Token,
			InboundID: body.InboundID,
			IsActive:  true,
		})
		if err != nil {
			httpError(w, 400, err.Error())
			return
		}
		writeJSON(w, 201, map[string]any{"panel_id": id, "status": "created"})
	})

	mux.HandleFunc("/api/v1/vendor/panel/list", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.Header.Get("X-Role") != "vendor" {
			httpError(w, 403, "forbidden")
			return
		}
		vid, _ := strconv.ParseInt(r.URL.Query().Get("vendor_id"), 10, 64)
		panels, err := xpr.ListByVendor(context.Background(), vid)
		if err != nil {
			httpError(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, panels)
	})

	// ── SSE notification stream ───────────────────────────────────────────
	mux.HandleFunc("/api/v1/events", hub.WS)

	// ── Background jobs ───────────────────────────────────────────────────
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			proxySvc.RunExpiryJob(context.Background())
		}
	}()

	return &Server{http: &stdhttp.Server{
		Addr:    fmt.Sprintf(":%s", addr),
		Handler: recovery(mux),
	}}
}

func (s *Server) Start() error { return s.http.ListenAndServe() }

func writeJSON(w stdhttp.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w stdhttp.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func probeTCP(addr string) string {
	if addr == "" {
		return "down"
	}
	c, err := net.DialTimeout("tcp", addr, 800*time.Millisecond)
	if err != nil {
		return "down"
	}
	_ = c.Close()
	return "up"
}

func recovery(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered path=%s err=%v", r.URL.Path, rec)
				httpError(w, 500, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}