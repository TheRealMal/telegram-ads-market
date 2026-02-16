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
	ID                  int64           `db:"id"`
	ListingID           int64           `db:"listing_id"`
	LessorID            int64           `db:"lessor_id"`
	LesseeID            int64           `db:"lessee_id"`
	ChannelID           *int64          `db:"channel_id"`
	Type                string          `db:"type"`
	Duration            int64           `db:"duration"`
	Price               int64           `db:"price"`
	EscrowAmount        int64           `db:"escrow_amount"`
	Details             json.RawMessage `db:"details"`
	LessorSignature     *string         `db:"lessor_signature"`
	LesseeSignature     *string         `db:"lessee_signature"`
	Status              string          `db:"status"`
	EscrowAddress       *string         `db:"escrow_address"`
	EscrowPrivateKey    *string         `db:"escrow_private_key"`
	EscrowReleaseTime   *time.Time      `db:"escrow_release_time"`
	LessorPayoutAddress *string         `db:"lessor_payout_address"`
	LesseePayoutAddress *string         `db:"lessee_payout_address"`
	CreatedAt           time.Time       `db:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at"`
}

type dealReturnRow struct {
	ID        int64     `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func dealRowToEntity(row dealRow) *entity.Deal {
	return &entity.Deal{
		ID:                  row.ID,
		ListingID:           row.ListingID,
		LessorID:            row.LessorID,
		LesseeID:            row.LesseeID,
		ChannelID:           row.ChannelID,
		Type:                row.Type,
		Duration:            row.Duration,
		Price:               row.Price,
		EscrowAmount:        row.EscrowAmount,
		Details:             row.Details,
		LessorSignature:     row.LessorSignature,
		LesseeSignature:     row.LesseeSignature,
		Status:              entity.DealStatus(row.Status),
		EscrowAddress:       row.EscrowAddress,
		EscrowPrivateKey:    row.EscrowPrivateKey,
		EscrowReleaseTime:   row.EscrowReleaseTime,
		LessorPayoutAddress: row.LessorPayoutAddress,
		LesseePayoutAddress: row.LesseePayoutAddress,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func (r *repository) CreateDeal(ctx context.Context, d *entity.Deal) error {
	rows, err := r.db.Query(ctx, `
		INSERT INTO market.deal (listing_id, lessor_id, lessee_id, channel_id, type, duration, price, escrow_amount, details, status)
		VALUES (@listing_id, @lessor_id, @lessee_id, @channel_id, @type, @duration, @price, @escrow_amount, @details, @status)
		RETURNING id, created_at, updated_at`,
		pgx.NamedArgs{
			"listing_id":    d.ListingID,
			"lessor_id":     d.LessorID,
			"lessee_id":     d.LesseeID,
			"channel_id":    d.ChannelID,
			"type":          d.Type,
			"duration":      d.Duration,
			"price":         d.Price,
			"escrow_amount": d.EscrowAmount,
			"details":       d.Details,
			"status":        d.Status,
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
		SELECT id, listing_id, lessor_id, lessee_id, channel_id, type, duration, price, escrow_amount, details,
		       lessor_signature, lessee_signature, status, escrow_address, escrow_private_key, escrow_release_time, lessor_payout_address, lessee_payout_address, created_at, updated_at
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
		SELECT id, listing_id, lessor_id, lessee_id, channel_id, type, duration, price, escrow_amount, details,
		       lessor_signature, lessee_signature, status, escrow_address, escrow_private_key, escrow_release_time, lessor_payout_address, lessee_payout_address, created_at, updated_at
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
		SELECT id, listing_id, lessor_id, lessee_id, channel_id, type, duration, price, escrow_amount, details,
		       lessor_signature, lessee_signature, status, escrow_address, escrow_private_key, escrow_release_time, lessor_payout_address, lessee_payout_address, created_at, updated_at
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

// GetDealsByListingIDForUser returns deals for the given listing where the user is lessor or lessee.
func (r *repository) GetDealsByListingIDForUser(ctx context.Context, listingID int64, userID int64) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, listing_id, lessor_id, lessee_id, channel_id, type, duration, price, escrow_amount, details,
		       lessor_signature, lessee_signature, status, escrow_address, escrow_private_key, escrow_release_time, lessor_payout_address, lessee_payout_address, created_at, updated_at
		FROM market.deal
		WHERE listing_id = @listing_id AND (lessor_id = @user_id OR lessee_id = @user_id)
		ORDER BY updated_at DESC`,
		pgx.NamedArgs{"listing_id": listingID, "user_id": userID})
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

// ListDealsWaitingEscrowRelease returns deals in status waiting_escrow_release (for release worker).
func (r *repository) ListDealsWaitingEscrowRelease(ctx context.Context) ([]*entity.Deal, error) {
	return r.listDealsByStatus(ctx, entity.DealStatusWaitingEscrowRelease)
}

