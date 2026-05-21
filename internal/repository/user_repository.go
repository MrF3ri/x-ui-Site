package repository

import (
	"database/sql"
	"garudapanel/internal/models"
)

type UserRepository struct{ db *sql.DB }
func NewUserRepository(db *sql.DB) *UserRepository { return &UserRepository{db: db} }
func (r *UserRepository) Create(email, hash, role string) (int64, error) { var id int64; err:=r.db.QueryRow(`INSERT INTO users(email,password_hash,role) VALUES($1,$2,$3) RETURNING id`, email,hash,role).Scan(&id); return id, err }
func (r *UserRepository) ByEmail(email string) (models.User, error) { var u models.User; err:=r.db.QueryRow(`SELECT id,vendor_id,email,password_hash,role,created_at FROM users WHERE email=$1 AND deleted_at IS NULL`, email).Scan(&u.ID,&u.VendorID,&u.Email,&u.PasswordHash,&u.Role,&u.CreatedAt); return u, err }
