package storefront

import (
	"database/sql"
	"html/template"
	"net/http"
	"strings"
)

type Handler struct {
	svc *Service
	t   *template.Template
}

func NewHandler(db *sql.DB) (*Handler, error) {
	t, err := template.ParseGlob("templates/store/*.html")
	if err != nil {
		return nil, err
	}
	return &Handler{svc: New(db), t: t}, nil
}

func (h *Handler) render(w http.ResponseWriter, name string, data any, code int) {
	w.WriteHeader(code)
	_ = h.t.ExecuteTemplate(w, name, data)
}

func (h *Handler) Router(w http.ResponseWriter, r *http.Request) {
	p := strings.Trim(strings.TrimPrefix(r.URL.Path, "/store"), "/")
	if p == "" {
		h.render(w, "index.html", map[string]any{"Title": "GarudaPanel"}, 200)
		return
	}
	seg := strings.Split(p, "/")
	vendor, err := h.svc.GetVendorBySlug(seg[0])
	if err != nil {
		h.render(w, "not_found.html", map[string]any{"Message": "vendor not found"}, 404)
		return
	}
	if vendor.Status != "active" {
		h.render(w, "not_found.html", map[string]any{"Message": "vendor suspended"}, 404)
		return
	}

	// Vendor root or product listing
	if len(seg) == 1 || (len(seg) == 2 && seg[1] == "") {
		prods, _ := h.svc.ListProducts(vendor.ID)
		h.render(w, "vendor.html", map[string]any{"Vendor": vendor, "Products": prods}, 200)
		return
	}
	if len(seg) == 2 && seg[1] == "products" {
		prods, _ := h.svc.ListProducts(vendor.ID)
		h.render(w, "vendor.html", map[string]any{"Vendor": vendor, "Products": prods}, 200)
		return
	}
	if len(seg) == 3 && seg[1] == "products" {
		prod, err := h.svc.GetProduct(vendor.ID, seg[2])
		if err != nil || !prod.IsActive {
			h.render(w, "not_found.html", map[string]any{"Message": "product not found"}, 404)
			return
		}
		h.render(w, "product.html", map[string]any{"Vendor": vendor, "Product": prod}, 200)
		return
	}

	// Checkout page: /store/{vendor}/products/{slug}/checkout
	if len(seg) == 4 && seg[1] == "products" && seg[3] == "checkout" {
		prod, err := h.svc.GetProduct(vendor.ID, seg[2])
		if err != nil || !prod.IsActive {
			h.render(w, "not_found.html", map[string]any{"Message": "product not found"}, 404)
			return
		}
		h.render(w, "checkout.html", map[string]any{"Vendor": vendor, "Product": prod}, 200)
		return
	}

	// Purchase result pages
	if len(seg) == 2 && seg[1] == "purchase-success" {
		h.render(w, "purchase_success.html", map[string]any{"Vendor": vendor}, 200)
		return
	}
	if len(seg) == 2 && seg[1] == "purchase-failed" {
		h.render(w, "purchase_failed.html", map[string]any{"Vendor": vendor}, 200)
		return
	}

	h.render(w, "not_found.html", map[string]any{"Message": "page not found"}, 404)
}
