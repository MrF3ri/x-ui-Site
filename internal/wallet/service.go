package wallet

import "garudapanel/internal/store"

type Service struct { st *store.Memory }
func NewService(st *store.Memory) *Service { return &Service{st: st} }
func (s *Service) Balance(userID int64) (int64, error) { return s.st.WalletBalance(userID) }
