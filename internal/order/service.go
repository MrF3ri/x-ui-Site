package order

import (
	"errors"
	"garudapanel/internal/models"
	"garudapanel/internal/notification"
	"garudapanel/internal/payment"
)

type Repo interface { Create(o models.Order) (int64,error); UpdateStatus(orderID int64, status string) error }
type Catalog interface { ByID(vendorID,id int64) (models.ServiceCatalogItem,error) }
type Service struct { repo Repo; pay *payment.Service; catalog Catalog; notifier *notification.Hub }
func New(repo Repo, pay *payment.Service, catalog Catalog, notifier *notification.Hub) *Service { return &Service{repo:repo,pay:pay,catalog:catalog,notifier:notifier} }
func (s *Service) CreateFromWallet(vendorID,userID,catalogID int64) (int64,error){ item,err:=s.catalog.ByID(vendorID,catalogID); if err!=nil{return 0,err}; if item.VendorID!=vendorID { return 0, errors.New("vendor isolation violation") }; if err:=s.pay.Deduct(userID,item.Price); err!=nil{return 0,err}; id,err:=s.repo.Create(models.Order{VendorID:vendorID,UserID:userID,CatalogID:catalogID,Amount:item.Price,Status:"approved"}); if err==nil { s.notifier.Notify("order_status_changed", map[string]any{"order_id":id,"status":"approved"})}; return id,err }
func (s *Service) Renew(orderID int64) error { if err:=s.repo.UpdateStatus(orderID,"renewed"); err!=nil{return err}; s.notifier.Notify("order_status_changed", map[string]any{"order_id":orderID,"status":"renewed"}); return nil }
