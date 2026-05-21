package provisioning

import (
	"context"
	"garudapanel/internal/eventbus"
	"garudapanel/internal/models"
	"garudapanel/internal/notification"
	"garudapanel/internal/xui"
)

type PanelRepo interface { FirstByVendor(vendorID int64) (xui.Panel,error) }
type Worker struct { bus *eventbus.Bus; engine *Engine; panelRepo PanelRepo; notifier *notification.Hub }
func NewWorker(bus *eventbus.Bus, engine *Engine, panelRepo PanelRepo, notifier *notification.Hub) *Worker { return &Worker{bus:bus,engine:engine,panelRepo:panelRepo,notifier:notifier} }
func (w *Worker) Start(ctx context.Context){ ch := w.bus.Subscribe("order.approved"); go func(){ for { select { case <-ctx.Done(): return; case e:= <-ch: o := e.Payload.(models.Order); _ = w.handle(o) } } }() }
func (w *Worker) handle(o models.Order) error { panel,err:=w.panelRepo.FirstByVendor(o.VendorID); if err!=nil{return err}; _,err = w.engine.adapter.Provision(panel, xui.ProvisionRequest{VendorID:o.VendorID,UserID:o.UserID,Email:"user",UUID:"generated",DurationDays:30,TrafficGB:100,Protocol:"any"}); if err!=nil{return err}; w.notifier.Notify("service.provisioned", map[string]any{"order_id":o.ID}); return nil }
