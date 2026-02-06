package service

import (
	"context"

	marketerrors "ads-mrkt/internal/market/domain/errors"
	"ads-mrkt/internal/market/domain/entity"
)

// CreateListing creates a listing. For type lessor, userID must be an admin of the channel (channelID required).
func (s *ListingService) CreateListing(ctx context.Context, userID int64, l *entity.Listing) error {
	l.UserID = userID
	if l.Type == entity.ListingTypeLessor {
		if l.ChannelID == nil {
			return marketerrors.ErrNotChannelAdmin
		}
		ok, err := s.adminRepo.IsChannelAdmin(ctx, userID, *l.ChannelID)
		if err != nil {
			return err
		}
		if !ok {
			return marketerrors.ErrNotChannelAdmin
		}
	}
	return s.listingRepo.CreateListing(ctx, l)
}

func (s *ListingService) GetListing(ctx context.Context, id int64) (*entity.Listing, error) {
	return s.listingRepo.GetListingByID(ctx, id)
}

func (s *ListingService) UpdateListing(ctx context.Context, userID int64, l *entity.Listing) error {
	existing, err := s.listingRepo.GetListingByID(ctx, l.ID)
	if err != nil || existing == nil {
		return marketerrors.ErrNotFound
	}
	if existing.UserID != userID {
		return marketerrors.ErrUnauthorizedSide
	}
	if l.Type == entity.ListingTypeLessor && l.ChannelID != nil {
		ok, err := s.adminRepo.IsChannelAdmin(ctx, userID, *l.ChannelID)
		if err != nil {
			return err
		}
		if !ok {
			return marketerrors.ErrNotChannelAdmin
		}
	}
	return s.listingRepo.UpdateListing(ctx, l)
}

func (s *ListingService) ListListingsByUserID(ctx context.Context, userID int64, typ *entity.ListingType) ([]*entity.Listing, error) {
	return s.listingRepo.ListListingsByUserID(ctx, userID, typ)
}

// ListListingsAll returns all listings, optionally filtered by type (for public discovery).
func (s *ListingService) ListListingsAll(ctx context.Context, typ *entity.ListingType) ([]*entity.Listing, error) {
	return s.listingRepo.ListListingsAll(ctx, typ)
}
