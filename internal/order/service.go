package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"garudapanel/internal/models"
	"garudapanel/internal/notification"
	"garudapanel/internal/repository"
)

// LifecycleState constants.
const (
	StateApproved     = "approved"
	StatePending      = "pending"
	StateProvisioning = "provisioning"
	StateActive       = "active"
	StateSuspended    = "suspended"
	StateExpired      = "expired"
	StateFailed       = "failed"
)

// Service orchestrates order creation with idempotency and transactional wallet deduction.
type Service struct {
	db          *sql.DB
	orders      *repository.OrderRepository
	idempotency *repository.IdempotencyRepository
	jobs        *repository.ProvisioningJobRepository
	wallet      *repository.WalletTxRepository
	catalog     *repository.CatalogPriceRepository
	notifier    *notification.Hub
	proxy       *repository.ProxyServiceRepository
}

func NewService(
	db *sql.DB,
	orders *repository.OrderRepository,
	idempotency *repository.IdempotencyRepository,
	jobs *repository.ProvisioningJobRepository,
	wallet *repository.WalletTxRepository,
	catalog *repository.CatalogPriceRepository,
	notifier *notification.Hub,
	proxy *repository.ProxyServiceRepository,
) *Service {
	return &Service{db: db, orders: orders, idempotency: idempotency, jobs: jobs, wallet: wallet, catalog: catalog, notifier: notifier, proxy: proxy}
}

// Renew handles renewal of an existing proxy service owned by userID.
// It performs wallet deduction, creates an order, enqueues provisioning, and extends expiry within a single transaction.
func (s *Service) Renew(ctx context.Context, userID, serviceID int64) (CreateResponse, error) {
	// Begin transaction
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return CreateResponse{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Lock service row for update and ensure ownership
	svcRec, err := s.proxy.ByIDForUserTx(ctx, tx, serviceID, userID)
	if err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, err
	}
	if svcRec.OrderID == nil {
		_ = tx.Rollback()
		return CreateResponse{}, errors.New("original order not found for service")
	}

	// Load original order to find catalog id/price
	origOrder, err := s.orders.ByID(ctx, svcRec.VendorID, *svcRec.OrderID)
	if err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, err
	}

	// Get catalog price
	cat, err := s.catalog.Get(ctx, svcRec.VendorID, origOrder.CatalogID)
	if err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, err
	}

	// Reserve idempotency key
	key := fmt.Sprintf("renew:%d:%d", serviceID, time.Now().UnixNano())
	if err = s.idempotency.ReserveTx(ctx, tx, svcRec.VendorID, userID, key); err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, err
	}

	// Deduct wallet
	if err = s.wallet.DeductTx(ctx, tx, userID, cat.Price); err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, err
	}

	// Create order record
	ikey := key
	orderID, err := s.orders.CreateTx(ctx, tx, models.Order{
		VendorID:       svcRec.VendorID,
		UserID:         userID,
		CatalogID:      origOrder.CatalogID,
		Amount:         cat.Price,
		Status:         StateApproved,
		LifecycleState: StatePending,
		IdempotencyKey: &ikey,
	})
	if err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, err
	}

	// Link idempotency to new order
	if err = s.idempotency.LinkOrderTx(ctx, tx, svcRec.VendorID, userID, key, orderID); err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, err
	}

	// Enqueue provisioning job
	if err = s.jobs.EnqueueTx(ctx, tx, svcRec.VendorID, orderID); err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, err
	}

	// Extend proxy service expiry
	if err = s.proxy.ExtendExpiryTx(ctx, tx, serviceID, svcRec.DurationDays); err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, err
	}

	if err = tx.Commit(); err != nil {
		return CreateResponse{}, err
	}

	// Notify
	s.notifier.Notify("order.renewed", map[string]any{"order_id": orderID, "service_id": serviceID, "user_id": userID})
	return CreateResponse{OrderID: orderID, Status: StateApproved, Lifecycle: StatePending}, nil
}

