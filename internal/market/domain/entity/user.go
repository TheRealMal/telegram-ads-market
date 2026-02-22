package entity

import "ads-mrkt/pkg/auth/role"

type User struct {
	ID            int64     `json:"id"`
	Username      string    `json:"username"`
	Photo         string    `json:"photo"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Locale        string    `json:"locale"`
	ReferrerID    int64     `json:"-"`
	AllowsPM      bool      `json:"-"`
	WalletAddress *string   `json:"wallet_address,omitempty"` // TON address in raw format
	Role          role.Role `json:"role"`                     // user | admin
}
