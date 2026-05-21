package repository

import (
	"database/sql"
	"garudapanel/internal/models"
	"garudapanel/internal/xui"
)

type PanelRepository struct { db *sql.DB }
func NewPanelRepository(db *sql.DB) *PanelRepository { return &PanelRepository{db: db} }
func (r *PanelRepository) FirstByVendor(vendorID int64) (xui.Panel,error){ var p xui.Panel; err:=r.db.QueryRow(`SELECT vendor_id,name,url,token,inbound_id FROM vendor_panels WHERE vendor_id=$1 AND deleted_at IS NULL ORDER BY id LIMIT 1`,vendorID).Scan(&p.VendorID,&p.Name,&p.URL,&p.Token,&p.InboundID); return p,err }

type ServiceRepository struct { db *sql.DB }
func NewServiceRepository(db *sql.DB) *ServiceRepository { return &ServiceRepository{db:db} }
func (r *ServiceRepository) Create(s models.ProxyService) (int64,error){ var id int64; err:=r.db.QueryRow(`INSERT INTO services(vendor_id,user_id,uuid,protocol,expires_at,traffic_gb,status) VALUES($1,$2,$3,$4,$5,$6,'active') RETURNING id`,s.VendorID,s.UserID,s.UUID,s.Protocol,s.ExpiresAt,s.TrafficGB).Scan(&id); return id,err }
func (r *ServiceRepository) Extend(serviceID int64, addDays int, addGB int) error { _,err:=r.db.Exec(`UPDATE services SET expires_at=expires_at + ($1 || ' day')::interval, traffic_gb=traffic_gb+$2, updated_at=now() WHERE id=$3`,addDays,addGB,serviceID); return err }
