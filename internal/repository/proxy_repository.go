package repository

import (
	"context"
	"database/sql"
	"time"
)

// ProxyServiceRecord represents a provisioned VPN service row.
type ProxyServiceRecord struct {
	ID                 int64
	VendorID           int64
	UserID             int64
	OrderID            *int64
	PanelID            *int64
	UUID               string
	Protocol           string
	SubscriptionURL    string
	QRPayload          string
	ConfigPayload      string
	Status             string
	ExpiresAt          *time.Time
	TrafficUsedGB      int
	TrafficLimitGB     int
	DurationDays       int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ProxyServiceRepository struct{ db *sql.DB }

func NewProxyServiceRepository(db *sql.DB) *ProxyServiceRepository {
	return &ProxyServiceRepository{db: db}
}

// CreateTx inserts a new proxy service inside a transaction.
func (r *ProxyServiceRepository) CreateTx(ctx context.Context, tx *sql.Tx, rec ProxyServiceRecord) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx,
		`INSERT INTO proxy_services
		 (vendor_id, user_id, order_id, panel_id, uuid, protocol,
		  subscription_url, qr_payload, config_payload, status,
		  expires_at, traffic_limit_gb, duration_days)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		 RETURNING id`,
		rec.VendorID, rec.UserID, rec.OrderID, rec.PanelID,
		rec.UUID, rec.Protocol,
		rec.SubscriptionURL, rec.QRPayload, rec.ConfigPayload,
		rec.Status, rec.ExpiresAt, rec.TrafficLimitGB, rec.DurationDays,
	).Scan(&id)
	return id, err
}

