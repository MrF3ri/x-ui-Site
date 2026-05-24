package models

import "time"

type ServiceCatalogItem struct {
	ID           int64     `json:"id"`
	VendorID     int64     `json:"vendor_id"`
	Name         string    `json:"name"`
	Protocol     string    `json:"protocol"`
	DurationDays int       `json:"duration_days"`
	TrafficGB    int       `json:"traffic_gb"`
	Price        int64     `json:"price"`
	CreatedAt    time.Time `json:"created_at"`
}

type Order struct {
	ID             int64     `json:"id"`
	VendorID       int64     `json:"vendor_id"`
	UserID         int64     `json:"user_id"`
	CatalogID      int64     `json:"catalog_id"`
	ServiceID      *int64    `json:"service_id,omitempty"`
	Amount         int64     `json:"amount"`
	Status         string    `json:"status"`
	LifecycleState string    `json:"lifecycle_state"`
	IdempotencyKey *string   `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type ProxyService struct {
	ID        int64     `json:"id"`
	VendorID  int64     `json:"vendor_id"`
	UserID    int64     `json:"user_id"`
	UUID      string    `json:"uuid"`
	Protocol  string    `json:"protocol"`
	ExpiresAt time.Time `json:"expires_at"`
	TrafficGB int       `json:"traffic_gb"`
	CreatedAt time.Time `json:"created_at"`
}

type Receipt struct {
	ID        int64     `json:"id"`
	OrderID   int64     `json:"order_id"`
	VendorID  int64     `json:"vendor_id"`
	UserID    int64     `json:"user_id"`
	ObjectKey string    `json:"object_key"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}