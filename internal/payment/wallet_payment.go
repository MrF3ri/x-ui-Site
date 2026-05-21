package payment

type WalletRepo interface { ApplyTransaction(userID int64, amount int64, typ string) error; Balance(userID int64) (int64,error) }
type Service struct { wallet WalletRepo }
func New(wallet WalletRepo) *Service { return &Service{wallet: wallet} }
func (s *Service) Deduct(userID int64, amount int64) error { return s.wallet.ApplyTransaction(userID, -amount, "order_debit") }
