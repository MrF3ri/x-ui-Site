package repository

import "database/sql"

type WalletRepository struct{ db *sql.DB }
func NewWalletRepository(db *sql.DB) *WalletRepository { return &WalletRepository{db: db} }
func (r *WalletRepository) EnsureForUser(userID int64) error { _,err:=r.db.Exec(`INSERT INTO wallets(user_id,balance) VALUES($1,0) ON CONFLICT DO NOTHING`,userID); return err }
func (r *WalletRepository) Balance(userID int64) (int64,error) { var b int64; err:=r.db.QueryRow(`SELECT balance FROM wallets WHERE user_id=$1 AND deleted_at IS NULL`,userID).Scan(&b); return b,err }
func (r *WalletRepository) ApplyTransaction(userID int64, amount int64, typ string) error {
	tx, err := r.db.Begin(); if err != nil { return err }
	defer tx.Rollback()
	var walletID int64
	if err = tx.QueryRow(`SELECT id FROM wallets WHERE user_id=$1 FOR UPDATE`, userID).Scan(&walletID); err != nil { return err }
	if _, err = tx.Exec(`UPDATE wallets SET balance=balance+$1, updated_at=now() WHERE id=$2`, amount, walletID); err != nil { return err }
	if _, err = tx.Exec(`INSERT INTO wallet_transactions(wallet_id,user_id,amount,type) VALUES($1,$2,$3,$4)`, walletID,userID,amount,typ); err != nil { return err }
	return tx.Commit()
}
