package service

import (
	"context"

	"ads-mrkt/internal/market/domain"
	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
)

func (s *DealService) CreateDeal(ctx context.Context, d *entity.Deal) error {
	d.Status = entity.DealStatusDraft
	return s.dealRepo.CreateDeal(ctx, d)
}

func (s *DealService) GetDeal(ctx context.Context, id int64) (*entity.Deal, error) {
	return s.dealRepo.GetDealByID(ctx, id)
}

func (s *DealService) GetDealsByListingID(ctx context.Context, listingID int64) ([]*entity.Deal, error) {
	return s.dealRepo.GetDealsByListingID(ctx, listingID)
}

// UpdateDealDraft updates type, duration, price, details when status is draft. Clears both signatures.
// Caller must be lessor or lessee.
func (s *DealService) UpdateDealDraft(ctx context.Context, userID int64, d *entity.Deal) error {
	existing, err := s.dealRepo.GetDealByID(ctx, d.ID)
	if err != nil || existing == nil {
		return marketerrors.ErrNotFound
	}
	if existing.Status != entity.DealStatusDraft {
		return marketerrors.ErrDealNotDraft
	}
	if userID != existing.LessorID && userID != existing.LesseeID {
		return marketerrors.ErrUnauthorizedSide
	}
	d.LessorID = existing.LessorID
	d.LesseeID = existing.LesseeID
	d.ListingID = existing.ListingID
	d.Status = entity.DealStatusDraft
	return s.dealRepo.UpdateDealDraftFieldsAndClearSignatures(ctx, d)
}

// SignDeal sets the current user's signature (hash of type, duration, price, details, user_id) in a transaction.
// If both parties have signed the current terms, status is set to approved.
func (s *DealService) SignDeal(ctx context.Context, userID int64, dealID int64) error {
	existing, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil || existing == nil {
		return marketerrors.ErrNotFound
	}
	sig := domain.ComputeDealSignature(existing.Type, existing.Duration, existing.Price, existing.Details, userID)
	return s.dealRepo.SignDealInTx(ctx, dealID, userID, sig)
}
