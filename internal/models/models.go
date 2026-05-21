package models

import "time"

type User struct { ID int64 `json:"id"`; VendorID *int64 `json:"vendor_id,omitempty"`; Email string `json:"email"`; PasswordHash string `json:"-"`; Role string `json:"role"`; CreatedAt time.Time `json:"created_at"` }
type Vendor struct { ID int64 `json:"id"`; OwnerUserID int64 `json:"owner_user_id"`; Name string `json:"name"`; Slug string `json:"slug"`; CreatedAt time.Time `json:"created_at"` }
type Wallet struct { ID int64 `json:"id"`; UserID int64 `json:"user_id"`; VendorID *int64 `json:"vendor_id,omitempty"`; Balance int64 `json:"balance"`; UpdatedAt time.Time `json:"updated_at"` }
