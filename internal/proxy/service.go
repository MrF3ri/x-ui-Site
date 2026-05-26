package proxy

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"garudapanel/internal/notification"
	"garudapanel/internal/repository"
	"garudapanel/internal/xui"
)

const (
	StatusQueued       = "queued"
	StatusProvisioning = "provisioning"
	StatusActive       = "active"
	StatusFailed       = "failed"
	StatusSuspended    = "suspended"
	StatusExpired      = "expired"
)

// Service handles proxy service lifecycle.
type Service struct {
	db        *sql.DB
	proxies   *repository.ProxyServiceRepository
	panels    *repository.XUIPanelRepository
	jobs      *repository.ProvisioningJobRepository
	orders    *repository.OrderRepository
	notifier  *notification.Hub
	xuiClient *xui.Client
}

func NewService(
	db *sql.DB,
	proxies *repository.ProxyServiceRepository,
	panels *repository.XUIPanelRepository,
	jobs *repository.ProvisioningJobRepository,
	orders *repository.OrderRepository,
	notifier *notification.Hub,
) *Service {
	return &Service{
		db:       db,
		proxies:  proxies,
		panels:   panels,
		jobs:     jobs,
		orders:   orders,
		notifier: notifier,
	}
}

// ProvisionRequest is the input for manual provisioning.
type ProvisionRequest struct {
	VendorID       int64
	UserID         int64
	OrderID        *int64
	Protocol       string
	TrafficLimitGB int
	DurationDays   int
}

// ProvisionResponse is the result of a successful provision.
type ProvisionResponse struct {
	ServiceID       int64     `json:"service_id"`
	UUID            string    `json:"uuid"`
	SubscriptionURL string    `json:"subscription_url"`
	ExpiresAt       time.Time `json:"expires_at"`
	Status          string    `json:"status"`
}

// Provision creates a proxy service record and calls the xui panel.
func (s *Service) Provision(ctx context.Context, req ProvisionRequest) (ProvisionResponse, error) {
	// Find active panel for vendor
	panel, err := s.panels.FirstActive(ctx, req.VendorID)
	if err != nil {
		return ProvisionResponse{}, fmt.Errorf("no active panel for vendor %d: %w", req.VendorID, err)
	}

	uuid := generateUUID()
	expiresAt := time.Now().Add(time.Duration(req.DurationDays) * 24 * time.Hour)
	subURL := buildSubscriptionURL(panel.URL, uuid)

	rec := repository.ProxyServiceRecord{
		VendorID:        req.VendorID,
		UserID:          req.UserID,
		OrderID:         req.OrderID,
		PanelID:         &panel.ID,
		UUID:            uuid,
		Protocol:        req.Protocol,
		SubscriptionURL: subURL,
		QRPayload:       subURL,
		ConfigPayload:   "",
		Status:          StatusProvisioning,
		ExpiresAt:       &expiresAt,
		TrafficLimitGB:  req.TrafficLimitGB,
		DurationDays:    req.DurationDays,
	}

	// Insert service record in transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ProvisionResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	serviceID, err := s.proxies.CreateTx(ctx, tx, rec)
	if err != nil {
		return ProvisionResponse{}, fmt.Errorf("create proxy service: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return ProvisionResponse{}, fmt.Errorf("commit: %w", err)
	}

	// Call xui panel (outside tx — network call)
	xuiPanel := xui.Panel{
		VendorID:  panel.VendorID,
		Name:      panel.Name,
		URL:       panel.URL,
		Token:     panel.Token,
		InboundID: panel.InboundID,
	}
	xuiClient := xui.NewClient(panel.URL, panel.Token)
	adapter := xui.NewAdapter(xuiClient)

	_, xuiErr := adapter.Provision(xuiPanel, xui.ProvisionRequest{
		VendorID:     req.VendorID,
		UserID:       req.UserID,
		Protocol:     req.Protocol,
		UUID:         uuid,
		DurationDays: req.DurationDays,
		TrafficGB:    req.TrafficLimitGB,
	})

	finalStatus := StatusActive
	if xuiErr != nil {
		log.Printf("proxy: xui provision failed service_id=%d: %v", serviceID, xuiErr)
		finalStatus = StatusFailed
	}

	// Update status based on xui result
	_ = s.proxies.UpdateStatus(ctx, req.VendorID, serviceID, finalStatus)

	if xuiErr != nil {
		return ProvisionResponse{}, fmt.Errorf("xui provision: %w", xuiErr)
	}

	s.notifier.Notify("service.provisioned", map[string]any{
		"service_id": serviceID,
		"vendor_id":  req.VendorID,
		"user_id":    req.UserID,
		"uuid":       uuid,
	})

	return ProvisionResponse{
		ServiceID:       serviceID,
		UUID:            uuid,
		SubscriptionURL: subURL,
		ExpiresAt:       expiresAt,
		Status:          finalStatus,
	}, nil
}

// Suspend deactivates a service on the panel and marks it suspended.
func (s *Service) Suspend(ctx context.Context, vendorID, serviceID int64) error {
	rec, err := s.proxies.ByID(ctx, vendorID, serviceID)
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}
	if rec.Status == StatusSuspended {
		return errors.New("service already suspended")
	}
	if err = s.proxies.Suspend(ctx, vendorID, serviceID); err != nil {
		return err
	}
	s.notifier.Notify("service.suspended", map[string]any{
		"service_id": serviceID, "vendor_id": vendorID,
	})
	return nil
}

// Resume re-activates a suspended service.
func (s *Service) Resume(ctx context.Context, vendorID, serviceID int64) error {
	rec, err := s.proxies.ByID(ctx, vendorID, serviceID)
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}
	if rec.Status != StatusSuspended {
		return errors.New("service is not suspended")
	}
	if err = s.proxies.Resume(ctx, vendorID, serviceID); err != nil {
		return err
	}
	s.notifier.Notify("service.resumed", map[string]any{
		"service_id": serviceID, "vendor_id": vendorID,
	})
	return nil
}

// RunExpiryJob marks expired services — call this on a ticker.
func (s *Service) RunExpiryJob(ctx context.Context) {
	n, err := s.proxies.MarkExpired(ctx)
	if err != nil {
		log.Printf("expiry job error: %v", err)
		return
	}
	if n > 0 {
		log.Printf("expiry job: marked %d services as expired", n)
	}
}

// GetByID returns a proxy service (vendor-isolated).
func (s *Service) GetByID(ctx context.Context, vendorID, serviceID int64) (repository.ProxyServiceRecord, error) {
	return s.proxies.ByID(ctx, vendorID, serviceID)
}

// ListByUser returns all proxy services for a user under a vendor.
func (s *Service) ListByUser(ctx context.Context, vendorID, userID int64) ([]repository.ProxyServiceRecord, error) {
	return s.proxies.ListByUser(ctx, vendorID, userID)
}

// --- helpers ---

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:]),
	)
}

func buildSubscriptionURL(panelURL, uuid string) string {
	return fmt.Sprintf("%s/sub/%s", panelURL, uuid)
}
