package dashboard

import (
	"database/sql"
	"encoding/base64"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"

	"garudapanel/internal/middleware"
	"garudapanel/internal/repository"
	"garudapanel/internal/wallet"
)

type Handler struct {
	t         *sql.DB // keep DB for now if needed
	orders    *repository.OrderRepository
	proxy     *repository.ProxyServiceRepository
	walletR   *repository.WalletRepository
	walletSvc *wallet.Service
	tmpl      *template.Template
}

func NewHandler(db *sql.DB, orders *repository.OrderRepository, proxy *repository.ProxyServiceRepository, walletR *repository.WalletRepository, walletSvc *wallet.Service) *Handler {
	t := template.Must(template.ParseGlob("templates/dashboard/*.html"))
	return &Handler{t: db, orders: orders, proxy: proxy, walletR: walletR, walletSvc: walletSvc, tmpl: t}
}

func (h *Handler) Router(w http.ResponseWriter, r *http.Request) {
	// the routes are mounted at /dashboard and /dashboard/*
	p := strings.TrimPrefix(r.URL.Path, "/dashboard")
	p = strings.Trim(p, "/")
	userID := middleware.UserIDFromCtx(r.Context())
	if p == "" {
		// summary
		// balance
		bal, _ := h.walletSvc.Balance(userID)
		// services
		svcs, _ := h.proxy.ListAllByUser(r.Context(), userID)
		// orders
		orders, _ := h.orders.ListByUser(r.Context(), userID, 10)
		data := map[string]any{"Balance": bal, "Services": svcs, "Orders": orders}
		_ = h.tmpl.ExecuteTemplate(w, "dashboard_index.html", data)
		return
	}
	seg := strings.Split(p, "/")
	if seg[0] == "services" {
		if len(seg) == 1 {
			svcs, _ := h.proxy.ListAllByUser(r.Context(), userID)
			_ = h.tmpl.ExecuteTemplate(w, "dashboard_services.html", map[string]any{"Services": svcs})
			return
		}
		// detail
		id, _ := strconv.ParseInt(seg[1], 10, 64)
		rec, err := h.proxy.ByIDForUser(r.Context(), id, userID)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		// generate QR data URI if subscription URL present
		var qrd string
		if rec.SubscriptionURL != "" {
			png, _ := qrcode.Encode(rec.SubscriptionURL, qrcode.Medium, 256)
			qrd = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
		}
		_ = h.tmpl.ExecuteTemplate(w, "dashboard_service_detail.html", map[string]any{"Service": rec, "QRCodeDataURI": qrd})
		return
	}
	if seg[0] == "orders" {
		orders, _ := h.orders.ListByUser(r.Context(), userID, 50)
		_ = h.tmpl.ExecuteTemplate(w, "dashboard_orders.html", map[string]any{"Orders": orders})
		return
	}
	if seg[0] == "wallet" {
		bal, _ := h.walletSvc.Balance(userID)
		txs, _ := h.walletR.ListTransactions(userID, 20)
		_ = h.tmpl.ExecuteTemplate(w, "dashboard_wallet.html", map[string]any{"Balance": bal, "Transactions": txs})
		return
	}
	http.NotFound(w, r)
}

// (templates are parsed at NewHandler time using html/template)
