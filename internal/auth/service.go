package auth

import (
	"errors"
	"garudapanel/internal/security"
	"garudapanel/internal/store"
)

type Service struct { st *store.Memory; secret string }
func NewService(st *store.Memory, secret string) *Service { return &Service{st: st, secret: secret} }
func (s *Service) Register(email,password string) error { h,err:=security.HashPassword(password); if err!=nil{return err}; _,err=s.st.CreateUser(email,h,"user"); return err }
func (s *Service) Login(email,password string) (string,error) { u,err:=s.st.GetUserByEmail(email); if err!=nil{return "",err}; if !security.ComparePassword(u.PasswordHash,password) { return "", errors.New("invalid credentials") }; return security.SignJWT(s.secret,u.ID,u.Role) }
