package frontend

import (
	"html/template"
	"net/http"
)

type Handler struct{ t *template.Template }

func New() (*Handler, error) {
	t, err := template.ParseGlob("templates/**/*.html")
	if err != nil { return nil, err }
	return &Handler{t: t}, nil
}

func (h *Handler) Store(w http.ResponseWriter, r *http.Request) {
	_ = h.t.ExecuteTemplate(w, "store_index.html", map[string]any{"Title": "GarudaPanel Store"})
}

func (h *Handler) Product(w http.ResponseWriter, r *http.Request) {
	_ = h.t.ExecuteTemplate(w, "store_product.html", map[string]any{"Title": "Product"})
}

func (h *Handler) Service(w http.ResponseWriter, r *http.Request) {
	_ = h.t.ExecuteTemplate(w, "store_service.html", map[string]any{"Title": "Service"})
}
