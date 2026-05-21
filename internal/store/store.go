package store

import (
	"errors"
	"garudapanel/internal/models"
	"sync"
	"time"
)

type Memory struct { mu sync.Mutex; users []models.User; vendors []models.Vendor; wallets []models.Wallet }
func NewMemory() *Memory { return &Memory{} }
func (m *Memory) CreateUser(email, hash, role string) (int64, error) { m.mu.Lock(); defer m.mu.Unlock(); for _,u:= range m.users { if u.Email==email { return 0, errors.New("email exists") } }; id:=int64(len(m.users)+1); m.users=append(m.users, models.User{ID:id,Email:email,PasswordHash:hash,Role:role,CreatedAt:time.Now()}); m.wallets=append(m.wallets, models.Wallet{ID:int64(len(m.wallets)+1),UserID:id,Balance:0,UpdatedAt:time.Now()}); return id,nil }
func (m *Memory) GetUserByEmail(email string) (models.User, error) { m.mu.Lock(); defer m.mu.Unlock(); for _,u:= range m.users { if u.Email==email { return u,nil } }; return models.User{}, errors.New("not found") }
func (m *Memory) CreateVendor(owner int64,name,slug string) error { m.mu.Lock(); defer m.mu.Unlock(); for _,v:= range m.vendors { if v.Slug==slug { return errors.New("slug exists") } }; m.vendors=append(m.vendors, models.Vendor{ID:int64(len(m.vendors)+1),OwnerUserID:owner,Name:name,Slug:slug,CreatedAt:time.Now()}); return nil }
func (m *Memory) ListVendor(owner int64) []models.Vendor { m.mu.Lock(); defer m.mu.Unlock(); out:=[]models.Vendor{}; for _,v:= range m.vendors { if v.OwnerUserID==owner { out=append(out,v) } }; return out }
func (m *Memory) WalletBalance(user int64) (int64,error) { m.mu.Lock(); defer m.mu.Unlock(); for _,w:= range m.wallets { if w.UserID==user { return w.Balance,nil } }; return 0, errors.New("not found") }
