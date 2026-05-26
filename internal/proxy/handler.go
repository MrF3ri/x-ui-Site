package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

// Handler exposes HTTP endpoints for proxy service management.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/service/provision", h.provision)
	mux.HandleFunc("/api/v1/service/get",       h.get)
	mux.HandleFunc("/api/v1/service/list",      h.list)
	mux.HandleFunc("/api/v1/service/suspend",   h.suspend)
	mux.HandleFunc("/api/v1/service/resume",    h.resume)
}

func (h *Handler) provision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errBody("method not allowed"))
		return
	}
	var body struct {
		VendorID       int64  `json:"vendor_id"`
		UserID         int64  `json:"user_id"`
		OrderID        *int64 `json:"order_id"`
		Protocol       string `json:"protocol"`
		TrafficLimitGB int    `json:"traffic_limit_gb"`
		DurationDays   int    `json:"duration_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody("bad request"))
		return
	}
	if body.Protocol == "" {
		body.Protocol = "vless"
	}
	if body.DurationDays == 0 {
		body.DurationDays = 30
	}
	if body.TrafficLimitGB == 0 {
		body.TrafficLimitGB = 100
	}

	resp, err := h.svc.Provision(context.Background(), ProvisionRequest{
		VendorID:       body.VendorID,
		UserID:         body.UserID,
		OrderID:        body.OrderID,
		Protocol:       body.Protocol,
		TrafficLimitGB: body.TrafficLimitGB,
		DurationDays:   body.DurationDays,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err.Error()))
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	vendorID, _ := strconv.ParseInt(r.URL.Query().Get("vendor_id"), 10, 64)
	serviceID, _ := strconv.ParseInt(r.URL.Query().Get("service_id"), 10, 64)
	if vendorID == 0 || serviceID == 0 {
		writeJSON(w, http.StatusBadRequest, errBody("vendor_id and service_id required"))
		return
	}
	rec, err := h.svc.GetByID(context.Background(), vendorID, serviceID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errBody("service not found"))
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	vendorID, _ := strconv.ParseInt(r.URL.Query().Get("vendor_id"), 10, 64)
	userID, _ := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
	if vendorID == 0 || userID == 0 {
		writeJSON(w, http.StatusBadRequest, errBody("vendor_id and user_id required"))
		return
	}
	recs, err := h.svc.ListByUser(context.Background(), vendorID, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errBody("failed to list services"))
		return
	}
	writeJSON(w, http.StatusOK, recs)
}

func (h *Handler) suspend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errBody("method not allowed"))
		return
	}
	var body struct {
		VendorID  int64 `json:"vendor_id"`
		ServiceID int64 `json:"service_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody("bad request"))
		return
	}
	if err := h.svc.Suspend(context.Background(), body.VendorID, body.ServiceID); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "suspended"})
}

func (h *Handler) resume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errBody("method not allowed"))
		return
	}
	var body struct {
		VendorID  int64 `json:"vendor_id"`
		ServiceID int64 `json:"service_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody("bad request"))
		return
	}
	if err := h.svc.Resume(context.Background(), body.VendorID, body.ServiceID); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func errBody(msg string) map[string]string { return map[string]string{"error": msg} }
