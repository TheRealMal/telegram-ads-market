package service

import (
	"context"

	"ads-mrkt/internal/market/domain/entity"
)

type DealRepository interface {
	CreateDeal(ctx context.Context, d *entity.Deal) error
	GetDealByID(ctx context.Context, id int64) (*entity.Deal, error)
	GetDealsByListingID(ctx context.Context, listingID int64) ([]*entity.Deal, error)
	UpdateDealDraftFieldsAndClearSignatures(ctx context.Context, d *entity.Deal) error
	SetDealLessorSignature(ctx context.Context, dealID int64, sig string) error
	SetDealLesseeSignature(ctx context.Context, dealID int64, sig string) error
	SetDealStatusApproved(ctx context.Context, dealID int64) error
	SignDealInTx(ctx context.Context, dealID int64, userID int64, sig string) error
}

type DealService struct {
	dealRepo DealRepository
}

func NewDealService(dealRepo DealRepository) *DealService {
	return &DealService{dealRepo: dealRepo}
}
