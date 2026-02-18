package http

import (
	"context"

	"ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/pkg/auth"
)

type userService interface {
	AuthUser(ctx context.Context, initDataStr string, referrerID int64) (*entity.User, error)
	SetWallet(ctx context.Context, userID int64, walletAddressRaw string) error
	ClearWallet(ctx context.Context, userID int64) error
}

type listingService interface {
	CreateListing(ctx context.Context, userID int64, l *entity.Listing) error
	GetListing(ctx context.Context, id int64) (*entity.Listing, error)
	UpdateListing(ctx context.Context, userID int64, l *entity.Listing) error
	DeleteListing(ctx context.Context, userID int64, id int64) error
	ListListingsByUserID(ctx context.Context, userID int64, typ *entity.ListingType) ([]*entity.Listing, error)
	ListListingsAll(ctx context.Context, typ *entity.ListingType, categories []string, minFollowers *int64) ([]*entity.Listing, error)
}

type dealService interface {
	CreateDeal(ctx context.Context, d *entity.Deal) error
	GetDeal(ctx context.Context, id int64) (*entity.Deal, error)
	GetDealForUser(ctx context.Context, id int64, userID int64) (*entity.Deal, error)
	GetDealsByListingID(ctx context.Context, listingID int64) ([]*entity.Deal, error)
	GetDealsByListingIDForUser(ctx context.Context, listingID int64, userID int64) ([]*entity.Deal, error)
	GetDealsByUserID(ctx context.Context, userID int64) ([]*entity.Deal, error)
	UpdateDealDraft(ctx context.Context, userID int64, d *entity.Deal) error
	SignDeal(ctx context.Context, userID int64, dealID int64) error
	SetDealPayoutAddress(ctx context.Context, userID int64, dealID int64, payoutAddressRaw string) error
	RejectDeal(ctx context.Context, userID int64, dealID int64) error
}

type dealChatService interface {
	SendDealChatMessage(ctx context.Context, dealID, userID int64) (*entity.DealChat, error)
	ListDealMessages(ctx context.Context, dealID, userID int64) ([]*entity.DealChat, error)
}

type channelService interface {
	ListMyChannels(ctx context.Context, userID int64) ([]*entity.Channel, error)
	RefreshChannel(ctx context.Context, channelID int64, userID int64) (*entity.Channel, error)
	GetChannelStats(ctx context.Context, channelID int64, userID int64) (interface{}, error)
}

type handler struct {
	userService     userService
	listingService  listingService
	dealService     dealService
	dealChatService dealChatService
	channelService  channelService
	jwtManager      *auth.JWTManager
}

func NewHandler(userService userService, listingService listingService, dealService dealService, dealChatService dealChatService, channelService channelService, jwtManager *auth.JWTManager) *handler {
	return &handler{
		userService:     userService,
		listingService:  listingService,
		dealService:     dealService,
		dealChatService: dealChatService,
		channelService:  channelService,
		jwtManager:      jwtManager,
	}
}
