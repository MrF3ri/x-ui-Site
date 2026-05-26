package storefront

import (
	"database/sql"
	"errors"
	"regexp"
)

type Vendor struct {
	ID                 int64
	Name, Slug, Status string
}
type Product struct {
	ID, VendorID                            int64
	Slug, Title, Description, Protocol      string
	InboundID, XUINodeID                    int64
	TrafficLimitGB, DurationDays            int
	PriceToman                              int64
	IsActive, AutoProvision, RenewalEnabled bool
	CountryCode, StockStatus                string
}

type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (s *Service) validateSlug(slug string) error {
	if !slugRe.MatchString(slug) {
		return errors.New("invalid slug")
	}
	return nil
}
func (s *Service) GetVendorBySlug(slug string) (Vendor, error) {
	if err := s.validateSlug(slug); err != nil {
		return Vendor{}, err
	}
	var v Vendor
	err := s.db.QueryRow(`SELECT id,name,slug,status FROM vendors WHERE slug=$1 AND deleted_at IS NULL`, slug).Scan(&v.ID, &v.Name, &v.Slug, &v.Status)
	return v, err
}
func (s *Service) ListProducts(vendorID int64) ([]Product, error) {
	rows, err := s.db.Query(`SELECT id,vendor_id,slug,title,description,protocol,inbound_id,xui_node_id,traffic_limit_gb,duration_days,price_toman,is_active,auto_provision,renewal_enabled,country_code,stock_status FROM catalog_items WHERE vendor_id=$1 AND deleted_at IS NULL ORDER BY id DESC`, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Product{}
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.VendorID, &p.Slug, &p.Title, &p.Description, &p.Protocol, &p.InboundID, &p.XUINodeID, &p.TrafficLimitGB, &p.DurationDays, &p.PriceToman, &p.IsActive, &p.AutoProvision, &p.RenewalEnabled, &p.CountryCode, &p.StockStatus); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s *Service) GetProduct(vendorID int64, slug string) (Product, error) {
	if err := s.validateSlug(slug); err != nil {
		return Product{}, err
	}
	var p Product
	err := s.db.QueryRow(`SELECT id,vendor_id,slug,title,description,protocol,inbound_id,xui_node_id,traffic_limit_gb,duration_days,price_toman,is_active,auto_provision,renewal_enabled,country_code,stock_status FROM catalog_items WHERE vendor_id=$1 AND slug=$2 AND deleted_at IS NULL`, vendorID, slug).Scan(&p.ID, &p.VendorID, &p.Slug, &p.Title, &p.Description, &p.Protocol, &p.InboundID, &p.XUINodeID, &p.TrafficLimitGB, &p.DurationDays, &p.PriceToman, &p.IsActive, &p.AutoProvision, &p.RenewalEnabled, &p.CountryCode, &p.StockStatus)
	return p, err
}
func (s *Service) SeedDemo() error {
	if s.db == nil {
		return nil
	}
	var c int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM vendors WHERE slug='demo'`).Scan(&c)
	if c > 0 {
		return nil
	}
	var uid int64
	if err := s.db.QueryRow(`INSERT INTO users(email,password_hash,role) VALUES('demo@garuda.local','x','vendor') RETURNING id`).Scan(&uid); err != nil {
		return nil
	}
	var vid int64
	if err := s.db.QueryRow(`INSERT INTO vendors(owner_user_id,name,slug,status) VALUES($1,'Demo Vendor','demo','active') RETURNING id`, uid).Scan(&vid); err != nil {
		return nil
	}
	_, _ = s.db.Exec(`INSERT INTO catalog_items(vendor_id,slug,title,description,protocol,inbound_id,xui_node_id,traffic_limit_gb,duration_days,price_toman,is_active,auto_provision,renewal_enabled,country_code,stock_status) VALUES ($1,'premium-vless','Premium VLESS','Fast stable route','vless',1,1,200,30,199000,true,true,true,'DE','in_stock'),($1,'starter-trojan','Starter Trojan','Budget route','trojan',2,1,100,30,99000,true,true,true,'NL','low_stock')`, vid)
	return nil
}
