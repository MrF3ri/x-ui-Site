package order

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

// Handler exposes HTTP endpoints for the order service.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/order/create", h.create)
	mux.HandleFunc("/api/v1/order/get", h.get)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errBody("method not allowed"))
		return
	}

	var body struct {
		VendorID       int64  `json:"vendor_id"`
		UserID         int64  `json:"user_id"`
		CatalogID      int64  `json:"catalog_id"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody("bad request"))
		return
	}

	resp, err := h.svc.Create(context.Background(), CreateRequest{
		VendorID:       body.VendorID,
		UserID:         body.UserID,
		CatalogID:      body.CatalogID,
		IdempotencyKey: body.IdempotencyKey,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err.Error()))
		return
	}

	code := http.StatusCreated
	if resp.Duplicate {
		code = http.StatusOK
	}
	writeJSON(w, code, resp)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	vendorID, _ := strconv.ParseInt(r.URL.Query().Get("vendor_id"), 10, 64)
	orderID, _ := strconv.ParseInt(r.URL.Query().Get("order_id"), 10, 64)
	if vendorID == 0 || orderID == 0 {
		writeJSON(w, http.StatusBadRequest, errBody("vendor_id and order_id required"))
		return
	}
	o, err := h.svc.orders.ByID(context.Background(), vendorID, orderID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errBody("order not found"))
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func errBody(msg string) map[string]string { return map[string]string{"error": msg} }