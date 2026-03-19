package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
)

// CreateListing validates inputs and creates a listing.
func (s *listingService) CreateListing(ctx context.Context, userID int64, input CreateListingInput) (*entity.Listing, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.HasWallet() {
		return nil, marketerrors.ErrWalletRequired
	}

	if err := entity.ValidateListingPrices(input.PricesTON); err != nil {
		return nil, fmt.Errorf("%w: %s", marketerrors.ErrInvalidInput, err.Error())
	}
	if err := entity.ValidateListingCategories(input.Categories); err != nil {
		return nil, fmt.Errorf("%w: %s", marketerrors.ErrInvalidInput, err.Error())
	}

	pricesNanoton, err := entity.ConvertListingPricesTONToNanoton(input.PricesTON)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", marketerrors.ErrInvalidInput, err.Error())
	}

	listingType := entity.ListingType(input.Type)
	if err := validateInstantPostListing(listingType, input.PricesTON, input.PreparedPost); err != nil {
		return nil, err
	}

	categories, err := json.Marshal(input.Categories)
	if err != nil {
		return nil, err
	}

	l := &entity.Listing{
		UserID:       userID,
		Status:       entity.ListingStatus(input.Status),
		ChannelID:    input.ChannelID,
		Type:         listingType,
		Prices:       pricesNanoton,
		Categories:   categories,
		Description:  input.Description,
		PreparedPost: input.PreparedPost,
	}
	if l.Status == "" {
		l.Status = entity.ListingStatusInactive
	}

	if l.Type == entity.ListingTypeLessor {
		if l.ChannelID == nil {
			return nil, marketerrors.ErrNotChannelAdmin
		}
		ok, err := s.adminRepo.IsChannelAdmin(ctx, userID, *l.ChannelID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, marketerrors.ErrNotChannelAdmin
		}
	}

	if err := s.listingRepo.CreateListing(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (s *listingService) GetListing(ctx context.Context, id int64) (*entity.Listing, error) {
	return s.listingRepo.GetListingByID(ctx, id)
}

// UpdateListing validates and applies the patch to the existing listing.
func (s *listingService) UpdateListing(ctx context.Context, userID int64, id int64, input UpdateListingInput) error {
	existing, err := s.listingRepo.GetListingByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return marketerrors.ErrNotFound
	}
	if existing.UserID != userID {
		return marketerrors.ErrUnauthorizedSide
	}

	if len(input.PricesTON) > 0 {
		if err := entity.ValidateListingPrices(input.PricesTON); err != nil {
			return fmt.Errorf("%w: %s", marketerrors.ErrInvalidInput, err.Error())
		}
	}
	if input.Categories != nil {
		if err := entity.ValidateListingCategories(*input.Categories); err != nil {
			return fmt.Errorf("%w: %s", marketerrors.ErrInvalidInput, err.Error())
		}
	}

	l := *existing
	l.ID = id
	if input.Status != nil {
		l.Status = entity.ListingStatus(*input.Status)
	}
	if input.Type != nil {
		l.Type = entity.ListingType(*input.Type)
	}
	if len(input.PricesTON) > 0 {
		pricesNanoton, err := entity.ConvertListingPricesTONToNanoton(input.PricesTON)
		if err != nil {
			return fmt.Errorf("%w: %s", marketerrors.ErrInvalidInput, err.Error())
		}
		l.Prices = pricesNanoton
	}
	if input.Categories != nil {
		categories, err := json.Marshal(*input.Categories)
		if err != nil {
			return err
		}
		l.Categories = categories
	}
	if input.Description != nil {
		l.Description = *input.Description
	}
	if input.PreparedPost != nil {
		l.PreparedPost = input.PreparedPost
	}

	listingType := l.Type
	pricesToCheck := l.Prices
	if input.PricesTON != nil {
		pricesToCheck = input.PricesTON
	}
	if err := validateInstantPostListing(listingType, pricesToCheck, l.PreparedPost); err != nil {
		return err
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

	return s.listingRepo.UpdateListing(ctx, &l)
}

// DeleteListing deletes a listing. Only the listing owner may delete.
func (s *listingService) DeleteListing(ctx context.Context, userID int64, id int64) error {
	existing, err := s.listingRepo.GetListingByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return marketerrors.ErrNotFound
	}
	if existing.UserID != userID {
		return marketerrors.ErrUnauthorizedSide
	}
	return s.listingRepo.DeleteListing(ctx, id)
}

func (s *listingService) ListListingsByUserID(ctx context.Context, userID int64, typ *entity.ListingType) ([]*entity.Listing, error) {
	return s.listingRepo.ListListingsByUserID(ctx, userID, typ)
}

// ListListingsAll returns all listings, optionally filtered by type, categories, and min channel followers (for public discovery).
// Categories must be from the predefined set; invalid categories are ignored.
func (s *listingService) ListListingsAll(ctx context.Context, typ *entity.ListingType, categories []string, minFollowers *int64) ([]*entity.Listing, error) {
	validCategories := make([]string, 0, len(categories))
	for _, c := range categories {
		if c == "" {
			continue
		}
		if entity.ValidateListingCategories([]string{c}) == nil {
			validCategories = append(validCategories, c)
		}
	}
	return s.listingRepo.ListListingsAll(ctx, typ, validCategories, minFollowers)
}

// validateInstantPostListing checks instant_post constraints when the listing has that ad type.
func validateInstantPostListing(typ entity.ListingType, prices json.RawMessage, preparedPost json.RawMessage) error {
	if !entity.ListingHasAdType(prices, string(entity.AdTypeInstantPost)) {
		return nil
	}
	if typ != entity.ListingTypeLessee {
		return marketerrors.ErrInstantPostLesseeOnly
	}
	msg := entity.GetMessageFromDetails(preparedPost)
	if strings.TrimSpace(msg) == "" {
		return marketerrors.ErrInvalidInstantPost
	}
	return nil
}
