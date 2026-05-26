package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"garudapanel/internal/models"
)

// OrderRepository handles all order persistence.
type OrderRepository struct{ db *sql.DB }

func NewOrderRepository(db *sql.DB) *OrderRepository { return &OrderRepository{db: db} }

// Create inserts a new order and returns its ID.
// Must be called inside a transaction for atomicity — pass the *sql.Tx cast as *sql.DB
// via the txDB helper, or use CreateTx directly.
func (r *OrderRepository) Create(ctx context.Context, o models.Order) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO orders(vendor_id, user_id, catalog_id, amount, status, lifecycle_state, idempotency_key)
		 VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		o.VendorID, o.UserID, o.CatalogID, o.Amount, o.Status, o.LifecycleState, o.IdempotencyKey,
	).Scan(&id)
	return id, err
}

// CreateTx inserts a new order within an existing transaction.
func (r *OrderRepository) CreateTx(ctx context.Context, tx *sql.Tx, o models.Order) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx,
		`INSERT INTO orders(vendor_id, user_id, catalog_id, amount, status, lifecycle_state, idempotency_key)
		 VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		o.VendorID, o.UserID, o.CatalogID, o.Amount, o.Status, o.LifecycleState, o.IdempotencyKey,
	).Scan(&id)
	return id, err
}

// LinkServiceTx sets the service_id for an order inside a transaction.
func (r *OrderRepository) LinkServiceTx(ctx context.Context, tx *sql.Tx, orderID, serviceID int64) error {
	_, err := tx.ExecContext(ctx, `UPDATE orders SET service_id=$1, updated_at=now() WHERE id=$2`, serviceID, orderID)
	return err
}

// UpdateStatus transitions order status and lifecycle_state.
func (r *OrderRepository) UpdateStatus(ctx context.Context, orderID int64, status, lifecycle string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE orders SET status=$1, lifecycle_state=$2, updated_at=now() WHERE id=$3`,
		status, lifecycle, orderID,
	)
	return err
}

// ByID returns an order, enforcing vendor isolation.
func (r *OrderRepository) ByID(ctx context.Context, vendorID, orderID int64) (models.Order, error) {
	var o models.Order
	err := r.db.QueryRowContext(ctx,
		`SELECT id, vendor_id, user_id, catalog_id, amount, status, lifecycle_state, created_at
		 FROM orders WHERE id=$1 AND vendor_id=$2 AND deleted_at IS NULL`,
		orderID, vendorID,
	).Scan(&o.ID, &o.VendorID, &o.UserID, &o.CatalogID, &o.Amount, &o.Status, &o.LifecycleState, &o.CreatedAt)
	return o, err
}

// ListByUser returns recent orders for a given user.
func (r *OrderRepository) ListByUser(ctx context.Context, userID int64, limit int) ([]models.Order, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, vendor_id, user_id, catalog_id, amount, status, lifecycle_state, created_at
		 FROM orders WHERE user_id=$1 AND deleted_at IS NULL ORDER BY id DESC LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Order{}
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.VendorID, &o.UserID, &o.CatalogID, &o.Amount, &o.Status, &o.LifecycleState, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// IdempotencyRepo handles order_idempotency_keys.
type IdempotencyRepository struct{ db *sql.DB }

func NewIdempotencyRepository(db *sql.DB) *IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

// Resolve returns an existing order ID for the key, or 0 if not found.
func (r *IdempotencyRepository) Resolve(ctx context.Context, vendorID, userID int64, key string) (int64, error) {
	var orderID sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT order_id FROM order_idempotency_keys
		 WHERE vendor_id=$1 AND user_id=$2 AND idempotency_key=$3`,
		vendorID, userID, key,
	).Scan(&orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if orderID.Valid {
		return orderID.Int64, nil
	}
	return 0, nil
}

// ReserveTx inserts the idempotency key row inside a transaction.
// Returns ErrConflict if the key already exists (race condition).
func (r *IdempotencyRepository) ReserveTx(ctx context.Context, tx *sql.Tx, vendorID, userID int64, key string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO order_idempotency_keys(vendor_id, user_id, idempotency_key)
		 VALUES($1,$2,$3)`,
		vendorID, userID, key,
	)
	return err
}

