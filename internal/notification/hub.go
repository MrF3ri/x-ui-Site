package notification

import (
	"encoding/json"
	"net/http"

	"garudapanel/internal/eventbus"
)

type Hub struct { bus *eventbus.Bus }
func NewHub(bus *eventbus.Bus) *Hub { return &Hub{bus: bus} }
func (h *Hub) WS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type","application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"hint":"upgrade to websocket in next phase", "status":"ok"})
}
func (h *Hub) Notify(topic string, payload any) { h.bus.Publish(eventbus.Event{Topic: topic, Payload: payload}) }