// ByID returns a proxy service enforcing vendor isolation.
func (r *ProxyServiceRepository) ByID(ctx context.Context, vendorID, id int64) (ProxyServiceRecord, error) {
	var rec ProxyServiceRecord
	err := r.db.QueryRowContext(ctx,
		`SELECT id, vendor_id, user_id, order_id, panel_id,
		        uuid, protocol, subscription_url, qr_payload,
		        config_payload, status, expires_at,
		        traffic_used_gb, traffic_limit_gb, duration_days,
		        created_at, updated_at
		 FROM proxy_services
		 WHERE id=$1 AND vendor_id=$2 AND deleted_at IS NULL`,
		id, vendorID,
	).Scan(
		&rec.ID, &rec.VendorID, &rec.UserID, &rec.OrderID, &rec.PanelID,
		&rec.UUID, &rec.Protocol, &rec.SubscriptionURL, &rec.QRPayload,
		&rec.ConfigPayload, &rec.Status, &rec.ExpiresAt,
		&rec.TrafficUsedGB, &rec.TrafficLimitGB, &rec.DurationDays,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	return rec, err
}

// ByUUID returns a proxy service by UUID (vendor-isolated).
func (r *ProxyServiceRepository) ByUUID(ctx context.Context, vendorID int64, uuid string) (ProxyServiceRecord, error) {
	var rec ProxyServiceRecord
	err := r.db.QueryRowContext(ctx,
		`SELECT id, vendor_id, user_id, order_id, panel_id,
		        uuid, protocol, subscription_url, qr_payload,
		        config_payload, status, expires_at,
		        traffic_used_gb, traffic_limit_gb, duration_days,
		        created_at, updated_at
		 FROM proxy_services
		 WHERE uuid=$1 AND vendor_id=$2 AND deleted_at IS NULL`,
		uuid, vendorID,
	).Scan(
		&rec.ID, &rec.VendorID, &rec.UserID, &rec.OrderID, &rec.PanelID,
		&rec.UUID, &rec.Protocol, &rec.SubscriptionURL, &rec.QRPayload,
		&rec.ConfigPayload, &rec.Status, &rec.ExpiresAt,
		&rec.TrafficUsedGB, &rec.TrafficLimitGB, &rec.DurationDays,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	return rec, err
}

// ListByUser returns all proxy services for a user under a vendor.
func (r *ProxyServiceRepository) ListByUser(ctx context.Context, vendorID, userID int64) ([]ProxyServiceRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, vendor_id, user_id, order_id, panel_id,
		        uuid, protocol, subscription_url, qr_payload,
		        config_payload, status, expires_at,
		        traffic_used_gb, traffic_limit_gb, duration_days,
		        created_at, updated_at
		 FROM proxy_services
		 WHERE vendor_id=$1 AND user_id=$2 AND deleted_at IS NULL
		 ORDER BY id DESC`,
		vendorID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProxyServiceRecord
	for rows.Next() {
		var rec ProxyServiceRecord
		if err := rows.Scan(
			&rec.ID, &rec.VendorID, &rec.UserID, &rec.OrderID, &rec.PanelID,
			&rec.UUID, &rec.Protocol, &rec.SubscriptionURL, &rec.QRPayload,
			&rec.ConfigPayload, &rec.Status, &rec.ExpiresAt,
			&rec.TrafficUsedGB, &rec.TrafficLimitGB, &rec.DurationDays,
			&rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// UpdateStatus changes the status of a proxy service.
func (r *ProxyServiceRepository) UpdateStatus(ctx context.Context, vendorID, id int64, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE proxy_services SET status=$1, updated_at=now()
		 WHERE id=$2 AND vendor_id=$3`,
		status, id, vendorID,
	)
	return err
}

// Suspend soft-suspends an active service.
func (r *ProxyServiceRepository) Suspend(ctx context.Context, vendorID, id int64) error {
	return r.UpdateStatus(ctx, vendorID, id, "suspended")
}

// Resume re-activates a suspended service.
func (r *ProxyServiceRepository) Resume(ctx context.Context, vendorID, id int64) error {
	return r.UpdateStatus(ctx, vendorID, id, "active")
}

// MarkExpired marks services whose expiry has passed.
func (r *ProxyServiceRepository) MarkExpired(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE proxy_services
		 SET status='expired', updated_at=now()
		 WHERE status='active' AND expires_at < now() AND deleted_at IS NULL`,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// XUIPanelRepository manages vendor xui panels.
type XUIPanelRepository struct{ db *sql.DB }

func NewXUIPanelRepository(db *sql.DB) *XUIPanelRepository {
	return &XUIPanelRepository{db: db}
}

type XUIPanelRecord struct {
	ID          int64
	VendorID    int64
	Name        string
	URL         string
	Token       string
	InboundID   int64
	IsActive    bool
	Health      string
	LastChecked *time.Time
}

func (r *XUIPanelRepository) Create(ctx context.Context, rec XUIPanelRecord) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO xui_panels(vendor_id, name, url, token, inbound_id, is_active)
		 VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		rec.VendorID, rec.Name, rec.URL, rec.Token, rec.InboundID, rec.IsActive,
	).Scan(&id)
	return id, err
}

func (r *XUIPanelRepository) FirstActive(ctx context.Context, vendorID int64) (XUIPanelRecord, error) {
	var rec XUIPanelRecord
	err := r.db.QueryRowContext(ctx,
		`SELECT id, vendor_id, name, url, token, inbound_id, is_active, health
		 FROM xui_panels
		 WHERE vendor_id=$1 AND is_active=TRUE AND deleted_at IS NULL
		 ORDER BY id LIMIT 1`,
		vendorID,
	).Scan(&rec.ID, &rec.VendorID, &rec.Name, &rec.URL, &rec.Token,
		&rec.InboundID, &rec.IsActive, &rec.Health)
	return rec, err
}

func (r *XUIPanelRepository) UpdateHealth(ctx context.Context, id int64, health string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE xui_panels SET health=$1, last_checked=now(), updated_at=now() WHERE id=$2`,
		health, id,
	)
	return err
}

func (r *XUIPanelRepository) ListByVendor(ctx context.Context, vendorID int64) ([]XUIPanelRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, vendor_id, name, url, token, inbound_id, is_active, health
		 FROM xui_panels WHERE vendor_id=$1 AND deleted_at IS NULL ORDER BY id`,
		vendorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []XUIPanelRecord
	for rows.Next() {
		var rec XUIPanelRecord
		if err := rows.Scan(&rec.ID, &rec.VendorID, &rec.Name, &rec.URL,
			&rec.Token, &rec.InboundID, &rec.IsActive, &rec.Health); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}
