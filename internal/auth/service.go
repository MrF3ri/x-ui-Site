package auth

import (
	"garudapanel/internal/repository"
	"garudapanel/internal/security"
)

type Service struct { users *repository.UserRepository; wallets *repository.WalletRepository; secret string }
func NewService(users *repository.UserRepository, wallets *repository.WalletRepository, secret string) *Service { return &Service{users: users, wallets: wallets, secret: secret} }
func (s *Service) Register(email,password string) error { h,err:=security.HashPassword(password); if err!=nil{return err}; id,err:=s.users.Create(email,h,"user"); if err!=nil{return err}; return s.wallets.EnsureForUser(id) }
func (s *Service) Login(email,password string) (string,error) { u,err:=s.users.ByEmail(email); if err!=nil{return "",err}; if !security.ComparePassword(u.PasswordHash,password){ return "",err }; return security.SignJWT(s.secret,u.ID,u.Role) }