// LinkOrderTx links an order ID to the idempotency key row.
func (r *IdempotencyRepository) LinkOrderTx(ctx context.Context, tx *sql.Tx, vendorID, userID int64, key string, orderID int64) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE order_idempotency_keys SET order_id=$1
		 WHERE vendor_id=$2 AND user_id=$3 AND idempotency_key=$4`,
		orderID, vendorID, userID, key,
	)
	return err
}

// ProvisioningJobRepository manages the provisioning job queue.
type ProvisioningJobRepository struct{ db *sql.DB }

func NewProvisioningJobRepository(db *sql.DB) *ProvisioningJobRepository {
	return &ProvisioningJobRepository{db: db}
}

func (r *ProvisioningJobRepository) EnqueueTx(ctx context.Context, tx *sql.Tx, vendorID, orderID int64) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO provisioning_jobs(vendor_id, order_id, status) VALUES($1,$2,'pending')`,
		vendorID, orderID,
	)
	return err
}

func (r *ProvisioningJobRepository) ClaimNext(ctx context.Context) (jobID, orderID, vendorID int64, err error) {
		err = r.db.QueryRowContext(ctx,
				`UPDATE provisioning_jobs SET status='provisioning', updated_at=now()
				 WHERE id = (
					 SELECT id FROM provisioning_jobs
					 WHERE status='pending' AND dead_letter=FALSE AND deleted_at IS NULL
					 ORDER BY id
					 FOR UPDATE SKIP LOCKED
					 LIMIT 1
				 ) RETURNING id, order_id, vendor_id`,
		).Scan(&jobID, &orderID, &vendorID)
		if err != nil {
				return 0, 0, 0, err
		}
		return jobID, orderID, vendorID, nil
}

func (r *ProvisioningJobRepository) MarkDone(ctx context.Context, jobID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE provisioning_jobs SET status='done', updated_at=now() WHERE id=$1`, jobID)
	return err
}

func (r *ProvisioningJobRepository) MarkFailed(ctx context.Context, jobID int64, lastErr string, dead bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE provisioning_jobs
		 SET status='failed', retries=retries+1, last_error=$2, dead_letter=$3, updated_at=now()
		 WHERE id=$1`,
		jobID, lastErr, dead,
	)
	return err
}

// WalletTxRepository wraps wallet operations used inside order transactions.
type WalletTxRepository struct{ db *sql.DB }

func NewWalletTxRepository(db *sql.DB) *WalletTxRepository {
	return &WalletTxRepository{db: db}
}

// DeductTx atomically deducts amount from wallet within an existing transaction.
// Acquires a FOR UPDATE lock on the wallet row to prevent double-spend.
func (r *WalletTxRepository) DeductTx(ctx context.Context, tx *sql.Tx, userID, amount int64) error {
	var walletID, balance int64
	if err := tx.QueryRowContext(ctx,
		`SELECT id, balance FROM wallets WHERE user_id=$1 AND deleted_at IS NULL FOR UPDATE`,
		userID,
	).Scan(&walletID, &balance); err != nil {
		return err
	}
	if balance < amount {
		return errors.New("insufficient wallet balance")
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE wallets SET balance=balance-$1, updated_at=now() WHERE id=$2`,
		amount, walletID,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO wallet_transactions(wallet_id, user_id, amount, type) VALUES($1,$2,$3,'order_debit')`,
		walletID, userID, -amount,
	); err != nil {
		return err
	}
	return nil
}

// CatalogPriceRepository is a minimal read used by the order service.
type CatalogPriceRepository struct{ db *sql.DB }

func NewCatalogPriceRepository(db *sql.DB) *CatalogPriceRepository {
	return &CatalogPriceRepository{db: db}
}

type CatalogPrice struct {
	VendorID int64
	Price    int64
	IsActive bool
	ExpiresAt time.Time // zero if not applicable
}

func (r *CatalogPriceRepository) Get(ctx context.Context, vendorID, catalogID int64) (CatalogPrice, error) {
	var cp CatalogPrice
	err := r.db.QueryRowContext(ctx,
		`SELECT vendor_id, price_toman, is_active FROM catalog_items
		 WHERE id=$1 AND vendor_id=$2 AND deleted_at IS NULL`,
		catalogID, vendorID,
	).Scan(&cp.VendorID, &cp.Price, &cp.IsActive)
	return cp, err
}