package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestWalletTx_DeductTx_Insufficient(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	repo := NewWalletTxRepository(db)
	mock.ExpectBegin()
	tx, _ := db.Begin()
	// select returns small balance
	mock.ExpectQuery("SELECT id, balance FROM wallets WHERE user_id=\\$1 AND deleted_at IS NULL FOR UPDATE").WithArgs(42).WillReturnRows(sqlmock.NewRows([]string{"id", "balance"}).AddRow(7, 500))
	// call
	err = repo.DeductTx(context.Background(), tx, 42, 1000)
	if err == nil {
		t.Fatalf("expected insufficient balance error")
	}
	if err.Error() != "insufficient wallet balance" {
		t.Fatalf("unexpected error: %v", err)
	}
	// rollback expected by caller
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWalletTx_DeductTx_Sufficient(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	repo := NewWalletTxRepository(db)
	mock.ExpectBegin()
	tx, _ := db.Begin()
	// select returns sufficient balance
	mock.ExpectQuery("SELECT id, balance FROM wallets WHERE user_id=\\$1 AND deleted_at IS NULL FOR UPDATE").WithArgs(99).WillReturnRows(sqlmock.NewRows([]string{"id", "balance"}).AddRow(11, 2000))
	// update balance
	mock.ExpectExec("UPDATE wallets SET balance=balance-").WithArgs(500, 11).WillReturnResult(sqlmock.NewResult(1, 1))
	// insert transaction
	mock.ExpectExec("INSERT INTO wallet_transactions\\(wallet_id, user_id, amount, type\\) VALUES\\(\\$1,\\$2,\\$3,'order_debit'\\)").WithArgs(11, 99, -500).WillReturnResult(sqlmock.NewResult(1, 1))
	// call
	if err := repo.DeductTx(context.Background(), tx, 99, 500); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestIdempotency_ReserveTx_Conflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	repo := NewIdempotencyRepository(db)
	mock.ExpectBegin()
	tx, _ := db.Begin()
	mock.ExpectExec("INSERT INTO order_idempotency_keys\\(vendor_id, user_id, idempotency_key\\)").WithArgs(1, 2, "k").WillReturnError(errors.New("unique violation"))
	if err := repo.ReserveTx(context.Background(), tx, 1, 2, "k"); err == nil {
		t.Fatalf("expected error on duplicate")
	}
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
