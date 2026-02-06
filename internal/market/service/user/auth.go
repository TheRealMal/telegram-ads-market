package service

import (
	"context"
	"fmt"

	"ads-mrkt/internal/market/domain/entity"
)

// AuthUser verifies Telegram Mini App init data, upserts the user into market.user, and returns the user.
func (s *UserService) AuthUser(ctx context.Context, initDataStr string) (*entity.User, error) {
	initData, err := parseAndVerifyInitData(s.botToken, initDataStr)
	if err != nil {
		return nil, fmt.Errorf("verify init data: %w", err)
	}
	if initData.User == nil {
		return nil, fmt.Errorf("init data has no user")
	}

	u := &entity.User{
		ID:        initData.User.ID,
		Username:  initData.User.Username,
		Photo:     initData.User.PhotoURL,
		FirstName: initData.User.FirstName,
		LastName:  initData.User.LastName,
		Locale:    initData.User.LanguageCode,
	}
	if err := s.userRepo.UpsertUser(ctx, u); err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return u, nil
}
