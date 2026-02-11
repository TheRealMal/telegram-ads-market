package service

import (
	"ads-mrkt/internal/market/domain/entity"
	"context"
)

type UserRepository interface {
	UpsertUser(ctx context.Context, u *entity.User) error
	GetUserByID(ctx context.Context, id int64) (*entity.User, error)
	SetUserWallet(ctx context.Context, userID int64, walletAddressRaw string) error
	ClearUserWallet(ctx context.Context, userID int64) error
}

type UserService struct {
	botToken string
	userRepo UserRepository
}

func NewUserService(botToken string, userRepo UserRepository) *UserService {
	return &UserService{botToken: botToken, userRepo: userRepo}
}
