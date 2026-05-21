package notification

import (
	"encoding/json"
	"net/http"
	"sync"

	"garudapanel/internal/eventbus"
)

type Hub struct { bus *eventbus.Bus; mu sync.Mutex; clients map[chan []byte]struct{} }
func NewHub(bus *eventbus.Bus) *Hub { return &Hub{bus: bus, clients: map[chan []byte]struct{}{}} }
func (h *Hub) WS(w http.ResponseWriter, r *http.Request) { // SSE transport for realtime push
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, ok := w.(http.Flusher); if !ok { http.Error(w,"stream unsupported",500); return }
	ch := make(chan []byte,16)
	h.mu.Lock(); h.clients[ch]=struct{}{}; h.mu.Unlock()
	defer func(){ h.mu.Lock(); delete(h.clients,ch); h.mu.Unlock(); close(ch) }()
	for {
		select {
		case <-r.Context().Done(): return
		case msg := <-ch:
			_, _ = w.Write([]byte("data: "+string(msg)+"\n\n")); flusher.Flush()
		}
	}
}
func (h *Hub) Notify(topic string, payload any) { h.bus.Publish(eventbus.Event{Topic: topic, Payload: payload}); b,_:=json.Marshal(map[string]any{"topic":topic,"payload":payload}); h.mu.Lock(); defer h.mu.Unlock(); for c:= range h.clients { select{case c<-b:default:} } }
