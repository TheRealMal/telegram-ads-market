package http

import (
	"context"

	"ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/pkg/auth"
)

type UserService interface {
	AuthUser(ctx context.Context, initDataStr string) (*entity.User, error)
}

type ListingService interface {
	CreateListing(ctx context.Context, userID int64, l *entity.Listing) error
	GetListing(ctx context.Context, id int64) (*entity.Listing, error)
	UpdateListing(ctx context.Context, userID int64, l *entity.Listing) error
	ListListingsByUserID(ctx context.Context, userID int64, typ *entity.ListingType) ([]*entity.Listing, error)
	ListListingsAll(ctx context.Context, typ *entity.ListingType) ([]*entity.Listing, error)
}

type DealService interface {
	CreateDeal(ctx context.Context, d *entity.Deal) error
	GetDeal(ctx context.Context, id int64) (*entity.Deal, error)
	GetDealsByListingID(ctx context.Context, listingID int64) ([]*entity.Deal, error)
	UpdateDealDraft(ctx context.Context, userID int64, d *entity.Deal) error
	SignDeal(ctx context.Context, userID int64, dealID int64) error
}

type ChannelService interface {
	ListMyChannels(ctx context.Context, userID int64) ([]*entity.Channel, error)
	RefreshChannel(ctx context.Context, channelID int64, userID int64) (*entity.Channel, error)
}

type Handler struct {
	userService    UserService
	listingService ListingService
	dealService    DealService
	channelService ChannelService
	jwtManager     *auth.JWTManager
}

func NewHandler(userService UserService, listingService ListingService, dealService DealService, channelService ChannelService, jwtManager *auth.JWTManager) *Handler {
	return &Handler{
		userService:    userService,
		listingService: listingService,
		dealService:    dealService,
		channelService: channelService,
		jwtManager:     jwtManager,
	}
}
