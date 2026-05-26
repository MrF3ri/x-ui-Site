package xui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL, token string
	http           *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{baseURL: baseURL, token: token, http: &http.Client{Timeout: 15 * time.Second}}
}

type createReq struct {
	InboundID  int64  `json:"inbound_id"`
	Email      string `json:"email"`
	UUID       string `json:"uuid"`
	ExpiryUnix int64  `json:"expiry_unix"`
	TrafficGB  int    `json:"traffic_gb"`
}

type createResp struct {
	UUID      string `json:"uuid"`
	ExpiresAt int64  `json:"expires_at"`
}

func (c *Client) CreateClient(ctx context.Context, panel Panel, inboundID int64, email, uuid string, expiryUnix int64, trafficGB int) (ProvisionResponse, error) {
	payload, _ := json.Marshal(createReq{InboundID: inboundID, Email: email, UUID: uuid, ExpiryUnix: expiryUnix, TrafficGB: trafficGB})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, panel.URL+"/api/clients", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+panel.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return ProvisionResponse{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return ProvisionResponse{}, fmt.Errorf("xui create failed status=%d", res.StatusCode)
	}
	var out createResp
	if err = json.NewDecoder(res.Body).Decode(&out); err != nil {
		return ProvisionResponse{}, err
	}
	return ProvisionResponse{UUID: out.UUID, ExpiresAt: time.Unix(out.ExpiresAt, 0)}, nil
}
