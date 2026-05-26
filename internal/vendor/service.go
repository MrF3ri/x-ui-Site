package vendor

import (
	"garudapanel/internal/models"
	"garudapanel/internal/repository"
)

type Service struct{ repo *repository.VendorRepository }

func NewService(repo *repository.VendorRepository) *Service { return &Service{repo: repo} }
func (s *Service) Create(ownerID int64, name, slug string) error {
	return s.repo.Create(ownerID, name, slug)
}
func (s *Service) ListByOwner(ownerID int64) ([]models.Vendor, error) {
	return s.repo.ListByOwner(ownerID)
}
