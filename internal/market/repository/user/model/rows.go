package model

import (
	"ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/pkg/auth/role"
)

type UserRow struct {
	ID            int64   `db:"id"`
	Username      string  `db:"username"`
	Photo         string  `db:"photo"`
	FirstName     string  `db:"first_name"`
	LastName      string  `db:"last_name"`
	Locale        string  `db:"locale"`
	ReferrerID    int64   `db:"referrer_id"`
	AllowsPM      bool    `db:"allows_pm"`
	WalletAddress *string `db:"wallet_address"`
	Role          string  `db:"role"`
}

type RoleRow struct {
	Role string `db:"role"`
}

func UserRowToEntity(row UserRow) *entity.User {
	return &entity.User{
		ID:            row.ID,
		Username:      row.Username,
		Photo:         row.Photo,
		FirstName:     row.FirstName,
		LastName:      row.LastName,
		Locale:        row.Locale,
		ReferrerID:    row.ReferrerID,
		AllowsPM:      row.AllowsPM,
		WalletAddress: row.WalletAddress,
		Role:          role.Role(row.Role),
	}
}
