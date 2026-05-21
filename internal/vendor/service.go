package vendor

import (
"garudapanel/internal/models"
"garudapanel/internal/store"
)

type Service struct { st *store.Memory }
func NewService(st *store.Memory) *Service { return &Service{st: st} }
func (s *Service) Create(ownerID int64, name, slug string) error { return s.st.CreateVendor(ownerID,name,slug) }
func (s *Service) ListByOwner(ownerID int64) ([]models.Vendor, error) { return s.st.ListVendor(ownerID), nil }
