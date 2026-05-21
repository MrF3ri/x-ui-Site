package repository

import (
	"database/sql"
	"garudapanel/internal/models"
)

type CatalogRepository struct { db *sql.DB }
func NewCatalogRepository(db *sql.DB) *CatalogRepository { return &CatalogRepository{db: db} }
func (r *CatalogRepository) Create(item models.ServiceCatalogItem) error { _,err:=r.db.Exec(`INSERT INTO service_catalog(vendor_id,name,protocol,duration_days,traffic_gb,price) VALUES($1,$2,$3,$4,$5,$6)`,item.VendorID,item.Name,item.Protocol,item.DurationDays,item.TrafficGB,item.Price); return err }
func (r *CatalogRepository) ByID(vendorID,id int64) (models.ServiceCatalogItem,error){ var x models.ServiceCatalogItem; err:=r.db.QueryRow(`SELECT id,vendor_id,name,protocol,duration_days,traffic_gb,price,created_at FROM service_catalog WHERE id=$1 AND vendor_id=$2 AND deleted_at IS NULL`,id,vendorID).Scan(&x.ID,&x.VendorID,&x.Name,&x.Protocol,&x.DurationDays,&x.TrafficGB,&x.Price,&x.CreatedAt); return x,err }

type OrderRepository struct { db *sql.DB }
func NewOrderRepository(db *sql.DB) *OrderRepository { return &OrderRepository{db: db} }
func (r *OrderRepository) Create(o models.Order) (int64,error){ var id int64; err:=r.db.QueryRow(`INSERT INTO orders(vendor_id,user_id,catalog_id,amount,status) VALUES($1,$2,$3,$4,$5) RETURNING id`,o.VendorID,o.UserID,o.CatalogID,o.Amount,o.Status).Scan(&id); return id,err }
func (r *OrderRepository) UpdateStatus(orderID int64, status string) error { _,err:=r.db.Exec(`UPDATE orders SET status=$1, updated_at=now() WHERE id=$2`,status,orderID); return err }