// ListDealsWaitingEscrowRefund returns deals in status waiting_escrow_refund (for refund worker).
func (r *repository) ListDealsWaitingEscrowRefund(ctx context.Context) ([]*entity.Deal, error) {
	return r.listDealsByStatus(ctx, entity.DealStatusWaitingEscrowRefund)
}

func (r *repository) listDealsByStatus(ctx context.Context, status entity.DealStatus) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, listing_id, lessor_id, lessee_id, channel_id, type, duration, price, escrow_amount, details,
		       lessor_signature, lessee_signature, status, escrow_address, escrow_private_key, escrow_release_time, lessor_payout_address, lessee_payout_address, created_at, updated_at
		FROM market.deal
		WHERE status = @status
		ORDER BY id ASC`,
		pgx.NamedArgs{"status": string(status)})
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

// ListDealsEscrowDepositConfirmedWithoutPostMessage returns deals with status escrow_deposit_confirmed that do not yet have a deal_post_message row.
func (r *repository) ListDealsEscrowDepositConfirmedWithoutPostMessage(ctx context.Context) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.id, d.listing_id, d.lessor_id, d.lessee_id, d.channel_id, d.type, d.duration, d.price, d.escrow_amount, d.details,
		       d.lessor_signature, d.lessee_signature, d.status, d.escrow_address, d.escrow_private_key, d.escrow_release_time, d.lessor_payout_address, d.lessee_payout_address, d.created_at, d.updated_at
		FROM market.deal d
		LEFT JOIN market.deal_post_message dpm ON dpm.deal_id = d.id
		WHERE d.status = @status AND dpm.id IS NULL
		ORDER BY d.id`,
		pgx.NamedArgs{"status": string(entity.DealStatusEscrowDepositConfirmed)})
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

// ListDealsByUserID returns all deals where the user is lessor or lessee, ordered by updated_at DESC.
func (r *repository) ListDealsByUserID(ctx context.Context, userID int64) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, listing_id, lessor_id, lessee_id, channel_id, type, duration, price, escrow_amount, details,
		       lessor_signature, lessee_signature, status, escrow_address, escrow_private_key, escrow_release_time, lessor_payout_address, lessee_payout_address, created_at, updated_at
		FROM market.deal
		WHERE lessor_id = @user_id OR lessee_id = @user_id
		ORDER BY updated_at DESC`,
		pgx.NamedArgs{"user_id": userID})
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
		SET type = @type, duration = @duration, price = @price, escrow_amount = @escrow_amount, details = @details,
		    lessor_signature = NULL, lessee_signature = NULL, updated_at = NOW()
		WHERE id = @id AND status = @status_draft`,
		pgx.NamedArgs{
			"id":            d.ID,
			"type":          d.Type,
			"duration":      d.Duration,
			"price":         d.Price,
			"escrow_amount": d.EscrowAmount,
			"details":       d.Details,
			"status_draft":  string(entity.DealStatusDraft),
		})
	return err
}

func (r *repository) SetDealLessorSignature(ctx context.Context, dealID int64, sig string) error {
	_, err := r.db.Exec(ctx, `UPDATE market.deal SET lessor_signature = @sig, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{"sig": sig, "id": dealID})
	return err
}

func (r *repository) SetDealLesseeSignature(ctx context.Context, dealID int64, sig string) error {
	_, err := r.db.Exec(ctx, `UPDATE market.deal SET lessee_signature = @sig, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{"sig": sig, "id": dealID})
	return err
}

