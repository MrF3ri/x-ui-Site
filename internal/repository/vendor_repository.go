package repository

import (
	"database/sql"
	"garudapanel/internal/models"
)

type VendorRepository struct{ db *sql.DB }

func NewVendorRepository(db *sql.DB) *VendorRepository { return &VendorRepository{db: db} }
func (r *VendorRepository) Create(ownerID int64, name, slug string) error {
	_, err := r.db.Exec(`INSERT INTO vendors(owner_user_id,name,slug,status) VALUES($1,$2,$3,'active')`, ownerID, name, slug)
	return err
}
func (r *VendorRepository) ListByOwner(ownerID int64) ([]models.Vendor, error) {
	rows, err := r.db.Query(`SELECT id,owner_user_id,name,slug,created_at FROM vendors WHERE owner_user_id=$1 AND deleted_at IS NULL`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Vendor{}
	for rows.Next() {
		var v models.Vendor
		if err := rows.Scan(&v.ID, &v.OwnerUserID, &v.Name, &v.Slug, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
