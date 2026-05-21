package xui

import "time"

type Panel struct { VendorID int64; Name, URL, Token string }
type ProvisionRequest struct { VendorID, UserID int64; Protocol string; DurationDays int; TrafficGB int }
type ProvisionResponse struct { UUID string; ExpiresAt time.Time }

type Adapter interface { Provision(panel Panel, req ProvisionRequest) (ProvisionResponse, error); Renew(panel Panel, uuid string, addDays int, addGB int) (ProvisionResponse, error) }

type MockAdapter struct{}
func (m MockAdapter) Provision(_ Panel, req ProvisionRequest) (ProvisionResponse, error) { return ProvisionResponse{UUID: "uuid-mock", ExpiresAt: time.Now().Add(time.Duration(req.DurationDays)*24*time.Hour)}, nil }
func (m MockAdapter) Renew(_ Panel, uuid string, addDays int, _ int) (ProvisionResponse, error) { return ProvisionResponse{UUID: uuid, ExpiresAt: time.Now().Add(time.Duration(addDays)*24*time.Hour)}, nil }
