package service

import (
	"context"

	"ads-mrkt/internal/market/domain"
	"ads-mrkt/internal/market/domain/entity"
)

type DealRepository interface {
	CreateDeal(ctx context.Context, d *entity.Deal) error
	GetDealByID(ctx context.Context, id int64) (*entity.Deal, error)
	GetDealsByListingID(ctx context.Context, listingID int64) ([]*entity.Deal, error)
	ListDealsByUserID(ctx context.Context, userID int64) ([]*entity.Deal, error)
	UpdateDealDraftFieldsAndClearSignatures(ctx context.Context, d *entity.Deal) error
	SetDealStatusApproved(ctx context.Context, dealID int64) error
	SignDealInTx(ctx context.Context, dealID int64, userID int64, sig string) error
	SetDealPayoutAddress(ctx context.Context, dealID int64, userID int64, payoutAddressRaw string) error
	SetDealStatusRejected(ctx context.Context, dealID int64) (bool, error)
}

type UserRepository interface {
	GetUserByID(ctx context.Context, id int64) (*entity.User, error)
}

type DealService struct {
	dealRepo     DealRepository
	userRepo     UserRepository
	escrowConfig domain.EscrowConfig
}

func NewDealService(dealRepo DealRepository, userRepo UserRepository, escrowConfig domain.EscrowConfig) *DealService {
	return &DealService{dealRepo: dealRepo, userRepo: userRepo, escrowConfig: escrowConfig}
}
