package service

import (
	"ads-mrkt/internal/market/domain/entity"
	"context"
)

type userRepository interface {
	UpsertUser(ctx context.Context, u *entity.User) error
	GetUserByID(ctx context.Context, id int64) (*entity.User, error)
	SetUserWallet(ctx context.Context, userID int64, walletAddressRaw string) error
	ClearUserWallet(ctx context.Context, userID int64) error
}

type userService struct {
	botToken string
	userRepo userRepository
}

func NewUserService(botToken string, userRepo userRepository) *userService {
	return &userService{botToken: botToken, userRepo: userRepo}
}