func (r *repository) SetDealPayoutAddress(ctx context.Context, dealID int64, userID int64, payoutAddressRaw string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal
		SET lessor_payout_address = CASE WHEN @user_id = lessor_id THEN @payout ELSE lessor_payout_address END,
		    lessee_payout_address = CASE WHEN @user_id = lessee_id THEN @payout ELSE lessee_payout_address END,
		    updated_at = NOW()
		WHERE id = @deal_id AND status = @status_draft AND (@user_id = lessor_id OR @user_id = lessee_id)`,
		pgx.NamedArgs{"deal_id": dealID, "user_id": userID, "payout": payoutAddressRaw, "status_draft": string(entity.DealStatusDraft)})
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

// SignDealInTx runs sign-deal steps in a single transaction: set signer's signature,
// re-fetch deal, set status to approved if both signatures match (same payout payload).
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
	lessorPayout := ""
	if updated.LessorPayoutAddress != nil {
		lessorPayout = *updated.LessorPayoutAddress
	}
	lesseePayout := ""
	if updated.LesseePayoutAddress != nil {
		lesseePayout = *updated.LesseePayoutAddress
	}
	lessorSig := domain.ComputeDealSignature(updated.Type, updated.Duration, updated.Price, updated.Details, updated.LessorID, lessorPayout, lesseePayout)
	lesseeSig := domain.ComputeDealSignature(updated.Type, updated.Duration, updated.Price, updated.Details, updated.LesseeID, lessorPayout, lesseePayout)
	if updated.LessorSignature != nil && updated.LesseeSignature != nil &&
		*updated.LessorSignature == lessorSig && *updated.LesseeSignature == lesseeSig {
		return r.SetDealStatusApproved(txCtx, dealID)
	}
	return nil
}

// SetDealEscrowAddress sets escrow address and private key when deal status is approved.
// Escrow release time is not set on creation (handled by deal_post_message.until_ts).
func (r *repository) SetDealEscrowAddress(ctx context.Context, dealID int64, address string, privateKey string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal
		SET escrow_address = @address, escrow_private_key = @private_key, status = @status_waiting_escrow_deposit, updated_at = NOW()
		WHERE id = @id AND status = @status_approved`,
		pgx.NamedArgs{
			"address":                       address,
			"private_key":                   privateKey,
			"id":                            dealID,
			"status_approved":               string(entity.DealStatusApproved),
			"status_waiting_escrow_deposit": string(entity.DealStatusWaitingEscrowDeposit),
		})
	return err
}

// GetDealByEscrowAddress returns the deal with the given escrow address and status waiting_escrow_deposit, or nil.
func (r *repository) GetDealByEscrowAddress(ctx context.Context, escrowAddress string) (*entity.Deal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, listing_id, lessor_id, lessee_id, channel_id, type, duration, price, escrow_amount, details,
		       lessor_signature, lessee_signature, status, escrow_address, escrow_private_key, escrow_release_time, lessor_payout_address, lessee_payout_address, created_at, updated_at
		FROM market.deal
		WHERE escrow_address = @escrow_address AND status = @status`,
		pgx.NamedArgs{
			"escrow_address": escrowAddress,
			"status":         string(entity.DealStatusWaitingEscrowDeposit),
		})
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

// SetDealStatusExpiredByEscrowAddress sets deal status to expired for the deal with the given escrow address (if in waiting_escrow_deposit).
func (r *repository) SetDealStatusExpiredByEscrowAddress(ctx context.Context, escrowAddress string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW()
		WHERE escrow_address = @escrow_address AND status = @status_waiting`,
		pgx.NamedArgs{
			"escrow_address": escrowAddress,
			"status":         string(entity.DealStatusExpired),
			"status_waiting": string(entity.DealStatusWaitingEscrowDeposit),
		})
	return err
}

// SetDealStatusEscrowDepositConfirmed sets deal status to escrow_deposit_confirmed by deal ID.
func (r *repository) SetDealStatusEscrowDepositConfirmed(ctx context.Context, dealID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW()
		WHERE id = @id AND status = @status_waiting`,
		pgx.NamedArgs{
			"id":             dealID,
			"status":         string(entity.DealStatusEscrowDepositConfirmed),
			"status_waiting": string(entity.DealStatusWaitingEscrowDeposit),
		})
	return err
}

// SetDealStatusEscrowReleaseConfirmed sets deal status to escrow_release_confirmed. Call after successful release.
func (r *repository) SetDealStatusEscrowReleaseConfirmed(ctx context.Context, dealID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW()
		WHERE id = @id AND status = @status_waiting`,
		pgx.NamedArgs{
			"id":             dealID,
			"status":         string(entity.DealStatusEscrowReleaseConfirmed),
			"status_waiting": string(entity.DealStatusWaitingEscrowRelease),
		})
	return err
}

// SetDealStatusEscrowRefundConfirmed sets deal status to escrow_refund_confirmed. Call after successful refund.
func (r *repository) SetDealStatusEscrowRefundConfirmed(ctx context.Context, dealID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW()
		WHERE id = @id AND status = @status_waiting`,
		pgx.NamedArgs{
			"id":             dealID,
			"status":         string(entity.DealStatusEscrowRefundConfirmed),
			"status_waiting": string(entity.DealStatusWaitingEscrowRefund),
		})
	return err
}

// SetDealStatusRejected sets deal status to rejected only when current status is draft. Returns true if a row was updated.
func (r *repository) SetDealStatusRejected(ctx context.Context, dealID int64) (bool, error) {
	cmd, err := r.db.Exec(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW()
		WHERE id = @id AND status = @status_draft`,
		pgx.NamedArgs{
			"id":           dealID,
			"status":       string(entity.DealStatusRejected),
			"status_draft": string(entity.DealStatusDraft),
		})
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}
