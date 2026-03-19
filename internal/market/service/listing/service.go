package service

import (
	"context"
	"encoding/json"

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

type userRepository interface {
	GetUserByID(ctx context.Context, id int64) (*entity.User, error)
}

// CreateListingInput holds all request inputs for creating a listing.
type CreateListingInput struct {
	Status       string
	ChannelID    *int64
	Type         string
	PricesTON    json.RawMessage
	Categories   []string
	Description  string
	PreparedPost json.RawMessage
}

// UpdateListingInput holds the optional request inputs for updating a listing.
type UpdateListingInput struct {
	Status       *string
	Type         *string
	PricesTON    json.RawMessage
	Categories   *[]string
	Description  *string
	PreparedPost json.RawMessage
}

type listingService struct {
	listingRepo listingRepository
	adminRepo   channelAdminRepository
	userRepo    userRepository
}

func NewListingService(listingRepo listingRepository, adminRepo channelAdminRepository, userRepo userRepository) *listingService {
	return &listingService{listingRepo: listingRepo, adminRepo: adminRepo, userRepo: userRepo}
}
