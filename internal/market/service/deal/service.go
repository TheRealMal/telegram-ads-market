package service

import (
	"context"
	"time"

	"ads-mrkt/internal/market/domain/entity"
)

type dealRepository interface {
	CreateDeal(ctx context.Context, d *entity.Deal) error
	GetDealByID(ctx context.Context, id int64) (*entity.Deal, error)
	GetDealsByListingID(ctx context.Context, listingID int64) ([]*entity.Deal, error)
	GetDealsByListingIDForUser(ctx context.Context, listingID int64, userID int64) ([]*entity.Deal, error)
	ListDealsByUserID(ctx context.Context, userID int64) ([]*entity.Deal, error)
	UpdateDealDraftFieldsAndClearSignatures(ctx context.Context, d *entity.Deal) error
	SetDealStatusApproved(ctx context.Context, dealID int64) error
	SignDealInTx(ctx context.Context, dealID int64, userID int64, sig string) error
	SetDealPayoutAddress(ctx context.Context, dealID int64, userID int64, payoutAddressRaw string) error
	SetDealStatusRejected(ctx context.Context, dealID int64) (bool, error)
	ListDealsWaitingEscrowDepositOlderThan(ctx context.Context, before time.Time) ([]*entity.Deal, error)
	SetDealStatusExpiredByDealID(ctx context.Context, dealID int64) error
	ListDealsEscrowConfirmedToComplete(ctx context.Context) ([]*entity.Deal, error)
	SetDealStatusCompleted(ctx context.Context, dealID int64) error
}

type userRepository interface {
	GetUserByID(ctx context.Context, id int64) (*entity.User, error)
}

type escrowService interface {
	ComputeEscrowAmount(priceNanoton int64) int64
}

type telegramNotificationAdder interface {
	AddTelegramNotificationEvent(ctx context.Context, chatID int64, message string) error
}

type dealService struct {
	dealRepo          dealRepository
	userRepo          userRepository
	escrowSvc         escrowService
	notificationAdder telegramNotificationAdder
}

func NewDealService(dealRepo dealRepository, userRepo userRepository, escrowSvc escrowService, notificationAdder telegramNotificationAdder) *dealService {
	return &dealService{
		dealRepo:          dealRepo,
		userRepo:          userRepo,
		escrowSvc:         escrowSvc,
		notificationAdder: notificationAdder,
	}
}
