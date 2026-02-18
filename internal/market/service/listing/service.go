package service

import (
	"context"

	"ads-mrkt/internal/market/domain/entity"
)

type listingRepository interface {
	CreateListing(ctx context.Context, l *entity.Listing) error
	GetListingByID(ctx context.Context, id int64) (*entity.Listing, error)
	UpdateListing(ctx context.Context, l *entity.Listing) error
	DeleteListing(ctx context.Context, id int64) error
	ListListingsByUserID(ctx context.Context, userID int64, typ *entity.ListingType) ([]*entity.Listing, error)
	ListListingsAll(ctx context.Context, typ *entity.ListingType, categories []string, minFollowers *int64) ([]*entity.Listing, error)
}

type channelAdminRepository interface {
	IsChannelAdmin(ctx context.Context, userID, channelID int64) (bool, error)
}

type listingService struct {
	listingRepo listingRepository
	adminRepo   channelAdminRepository
}

func NewListingService(listingRepo listingRepository, adminRepo channelAdminRepository) *listingService {
	return &listingService{listingRepo: listingRepo, adminRepo: adminRepo}
}
