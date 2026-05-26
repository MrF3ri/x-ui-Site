package provisioning

import (
	"context"
	"fmt"
	"testing"
	"time"

	"garudapanel/internal/eventbus"
	"garudapanel/internal/notification"
	"garudapanel/internal/repository"
	"garudapanel/internal/security"
	"garudapanel/internal/xui"
	"github.com/DATA-DOG/go-sqlmock"
)

type fakeAdapter struct {
	called bool
}

func (f *fakeAdapter) Provision(panel xui.Panel, req xui.ProvisionRequest) (xui.ProvisionResponse, error) {
	f.called = true
	if panel.Token != "token-abc123" {
		return xui.ProvisionResponse{}, fmt.Errorf("unexpected panel token: %s", panel.Token)
	}
	return xui.ProvisionResponse{UUID: "u-123", ExpiresAt: time.Now().Add(24 * time.Hour)}, nil
}

func (f *fakeAdapter) Renew(panel xui.Panel, uuid string, addDays int, addGB int) (xui.ProvisionResponse, error) {
	return xui.ProvisionResponse{}, nil
}

func TestWorkerV2_handleJob_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	passphrase := "0123456789abcdef0123456789abcdef"
	orderRepo := repository.NewOrderRepository(db)
	catalogRepo := repository.NewCatalogRepository(db)
	panelsRepo := repository.NewXUIPanelRepository(db, passphrase)
	proxyRepo := repository.NewProxyServiceRepository(db)
	jobsRepo := repository.NewProvisioningJobRepository(db)
	notifier := notification.NewHub(eventbus.New())

	orderID := int64(100)
	vendorID := int64(5)
	userID := int64(42)
	catalogID := int64(9)
	plainToken := "token-abc123"
	encToken, err := security.Encrypt(passphrase, plainToken)
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}

	orderRows := sqlmock.NewRows([]string{"id", "vendor_id", "user_id", "catalog_id", "amount", "status", "lifecycle_state", "created_at"}).
		AddRow(orderID, vendorID, userID, catalogID, int64(1000), "pending", "pending", time.Now())
	mock.ExpectQuery(`SELECT id, vendor_id, user_id, catalog_id, amount, status, lifecycle_state, created_at FROM orders WHERE id=\$1 AND vendor_id=\$2 AND deleted_at IS NULL`).
		WithArgs(orderID, vendorID).
		WillReturnRows(orderRows)

	panelRows := sqlmock.NewRows([]string{"id", "vendor_id", "name", "url", "token", "inbound_id", "is_active", "health"}).
		AddRow(int64(7), vendorID, "xui-panel", "https://xui.example.com", encToken, int64(88), true, "ok")
	mock.ExpectQuery(`SELECT id, vendor_id, name, url, token, inbound_id, is_active, health FROM xui_panels WHERE vendor_id=\$1 AND is_active=TRUE AND deleted_at IS NULL ORDER BY id LIMIT 1`).
		WithArgs(vendorID).
		WillReturnRows(panelRows)

	catalogRows := sqlmock.NewRows([]string{"vendor_id", "slug", "title", "description", "protocol", "inbound_id", "xui_node_id", "traffic_limit_gb", "duration_days", "price_toman", "is_active", "auto_provision", "renewal_enabled", "country_code", "stock_status"}).
		AddRow(vendorID, "example", "Example", "desc", "wireguard", int64(88), int64(99), int64(50), int64(30), int64(10000), true, true, true, "US", "in_stock")
	mock.ExpectQuery(`SELECT vendor_id,slug,title,description,protocol,inbound_id,xui_node_id,traffic_limit_gb,duration_days,price_toman,is_active,auto_provision,renewal_enabled,country_code,stock_status FROM catalog_items WHERE id=\$1 AND deleted_at IS NULL`).
		WithArgs(catalogID).
		WillReturnRows(catalogRows)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO proxy_services`).
		WithArgs(vendorID, userID, orderID, nil, sqlmock.AnyArg(), "wireguard", "https://xui.example.com/sub/u-123", "https://xui.example.com/sub/u-123", "", "active", sqlmock.AnyArg(), int64(50), int64(30)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(22)))
	mock.ExpectExec(`UPDATE orders SET service_id=\$1, updated_at=now\(\) WHERE id=\$2`).
		WithArgs(int64(22), orderID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	adapter := &fakeAdapter{}
	worker := NewWorkerV2(db, jobsRepo, orderRepo, catalogRepo, panelsRepo, proxyRepo, adapter, notifier)
	if err := worker.handleJob(context.Background(), int64(1), orderID, vendorID); err != nil {
		t.Fatalf("handleJob failed: %v", err)
	}
	if !adapter.called {
		t.Fatalf("adapter provision was not called")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