// CreateRequest is the input for order creation.
type CreateRequest struct {
	VendorID       int64
	UserID         int64
	CatalogID      int64
	IdempotencyKey string // caller-supplied, required
}

// CreateResponse is returned on success.
type CreateResponse struct {
	OrderID   int64  `json:"order_id"`
	Status    string `json:"status"`
	Lifecycle string `json:"lifecycle_state"`
	Duplicate bool   `json:"duplicate,omitempty"` // true when idempotency key was already processed
}

// Create creates an order atomically:
//  1. Checks for existing idempotency key (returns existing order if found).
//  2. Validates catalog item belongs to vendor and is active.
//  3. Opens a DB transaction.
//  4. Locks wallet row FOR UPDATE and checks balance.
//  5. Deducts balance + inserts ledger entry.
//  6. Inserts order row.
//  7. Inserts idempotency key row.
//  8. Enqueues provisioning job.
//  9. Commits.
//
// 10. Emits notification event.
func (s *Service) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	if req.IdempotencyKey == "" {
		return CreateResponse{}, errors.New("idempotency_key is required")
	}

	// ── Fast path: idempotency hit before touching a transaction ──────────
	existingID, err := s.idempotency.Resolve(ctx, req.VendorID, req.UserID, req.IdempotencyKey)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("idempotency resolve: %w", err)
	}
	if existingID != 0 {
		o, err := s.orders.ByID(ctx, req.VendorID, existingID)
		if err != nil {
			return CreateResponse{}, fmt.Errorf("load existing order: %w", err)
		}
		return CreateResponse{OrderID: o.ID, Status: o.Status, Lifecycle: o.LifecycleState, Duplicate: true}, nil
	}

	// ── Validate catalog item ─────────────────────────────────────────────
	cat, err := s.catalog.Get(ctx, req.VendorID, req.CatalogID)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("catalog lookup: %w", err)
	}
	if cat.VendorID != req.VendorID {
		return CreateResponse{}, errors.New("vendor isolation violation")
	}
	if !cat.IsActive {
		return CreateResponse{}, errors.New("catalog item is not active")
	}

	// ── Transactional block ───────────────────────────────────────────────
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return CreateResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("order: rollback error: %v", rbErr)
			}
		}
	}()

	// Reserve idempotency key (INSERT — will fail on duplicate key race).
	if err = s.idempotency.ReserveTx(ctx, tx, req.VendorID, req.UserID, req.IdempotencyKey); err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, fmt.Errorf("idempotency reserve (possible duplicate race): %w", err)
	}

	// Wallet deduction (locks wallet row FOR UPDATE internally).
	if err = s.wallet.DeductTx(ctx, tx, req.UserID, cat.Price); err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, fmt.Errorf("wallet deduction: %w", err)
	}

	// Create order.
	ikey := req.IdempotencyKey
	orderID, err := s.orders.CreateTx(ctx, tx, models.Order{
		VendorID:       req.VendorID,
		UserID:         req.UserID,
		CatalogID:      req.CatalogID,
		Amount:         cat.Price,
		Status:         StateApproved,
		LifecycleState: StatePending,
		IdempotencyKey: &ikey,
	})
	if err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, fmt.Errorf("order insert: %w", err)
	}

	// Link idempotency key → order.
	if err = s.idempotency.LinkOrderTx(ctx, tx, req.VendorID, req.UserID, req.IdempotencyKey, orderID); err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, fmt.Errorf("idempotency link: %w", err)
	}

	// Enqueue provisioning job.
	if err = s.jobs.EnqueueTx(ctx, tx, req.VendorID, orderID); err != nil {
		_ = tx.Rollback()
		return CreateResponse{}, fmt.Errorf("enqueue provisioning: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return CreateResponse{}, fmt.Errorf("commit: %w", err)
	}

	// Emit notification (best-effort, non-blocking).
	s.notifier.Notify("order.approved", map[string]any{
		"order_id":  orderID,
		"vendor_id": req.VendorID,
		"user_id":   req.UserID,
	})

	return CreateResponse{OrderID: orderID, Status: StateApproved, Lifecycle: StatePending}, nil
}
