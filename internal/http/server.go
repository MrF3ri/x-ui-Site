package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	stdhttp "net/http"
	"strconv"
	"strings"
	"time"

	"garudapanel/internal/audit"
	"garudapanel/internal/auth"
	"garudapanel/internal/dashboard"
	"garudapanel/internal/eventbus"
	api "garudapanel/internal/http/api"
	"garudapanel/internal/middleware"
	"garudapanel/internal/notification"
	"garudapanel/internal/order"
	"garudapanel/internal/provisioning"
	"garudapanel/internal/proxy"
	"garudapanel/internal/repository"
	"garudapanel/internal/storefront"
	"garudapanel/internal/vendor"
	"garudapanel/internal/wallet"
	"garudapanel/internal/xui"
)

type Server struct{ http *stdhttp.Server }

func New(addr string, db *sql.DB, jwtSecret, panelKey, appEnv, redisAddr, minioEndpoint string) *Server {
	mux := stdhttp.NewServeMux()

	// ── Infrastructure ────────────────────────────────────────────────────
	bus := eventbus.New()
	hub := notification.NewHub(bus)
	auditor := audit.NewLogger(db)
	limiter := middleware.NewRateLimiter(120, time.Minute) // 120 req/min per IP

	// ── Repositories ──────────────────────────────────────────────────────
	ur := repository.NewUserRepository(db)
	wr := repository.NewWalletRepository(db)
	vr := repository.NewVendorRepository(db)
	cr := repository.NewCatalogRepository(db)
	or_ := repository.NewOrderRepository(db)
	ir := repository.NewIdempotencyRepository(db)
	jr := repository.NewProvisioningJobRepository(db)
	wt := repository.NewWalletTxRepository(db)
	cp := repository.NewCatalogPriceRepository(db)
	psr := repository.NewProxyServiceRepository(db)
	xpr := repository.NewXUIPanelRepository(db, panelKey)

	// ── Services ──────────────────────────────────────────────────────────
	authSvc := auth.NewService(ur, wr, jwtSecret)
	vendorSvc := vendor.NewService(vr)
	walletSvc := wallet.NewService(wr)
	orderSvc := order.NewService(db, or_, ir, jr, wt, cp, hub, psr)
	proxySvc := proxy.NewService(db, psr, xpr, jr, or_, hub)

	// ── Provisioning worker (background)
	go func() {
		ctx := context.Background()
		adapter := xui.NewAdapter(xui.NewClient("", ""))
		w := provisioning.NewWorkerV2(db, jr, or_, cr, xpr, psr, adapter, hub)
		w.Start(ctx)
	}()

	// ── Static assets ─────────────────────────────────────────────────────
	mux.Handle("/assets/", stdhttp.StripPrefix("/assets/",
		stdhttp.FileServer(stdhttp.Dir("public/assets"))))

	// ── Storefront (public, SSR wallet-aware) ─────────────────────────────
	if sfh, err := storefront.NewHandler(db, jwtSecret, walletSvc); err == nil {
		mux.HandleFunc("/store", sfh.Router)
		mux.HandleFunc("/store/", sfh.Router)
	} else {
		log.Printf("storefront init failed: %v", err)
	}

	// ── Health (public) ───────────────────────────────────────────────────
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

	// ── Auth routes (public, rate-limited) ────────────────────────────────
	apiRouter := api.New(authSvc, vendorSvc, walletSvc)
	apiRouter.Register(mux)

	// ── Order routes (JWT required) ───────────────────────────────────────
	jwtMW := middleware.JWT(jwtSecret)
	userMW := middleware.RequireRole("user", "vendor", "admin")
	vendMW := middleware.RequireRole("vendor", "admin")
	adminMW := middleware.RequireRole("admin")

	// POST /api/v1/order/create
	mux.HandleFunc("/api/v1/order/create", middleware.Chain(
		func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			if r.Method != stdhttp.MethodPost {
				httpError(w, 405, "method not allowed")
				return
			}
			var body struct {
				VendorID       int64  `json:"vendor_id"`
				CatalogID      int64  `json:"catalog_id"`
				IdempotencyKey string `json:"idempotency_key"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				httpError(w, 400, "bad request")
				return
			}
			userID := middleware.UserIDFromCtx(r.Context())
			resp, err := orderSvc.Create(r.Context(), order.CreateRequest{
				VendorID:       body.VendorID,
				UserID:         userID,
				CatalogID:      body.CatalogID,
				IdempotencyKey: body.IdempotencyKey,
			})
			if err != nil {
				auditor.Log(r.Context(), audit.FromRequest(r, "order.create", "order", "error"))
				httpError(w, 400, err.Error())
				return
			}
			auditor.Log(r.Context(), audit.FromRequest(r, "order.create", "order", "ok"))
			code := 201
			if resp.Duplicate {
				code = 200
			}
			writeJSON(w, code, resp)
		},
		jwtMW, userMW,
	))

	// POST /api/v1/order/renew/:serviceID
	mux.HandleFunc("/api/v1/order/renew/", middleware.Chain(
		func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			if r.Method != stdhttp.MethodPost {
				httpError(w, 405, "method not allowed")
				return
			}
			sidStr := strings.TrimPrefix(r.URL.Path, "/api/v1/order/renew/")
			sid, _ := strconv.ParseInt(sidStr, 10, 64)
			if sid == 0 {
				httpError(w, 400, "invalid service id")
				return
			}
			userID := middleware.UserIDFromCtx(r.Context())
			resp, err := orderSvc.Renew(r.Context(), userID, sid)
			if err != nil {
				auditor.Log(r.Context(), audit.FromRequest(r, "order.renew", "order", "error"))
				httpError(w, 400, err.Error())
				return
			}
			auditor.Log(r.Context(), audit.FromRequest(r, "order.renew", "order", "ok"))
			writeJSON(w, 201, resp)
		},
		jwtMW, userMW,
	))

	// GET /api/v1/order/get
	mux.HandleFunc("/api/v1/order/get", middleware.Chain(
		func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			vendorID, _ := strconv.ParseInt(r.URL.Query().Get("vendor_id"), 10, 64)
			orderID, _ := strconv.ParseInt(r.URL.Query().Get("order_id"), 10, 64)
			o, err := or_.ByID(r.Context(), vendorID, orderID)
			if err != nil {
				httpError(w, 404, "not found")
				return
			}
			// Enforce: users can only see their own orders
			if middleware.RoleFromCtx(r.Context()) == "user" &&
				o.UserID != middleware.UserIDFromCtx(r.Context()) {
				httpError(w, 403, "forbidden")
				return
			}
			writeJSON(w, 200, o)
		},
		jwtMW, userMW,
	))

	// ── Proxy service routes (JWT required) ───────────────────────────────
	proxyHandler := proxy.NewHandler(proxySvc)
	proxyHandler.Register(mux)

	// ── User dashboard (JWT required) ───────────────────────────────────
	if dh := dashboard.NewHandler(db, or_, psr, wr, walletSvc); dh != nil {
		mux.HandleFunc("/dashboard", middleware.Chain(dh.Router, jwtMW, userMW))
		mux.HandleFunc("/dashboard/", middleware.Chain(dh.Router, jwtMW, userMW))
	}

	// ── Vendor: catalog management ────────────────────────────────────────
	mux.HandleFunc("/api/v1/vendor/catalog/create", middleware.Chain(
		func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			if r.Method != stdhttp.MethodPost {
				httpError(w, 405, "method not allowed")
				return
			}
			var p struct {
				VendorID       int64  `json:"vendor_id"`
				Slug           string `json:"slug"`
				Title          string `json:"title"`
				Description    string `json:"description"`
				Protocol       string `json:"protocol"`
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
			auditor.Log(r.Context(), audit.FromRequest(r, "catalog.create", "catalog_item", "ok"))
			writeJSON(w, 201, map[string]string{"status": "created"})
		},
		jwtMW, vendMW,
	))

	mux.HandleFunc("/api/v1/vendor/catalog/list", middleware.Chain(
		func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			vid, _ := strconv.ParseInt(r.URL.Query().Get("vendor_id"), 10, 64)
			rows, err := db.QueryContext(r.Context(),
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
		},
		jwtMW, vendMW,
	))

	// ── Panel management ──────────────────────────────────────────────────
	mux.HandleFunc("/api/v1/vendor/panel/add", middleware.Chain(
		func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			if r.Method != stdhttp.MethodPost {
				httpError(w, 405, "method not allowed")
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
			id, err := xpr.Create(r.Context(), repository.XUIPanelRecord{
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
			auditor.Log(r.Context(), audit.FromRequest(r, "panel.add", "xui_panel", "ok"))
			writeJSON(w, 201, map[string]any{"panel_id": id, "status": "created"})
		},
		jwtMW, vendMW,
	))

	mux.HandleFunc("/api/v1/vendor/panel/list", middleware.Chain(
		func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			vid, _ := strconv.ParseInt(r.URL.Query().Get("vendor_id"), 10, 64)
			panels, err := xpr.ListByVendor(r.Context(), vid)
			if err != nil {
				httpError(w, 400, err.Error())
				return
			}
			writeJSON(w, 200, panels)
		},
		jwtMW, vendMW,
	))

	// ── Wallet top-up (admin only) ────────────────────────────────────────
	mux.HandleFunc("/api/v1/admin/wallet/topup", middleware.Chain(
		func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			if r.Method != stdhttp.MethodPost {
				httpError(w, 405, "method not allowed")
				return
			}
			var body struct {
				UserID int64 `json:"user_id"`
				Amount int64 `json:"amount"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				httpError(w, 400, "bad request")
				return
			}
			if err := wr.ApplyTransaction(body.UserID, body.Amount, "admin_topup"); err != nil {
				httpError(w, 400, err.Error())
				return
			}
			auditor.Log(r.Context(), audit.FromRequest(r, "wallet.topup", "wallet", "ok"))
			writeJSON(w, 200, map[string]string{"status": "ok"})
		},
		jwtMW, adminMW,
	))

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

	// ── Wrap with security headers + rate limiter + recovery ──────────────
	handler := middleware.SecureHeaders(
		limiter.Middleware(
			recovery(mux),
		),
	)

	return &Server{http: &stdhttp.Server{
		Addr:         fmt.Sprintf(":%s", addr),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
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
