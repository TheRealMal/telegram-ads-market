package service

import (
	"context"

	"ads-mrkt/internal/market/domain/entity"
)

type ListingRepository interface {
	CreateListing(ctx context.Context, l *entity.Listing) error
	GetListingByID(ctx context.Context, id int64) (*entity.Listing, error)
	UpdateListing(ctx context.Context, l *entity.Listing) error
	DeleteListing(ctx context.Context, id int64) error
	ListListingsByUserID(ctx context.Context, userID int64, typ *entity.ListingType) ([]*entity.Listing, error)
	ListListingsAll(ctx context.Context, typ *entity.ListingType, categories []string, minFollowers *int64) ([]*entity.Listing, error)
}

type ChannelAdminRepository interface {
	IsChannelAdmin(ctx context.Context, userID, channelID int64) (bool, error)
}

type ListingService struct {
	listingRepo ListingRepository
	adminRepo   ChannelAdminRepository
}

func NewListingService(listingRepo ListingRepository, adminRepo ChannelAdminRepository) *ListingService {
	return &ListingService{listingRepo: listingRepo, adminRepo: adminRepo}
}
