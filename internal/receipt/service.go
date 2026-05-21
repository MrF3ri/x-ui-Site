package receipt

import "errors"

type ObjectStore interface { PutWebP(key string, data []byte) error }
type Service struct { store ObjectStore; maxSize int }
func New(store ObjectStore) *Service { return &Service{store: store, maxSize: 5*1024*1024} }
func (s *Service) Upload(orderID,vendorID,userID int64, key string, data []byte) error { if len(data)==0 || len(data)>s.maxSize { return errors.New("invalid file size") }; return s.store.PutWebP(key,data) }
