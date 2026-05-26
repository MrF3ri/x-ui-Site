package xui

import (
	"context"
	"fmt"
	"time"
)

type Panel struct {
	VendorID         int64
	Name, URL, Token string
	InboundID        int64
}
type ProvisionRequest struct {
	VendorID, UserID      int64
	Email, Protocol, UUID string
	DurationDays          int
	TrafficGB             int
}
type ProvisionResponse struct {
	UUID      string
	ExpiresAt time.Time
}

type Adapter interface {
	Provision(panel Panel, req ProvisionRequest) (ProvisionResponse, error)
	Renew(panel Panel, uuid string, addDays int, addGB int) (ProvisionResponse, error)
}

type MultiPanelAdapter struct {
	client  *Client
	retries int
}

func NewAdapter(client *Client) *MultiPanelAdapter {
	return &MultiPanelAdapter{client: client, retries: 3}
}
func (a *MultiPanelAdapter) Provision(panel Panel, req ProvisionRequest) (ProvisionResponse, error) {
	exp := time.Now().Add(time.Duration(req.DurationDays) * 24 * time.Hour).Unix()
	var last error
	for i := 0; i < a.retries; i++ {
		out, err := a.client.CreateClient(context.Background(), panel, panel.InboundID, req.Email, req.UUID, exp, req.TrafficGB)
		if err == nil {
			return out, nil
		}
		last = err
		time.Sleep(time.Duration(i+1) * 250 * time.Millisecond)
	}
	return ProvisionResponse{}, fmt.Errorf("provision failed after retries: %w", last)
}
func (a *MultiPanelAdapter) Renew(panel Panel, uuid string, addDays int, addGB int) (ProvisionResponse, error) {
	return a.Provision(panel, ProvisionRequest{UUID: uuid, DurationDays: addDays, TrafficGB: addGB, Email: uuid})
}
