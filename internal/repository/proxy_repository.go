package repository

import (
	"context"
	"database/sql"
	"time"

	"garudapanel/internal/security"
)

// ProxyServiceRecord represents a provisioned VPN service row.
type ProxyServiceRecord struct {
	ID              int64
	VendorID        int64
	UserID          int64
	OrderID         *int64
	PanelID         *int64
	UUID            string
	Protocol        string
	SubscriptionURL string
	QRPayload       string
	ConfigPayload   string
	Status          string
	ExpiresAt       *time.Time
	TrafficUsedGB   int
	TrafficLimitGB  int
	DurationDays    int
	CreatedAt       time.Time
	UpdatedAt       time.Time
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

// ByIDForUserTx returns a proxy service by id and user_id within a transaction (FOR UPDATE).
func (r *ProxyServiceRepository) ByIDForUserTx(ctx context.Context, tx *sql.Tx, id, userID int64) (ProxyServiceRecord, error) {
	var rec ProxyServiceRecord
	err := tx.QueryRowContext(ctx,
		`SELECT id, vendor_id, user_id, order_id, panel_id,
				uuid, protocol, subscription_url, qr_payload,
				config_payload, status, expires_at,
				traffic_used_gb, traffic_limit_gb, duration_days,
				created_at, updated_at
		 FROM proxy_services
		 WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL FOR UPDATE`,
		id, userID,
	).Scan(
		&rec.ID, &rec.VendorID, &rec.UserID, &rec.OrderID, &rec.PanelID,
		&rec.UUID, &rec.Protocol, &rec.SubscriptionURL, &rec.QRPayload,
		&rec.ConfigPayload, &rec.Status, &rec.ExpiresAt,
		&rec.TrafficUsedGB, &rec.TrafficLimitGB, &rec.DurationDays,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	return rec, err
}

// ListAllByUser returns all proxy services for a given user across vendors.
func (r *ProxyServiceRepository) ListAllByUser(ctx context.Context, userID int64) ([]ProxyServiceRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, vendor_id, user_id, order_id, panel_id,
				uuid, protocol, subscription_url, qr_payload,
				config_payload, status, expires_at,
				traffic_used_gb, traffic_limit_gb, duration_days,
				created_at, updated_at
		 FROM proxy_services
		 WHERE user_id=$1 AND deleted_at IS NULL
		 ORDER BY id DESC`,
		userID,
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

// ByIDForUser returns a proxy service by id and user_id (non-transactional).
func (r *ProxyServiceRepository) ByIDForUser(ctx context.Context, id, userID int64) (ProxyServiceRecord, error) {
	var rec ProxyServiceRecord
	err := r.db.QueryRowContext(ctx,
		`SELECT id, vendor_id, user_id, order_id, panel_id,
				uuid, protocol, subscription_url, qr_payload,
				config_payload, status, expires_at,
				traffic_used_gb, traffic_limit_gb, duration_days,
				created_at, updated_at
		 FROM proxy_services
		 WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`,
		id, userID,
	).Scan(
		&rec.ID, &rec.VendorID, &rec.UserID, &rec.OrderID, &rec.PanelID,
		&rec.UUID, &rec.Protocol, &rec.SubscriptionURL, &rec.QRPayload,
		&rec.ConfigPayload, &rec.Status, &rec.ExpiresAt,
		&rec.TrafficUsedGB, &rec.TrafficLimitGB, &rec.DurationDays,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	return rec, err
}

// ExtendExpiryTx extends the expiry of a proxy service by given days inside a transaction.
func (r *ProxyServiceRepository) ExtendExpiryTx(ctx context.Context, tx *sql.Tx, id int64, days int) error {
	// If expires_at is null or in the past, set to now() + days; else add days to existing expiry.
	_, err := tx.ExecContext(ctx,
		`UPDATE proxy_services
		 SET expires_at = CASE WHEN expires_at IS NULL OR expires_at < now()
			 THEN now() + ($1 || ' days')::interval
			 ELSE expires_at + ($1 || ' days')::interval END,
			 updated_at = now()
		 WHERE id=$2`,
		days, id,
	)
	return err
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
type XUIPanelRepository struct {
	db         *sql.DB
	passphrase string
}

func NewXUIPanelRepository(db *sql.DB, passphrase string) *XUIPanelRepository {
	return &XUIPanelRepository{db: db, passphrase: passphrase}
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
	enc, err := security.Encrypt(r.passphrase, rec.Token)
	if err != nil {
		return 0, err
	}
	err = r.db.QueryRowContext(ctx,
		`INSERT INTO xui_panels(vendor_id, name, url, token, inbound_id, is_active)
		 VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		rec.VendorID, rec.Name, rec.URL, enc, rec.InboundID, rec.IsActive,
	).Scan(&id)
	return id, err
}

func (r *XUIPanelRepository) FirstActive(ctx context.Context, vendorID int64) (XUIPanelRecord, error) {
	var rec XUIPanelRecord
	var encToken string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, vendor_id, name, url, token, inbound_id, is_active, health
		 FROM xui_panels
		 WHERE vendor_id=$1 AND is_active=TRUE AND deleted_at IS NULL
		 ORDER BY id LIMIT 1`,
		vendorID,
	).Scan(&rec.ID, &rec.VendorID, &rec.Name, &rec.URL, &encToken,
		&rec.InboundID, &rec.IsActive, &rec.Health)
	if err != nil {
		return rec, err
	}
	// decrypt token
	tok, err := security.Decrypt(r.passphrase, encToken)
	if err != nil {
		return rec, err
	}
	rec.Token = tok
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
		var encToken string
		if err := rows.Scan(&rec.ID, &rec.VendorID, &rec.Name, &rec.URL,
			&encToken, &rec.InboundID, &rec.IsActive, &rec.Health); err != nil {
			return nil, err
		}
		tok, err := security.Decrypt(r.passphrase, encToken)
		if err != nil {
			return nil, err
		}
		rec.Token = tok
		out = append(out, rec)
	}
	return out, rows.Err()
}
