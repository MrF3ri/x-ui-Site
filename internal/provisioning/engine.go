package provisioning

import (
	"garudapanel/internal/models"
	"garudapanel/internal/xui"
)

type ServiceRepo interface {
	Create(s models.ProxyService) (int64, error)
	Extend(serviceID int64, addDays int, addGB int) error
}
type Engine struct {
	adapter     xui.Adapter
	serviceRepo ServiceRepo
}

func New(a xui.Adapter, sr ServiceRepo) *Engine { return &Engine{adapter: a, serviceRepo: sr} }
