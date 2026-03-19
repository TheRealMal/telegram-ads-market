package service

import (
	"context"
	"encoding/json"
	"time"

	evententity "ads-mrkt/internal/event/domain/entity"
	"ads-mrkt/internal/market/domain/entity"
	dealsigner "ads-mrkt/internal/market/service/deal_signer"
)

type dealRepository interface {
	CreateDeal(ctx context.Context, d *entity.Deal) error
	GetDealByID(ctx context.Context, id int64) (*entity.Deal, error)
	GetDealsByListingID(ctx context.Context, listingID int64) ([]*entity.Deal, error)
	GetDealsByListingIDForUser(ctx context.Context, listingID int64, userID int64) ([]*entity.Deal, error)
	ListDealsByUserID(ctx context.Context, userID int64) ([]*entity.Deal, error)
	UpdateDealDraftFieldsAndClearSignatures(ctx context.Context, d *entity.Deal) (*entity.Deal, error)
	SetDealStatusApproved(ctx context.Context, dealID int64) error
	SignDealInTx(ctx context.Context, dealID int64, userID int64, sig string) (*entity.Deal, error)
	SetDealPayoutAddress(ctx context.Context, dealID int64, userID int64, payoutAddressRaw string) (*entity.Deal, error)
	SetDealStatusRejected(ctx context.Context, dealID int64) (*entity.Deal, error)
	ListDealsWaitingEscrowDepositOlderThan(ctx context.Context, before time.Time) ([]*entity.Deal, error)
	SetDealStatusExpiredByDealID(ctx context.Context, dealID int64) error
	ListDealsEscrowConfirmedToComplete(ctx context.Context) ([]*entity.Deal, error)
	SetDealStatusCompleted(ctx context.Context, dealID int64) error
}

type userRepository interface {
	GetUserByID(ctx context.Context, id int64) (*entity.User, error)
}

type listingRepository interface {
	GetListingByID(ctx context.Context, id int64) (*entity.Listing, error)
}

type escrowService interface {
	ComputeEscrowAmount(priceNanoton int64) int64
}

type telegramNotificationAdder interface {
	AddTelegramNotificationEvent(ctx context.Context, event *evententity.EventTelegramNotification) error
}

type dealChatService interface {
	UpdateDealForumTopicEmoji(ctx context.Context, dealID int64, status entity.DealStatus)
}

type dealForumTopicRepository interface {
	GetDealForumTopicByDealID(ctx context.Context, dealID int64) (*entity.DealForumTopic, error)
}

// CreateDealInput holds the request inputs for creating a deal.
type CreateDealInput struct {
	ListingID int64
	ChannelID *int64
	Type      string
	Duration  int64
	PriceTON  float64
	Message   string
	Details   json.RawMessage
}

// UpdateDealDraftInput holds the optional fields for updating a deal draft.
type UpdateDealDraftInput struct {
	Type     *string
	Duration *int64
	PriceTON *float64
	Details  json.RawMessage
}

type dealService struct {
	dealRepo          dealRepository
	userRepo          userRepository
	listingRepo       listingRepository
	escrowSvc         escrowService
	notificationAdder telegramNotificationAdder
	dealChatSvc       dealChatService
	forumTopicRepo    dealForumTopicRepository
	signerSvc         *dealsigner.Service
}

func NewDealService(dealRepo dealRepository, userRepo userRepository, listingRepo listingRepository, escrowSvc escrowService, notificationAdder telegramNotificationAdder, dealChatSvc dealChatService, forumTopicRepo dealForumTopicRepository) *dealService {
	return &dealService{
		dealRepo:          dealRepo,
		userRepo:          userRepo,
		listingRepo:       listingRepo,
		escrowSvc:         escrowSvc,
		notificationAdder: notificationAdder,
		dealChatSvc:       dealChatSvc,
		forumTopicRepo:    forumTopicRepo,
		signerSvc:         dealsigner.NewService(dealRepo, userRepo, notificationAdder, forumTopicRepo),
	}
}
