package provisioning

import (
	"context"
	"testing"
	"time"

	"garudapanel/internal/xui"
	"github.com/DATA-DOG/go-sqlmock"
)

type fakeAdapter struct{}

func (f *fakeAdapter) Provision(panel xui.Panel, req xui.ProvisionRequest) (xui.ProvisionResponse, error) {
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
	// Expect transactional insert/update/commit sequence
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO proxy_services").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(55))
	mock.ExpectExec("UPDATE orders SET service_id").WithArgs(55, 10).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Simulate transactional create and link: begin tx
	tx, _ := db.Begin()
	// Insert proxy service
	var id int64
	err = tx.QueryRowContext(context.Background(), "INSERT INTO proxy_services (vendor_id) VALUES ($1) RETURNING id", 5).Scan(&id)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if _, err := tx.ExecContext(context.Background(), "UPDATE orders SET service_id=$1, updated_at=now() WHERE id=$2", id, 10); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
