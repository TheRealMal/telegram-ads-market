package service

import (
	"ads-mrkt/internal/market/domain/entity"
	"context"
)

type userRepository interface {
	UpsertUser(ctx context.Context, u *entity.User) error
	GetUserByID(ctx context.Context, id int64) (*entity.User, error)
	GetReferrals(ctx context.Context, userID int64) ([]*entity.User, error)
	SetUserWallet(ctx context.Context, userID int64, walletAddressRaw string) error
	ClearUserWallet(ctx context.Context, userID int64) error
}

type userService struct {
	environment string
	botToken    string
	userRepo    userRepository
}

func NewUserService(environment string, botToken string, userRepo userRepository) *userService {
	return &userService{
		environment: environment,
		botToken:    botToken,
		userRepo:    userRepo,
	}
}
