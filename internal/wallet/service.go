package wallet

import "garudapanel/internal/repository"

type Service struct { repo *repository.WalletRepository }
func NewService(repo *repository.WalletRepository) *Service { return &Service{repo: repo} }
func (s *Service) Balance(userID int64) (int64, error) { return s.repo.Balance(userID) }
