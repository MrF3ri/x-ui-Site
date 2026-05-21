package repository

import "database/sql"

type CatalogItemInput struct {
	VendorID int64
	Slug, Title, Description, Protocol string
	InboundID, XUINodeID int64
	TrafficLimitGB, DurationDays int
	PriceToman int64
	IsActive, AutoProvision, RenewalEnabled bool
	CountryCode, StockStatus string
}

type CatalogRepository struct { db *sql.DB }
func NewCatalogRepository(db *sql.DB) *CatalogRepository { return &CatalogRepository{db: db} }
func (r *CatalogRepository) Create(item CatalogItemInput) error { _,err:=r.db.Exec(`INSERT INTO catalog_items(vendor_id,slug,title,description,protocol,inbound_id,xui_node_id,traffic_limit_gb,duration_days,price_toman,is_active,auto_provision,renewal_enabled,country_code,stock_status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,item.VendorID,item.Slug,item.Title,item.Description,item.Protocol,item.InboundID,item.XUINodeID,item.TrafficLimitGB,item.DurationDays,item.PriceToman,item.IsActive,item.AutoProvision,item.RenewalEnabled,item.CountryCode,item.StockStatus); return err }
