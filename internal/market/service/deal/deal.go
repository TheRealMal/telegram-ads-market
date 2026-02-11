package service

import (
	"context"

	"ads-mrkt/internal/market/domain"
	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
)

func (s *DealService) CreateDeal(ctx context.Context, d *entity.Deal) error {
	d.Status = entity.DealStatusDraft
	d.EscrowAmount = domain.ComputeEscrowAmount(d.Price, s.escrowConfig)
	return s.dealRepo.CreateDeal(ctx, d)
}

func (s *DealService) GetDeal(ctx context.Context, id int64) (*entity.Deal, error) {
	return s.dealRepo.GetDealByID(ctx, id)
}

func (s *DealService) GetDealsByListingID(ctx context.Context, listingID int64) ([]*entity.Deal, error) {
	return s.dealRepo.GetDealsByListingID(ctx, listingID)
}

func (s *DealService) GetDealsByUserID(ctx context.Context, userID int64) ([]*entity.Deal, error) {
	return s.dealRepo.ListDealsByUserID(ctx, userID)
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
	d.EscrowAmount = domain.ComputeEscrowAmount(d.Price, s.escrowConfig)
	return s.dealRepo.UpdateDealDraftFieldsAndClearSignatures(ctx, d)
}

// SignDeal sets the current user's signature (hash of type, duration, price, details, user_id, payout addresses) in a transaction.
// Both payout addresses must already be set on the deal; user's wallet must match their deal payout. Then both signatures use the same payload.
func (s *DealService) SignDeal(ctx context.Context, userID int64, dealID int64) error {
	existing, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil || existing == nil {
		return marketerrors.ErrNotFound
	}
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return marketerrors.ErrNotFound
	}
	if user.WalletAddress == nil || *user.WalletAddress == "" {
		return marketerrors.ErrWalletNotSet
	}
	if existing.LessorPayoutAddress == nil || *existing.LessorPayoutAddress == "" ||
		existing.LesseePayoutAddress == nil || *existing.LesseePayoutAddress == "" {
		return marketerrors.ErrPayoutNotSet
	}
	myPayout := *existing.LesseePayoutAddress
	if userID == existing.LessorID {
		myPayout = *existing.LessorPayoutAddress
	}
	if *user.WalletAddress != myPayout {
		return marketerrors.ErrWalletNotSet // wallet does not match deal payout
	}
	lessorPayout := *existing.LessorPayoutAddress
	lesseePayout := *existing.LesseePayoutAddress
	sig := domain.ComputeDealSignature(existing.Type, existing.Duration, existing.Price, existing.Details, userID, lessorPayout, lesseePayout)
	return s.dealRepo.SignDealInTx(ctx, dealID, userID, sig)
}

// SetDealPayoutAddress sets the current user's payout address on the deal (lessor or lessee). Only in draft.
func (s *DealService) SetDealPayoutAddress(ctx context.Context, userID int64, dealID int64, payoutAddressRaw string) error {
	existing, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil || existing == nil {
		return marketerrors.ErrNotFound
	}
	if existing.Status != entity.DealStatusDraft {
		return marketerrors.ErrDealNotDraft
	}
	if userID != existing.LessorID && userID != existing.LesseeID {
		return marketerrors.ErrUnauthorizedSide
	}
	if payoutAddressRaw == "" {
		return marketerrors.ErrWalletNotSet
	}
	return s.dealRepo.SetDealPayoutAddress(ctx, dealID, userID, payoutAddressRaw)
}

// RejectDeal sets deal status to rejected. Only allowed when status is draft; caller must be lessor or lessee.
func (s *DealService) RejectDeal(ctx context.Context, userID int64, dealID int64) error {
	existing, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil || existing == nil {
		return marketerrors.ErrNotFound
	}
	if userID != existing.LessorID && userID != existing.LesseeID {
		return marketerrors.ErrUnauthorizedSide
	}
	updated, err := s.dealRepo.SetDealStatusRejected(ctx, dealID)
	if err != nil {
		return err
	}
	if !updated {
		return marketerrors.ErrDealNotDraft
	}
	return nil
}
