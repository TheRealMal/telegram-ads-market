package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"ads-mrkt/internal/market/domain"
	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"

	"github.com/jackc/pgx/v5"
)

type dealRow struct {
	ID                int64           `db:"id"`
	ListingID         int64           `db:"listing_id"`
	LessorID          int64           `db:"lessor_id"`
	LesseeID          int64           `db:"lessee_id"`
	Type              string          `db:"type"`
	Duration          int64           `db:"duration"`
	Price             int64           `db:"price"`
	Details           json.RawMessage `db:"details"`
	LessorSignature   *string         `db:"lessor_signature"`
	LesseeSignature   *string         `db:"lessee_signature"`
	Status            string          `db:"status"`
	EscrowAddress     *string         `db:"escrow_address"`
	EscrowPrivateKey  *string         `db:"escrow_private_key"`
	EscrowReleaseTime *time.Time      `db:"escrow_release_time"`
	CreatedAt         time.Time       `db:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at"`
}

type dealReturnRow struct {
	ID        int64     `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func dealRowToEntity(row dealRow) *entity.Deal {
	return &entity.Deal{
		ID:                row.ID,
		ListingID:         row.ListingID,
		LessorID:          row.LessorID,
		LesseeID:          row.LesseeID,
		Type:              row.Type,
		Duration:          row.Duration,
		Price:             row.Price,
		Details:           row.Details,
		LessorSignature:   row.LessorSignature,
		LesseeSignature:   row.LesseeSignature,
		Status:            entity.DealStatus(row.Status),
		EscrowAddress:     row.EscrowAddress,
		EscrowPrivateKey:  row.EscrowPrivateKey,
		EscrowReleaseTime: row.EscrowReleaseTime,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func (r *repository) CreateDeal(ctx context.Context, d *entity.Deal) error {
	rows, err := r.db.Query(ctx, `
		INSERT INTO market.deal (listing_id, lessor_id, lessee_id, type, duration, price, details, status)
		VALUES (@listing_id, @lessor_id, @lessee_id, @type, @duration, @price, @details, @status)
		RETURNING id, created_at, updated_at`,
		pgx.NamedArgs{
			"listing_id": d.ListingID,
			"lessor_id":  d.LessorID,
			"lessee_id":  d.LesseeID,
			"type":       d.Type,
			"duration":   d.Duration,
			"price":      d.Price,
			"details":    d.Details,
			"status":     d.Status,
		})
	if err != nil {
		return err
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dealReturnRow])
	if err != nil {
		return err
	}
	d.ID = row.ID
	d.CreatedAt = row.CreatedAt
	d.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *repository) GetDealByID(ctx context.Context, id int64) (*entity.Deal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, listing_id, lessor_id, lessee_id, type, duration, price, details,
		       lessor_signature, lessee_signature, status, escrow_address, escrow_private_key, escrow_release_time, created_at, updated_at
		FROM market.deal WHERE id = @id`,
		pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dealRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return dealRowToEntity(row), nil
}

// ListDealsApprovedWithoutEscrow returns deals with status approved and no escrow_address set (for escrow worker).
func (r *repository) ListDealsApprovedWithoutEscrow(ctx context.Context) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, listing_id, lessor_id, lessee_id, type, duration, price, details,
		       lessor_signature, lessee_signature, status, escrow_address, escrow_private_key, escrow_release_time, created_at, updated_at
		FROM market.deal
		WHERE status = @status AND escrow_address IS NULL
		ORDER BY id ASC`,
		pgx.NamedArgs{"status": string(entity.DealStatusApproved)})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[dealRow])
	if err != nil {
		return nil, err
	}
	list := make([]*entity.Deal, 0, len(slice))
	for _, row := range slice {
		list = append(list, dealRowToEntity(row))
	}
	return list, nil
}

func (r *repository) GetDealsByListingID(ctx context.Context, listingID int64) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, listing_id, lessor_id, lessee_id, type, duration, price, details,
		       lessor_signature, lessee_signature, status, escrow_address, escrow_private_key, escrow_release_time, created_at, updated_at
		FROM market.deal WHERE listing_id = @listing_id ORDER BY updated_at DESC`,
		pgx.NamedArgs{"listing_id": listingID})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[dealRow])
	if err != nil {
		return nil, err
	}
	list := make([]*entity.Deal, 0, len(slice))
	for _, row := range slice {
		list = append(list, dealRowToEntity(row))
	}
	return list, nil
}

