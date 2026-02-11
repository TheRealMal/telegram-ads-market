package http

import (
	"context"

	"ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/pkg/auth"
)

type UserService interface {
	AuthUser(ctx context.Context, initDataStr string, referrerID int64) (*entity.User, error)
	SetWallet(ctx context.Context, userID int64, walletAddressRaw string) error
}

type ListingService interface {
	CreateListing(ctx context.Context, userID int64, l *entity.Listing) error
	GetListing(ctx context.Context, id int64) (*entity.Listing, error)
	UpdateListing(ctx context.Context, userID int64, l *entity.Listing) error
	DeleteListing(ctx context.Context, userID int64, id int64) error
	ListListingsByUserID(ctx context.Context, userID int64, typ *entity.ListingType) ([]*entity.Listing, error)
	ListListingsAll(ctx context.Context, typ *entity.ListingType, categories []string, minFollowers *int64) ([]*entity.Listing, error)
}

type DealService interface {
	CreateDeal(ctx context.Context, d *entity.Deal) error
	GetDeal(ctx context.Context, id int64) (*entity.Deal, error)
	GetDealsByListingID(ctx context.Context, listingID int64) ([]*entity.Deal, error)
	GetDealsByUserID(ctx context.Context, userID int64) ([]*entity.Deal, error)
	UpdateDealDraft(ctx context.Context, userID int64, d *entity.Deal) error
	SignDeal(ctx context.Context, userID int64, dealID int64) error
	SetDealPayoutAddress(ctx context.Context, userID int64, dealID int64, payoutAddressRaw string) error
	RejectDeal(ctx context.Context, userID int64, dealID int64) error
}

type DealChatService interface {
	SendDealChatMessage(ctx context.Context, dealID, userID int64) (*entity.DealChat, error)
	ListDealMessages(ctx context.Context, dealID, userID int64) ([]*entity.DealChat, error)
}

type ChannelService interface {
	ListMyChannels(ctx context.Context, userID int64) ([]*entity.Channel, error)
	RefreshChannel(ctx context.Context, channelID int64, userID int64) (*entity.Channel, error)
	GetChannelStats(ctx context.Context, channelID int64, userID int64) (interface{}, error)
}

type Handler struct {
	userService     UserService
	listingService  ListingService
	dealService     DealService
	dealChatService DealChatService
	channelService  ChannelService
	jwtManager      *auth.JWTManager
}

func NewHandler(userService UserService, listingService ListingService, dealService DealService, dealChatService DealChatService, channelService ChannelService, jwtManager *auth.JWTManager) *Handler {
	return &Handler{
		userService:     userService,
		listingService:  listingService,
		dealService:     dealService,
		dealChatService: dealChatService,
		channelService:  channelService,
		jwtManager:      jwtManager,
	}
}
