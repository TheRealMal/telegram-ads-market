package service

import (
	"context"
	"fmt"

	"ads-mrkt/internal/market/domain/entity"
)

// AuthUser verifies Telegram Mini App init data, upserts the user into market.user, and returns the user.
func (s *userService) AuthUser(ctx context.Context, initDataStr string, referrerID int64) (*entity.User, error) {
	initData, err := parseAndVerifyInitData(s.botToken, initDataStr)
	if err != nil {
		return nil, fmt.Errorf("verify init data: %w", err)
	}
	if initData.User == nil {
		return nil, fmt.Errorf("init data has no user")
	}

	u := &entity.User{
		ID:         initData.User.ID,
		Username:   initData.User.Username,
		Photo:      initData.User.PhotoURL,
		FirstName:  initData.User.FirstName,
		LastName:   initData.User.LastName,
		Locale:     initData.User.LanguageCode,
		ReferrerID: referrerID,
		AllowsPM:   initData.User.AllowsWriteToPM,
	}
	if err := s.userRepo.UpsertUser(ctx, u); err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return u, nil
}

// SetWallet stores the user's TON wallet address (raw format) for deal payouts.
func (s *userService) SetWallet(ctx context.Context, userID int64, walletAddressRaw string) error {
	if walletAddressRaw == "" {
		return fmt.Errorf("wallet address is required")
	}
	return s.userRepo.SetUserWallet(ctx, userID, walletAddressRaw)
}

// ClearWallet removes the user's linked wallet (disconnect).
func (s *userService) ClearWallet(ctx context.Context, userID int64) error {
	return s.userRepo.ClearUserWallet(ctx, userID)
}