// UpdateDealDraftFields updates type, duration, price, details and sets both signatures to NULL.
// Call only when status is draft. Status filter is from domain (entity.DealStatusDraft).
func (r *repository) UpdateDealDraftFieldsAndClearSignatures(ctx context.Context, d *entity.Deal) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal
		SET type = @type, duration = @duration, price = @price, details = @details,
		    lessor_signature = NULL, lessee_signature = NULL, updated_at = NOW()
		WHERE id = @id AND status = @status_draft`,
		pgx.NamedArgs{
			"id":           d.ID,
			"type":         d.Type,
			"duration":     d.Duration,
			"price":        d.Price,
			"details":      d.Details,
			"status_draft": string(entity.DealStatusDraft),
		})
	return err
}

func (r *repository) SetDealLessorSignature(ctx context.Context, dealID int64, sig string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET lessor_signature = @sig, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{"sig": sig, "id": dealID})
	return err
}

func (r *repository) SetDealLesseeSignature(ctx context.Context, dealID int64, sig string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET lessee_signature = @sig, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{"sig": sig, "id": dealID})
	return err
}

// SetDealStatusApproved sets deal status to approved. Status value from domain (entity.DealStatusApproved).
func (r *repository) SetDealStatusApproved(ctx context.Context, dealID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{
			"id":     dealID,
			"status": string(entity.DealStatusApproved),
		})
	return err
}

// SignDealInTx runs sign-deal steps in a single transaction: get deal, set lessor or lessee signature,
// re-fetch deal, set status to approved if both signatures match.
func (r *repository) SignDealInTx(ctx context.Context, dealID int64, userID int64, sig string) (err error) {
	txCtx, beginErr := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if beginErr != nil {
		return beginErr
	}
	defer func() {
		_ = r.db.EndTx(txCtx, err, "SignDealInTx")
	}()

	existing, err := r.GetDealByID(txCtx, dealID)
	if err != nil {
		return err
	}
	if existing == nil {
		return marketerrors.ErrNotFound
	}
	if existing.Status != entity.DealStatusDraft {
		return marketerrors.ErrDealNotDraft
	}
	if userID != existing.LessorID && userID != existing.LesseeID {
		return marketerrors.ErrUnauthorizedSide
	}

	if userID == existing.LessorID {
		if err = r.SetDealLessorSignature(txCtx, dealID, sig); err != nil {
			return err
		}
	} else {
		if err = r.SetDealLesseeSignature(txCtx, dealID, sig); err != nil {
			return err
		}
	}

	updated, err := r.GetDealByID(txCtx, dealID)
	if err != nil || updated == nil {
		return err
	}
	// Both parties must have signed the current terms (signature includes user_id).
	lessorSig := domain.ComputeDealSignature(updated.Type, updated.Duration, updated.Price, updated.Details, updated.LessorID)
	lesseeSig := domain.ComputeDealSignature(updated.Type, updated.Duration, updated.Price, updated.Details, updated.LesseeID)
	if updated.LessorSignature != nil && updated.LesseeSignature != nil &&
		*updated.LessorSignature == lessorSig && *updated.LesseeSignature == lesseeSig {
		return r.SetDealStatusApproved(txCtx, dealID)
	}
	return nil
}

// SetDealEscrowAddress sets escrow address and private key when deal status is approved.
// Status filter from domain (entity.DealStatusApproved).
func (r *repository) SetDealEscrowAddress(ctx context.Context, dealID int64, address string, privateKey string, releaseTime time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal
		SET escrow_address = @address, escrow_private_key = @private_key, escrow_release_time = @release_time, status = @status_waiting_escrow_deposit, updated_at = NOW()
		WHERE id = @id AND status = @status_approved`,
		pgx.NamedArgs{
			"address":                       address,
			"private_key":                   privateKey,
			"release_time":                  releaseTime,
			"id":                            dealID,
			"status_approved":               string(entity.DealStatusApproved),
			"status_waiting_escrow_deposit": string(entity.DealStatusWaitingEscrowDeposit),
		})
	return err
}
