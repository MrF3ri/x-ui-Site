package servicecatalog

import "garudapanel/internal/models"

type Repo interface {
	Create(item models.ServiceCatalogItem) error
	ByID(vendorID, id int64) (models.ServiceCatalogItem, error)
}
type Service struct{ repo Repo }

func New(repo Repo) *Service                                   { return &Service{repo: repo} }
func (s *Service) Create(item models.ServiceCatalogItem) error { return s.repo.Create(item) }
func (s *Service) ByID(vendorID, id int64) (models.ServiceCatalogItem, error) {
	return s.repo.ByID(vendorID, id)
}
