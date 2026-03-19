package deal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
	"ads-mrkt/internal/market/repository/deal/model"
	"ads-mrkt/internal/market/repository/pgxutil"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dealSelectCols is the standard column list for market.deal SELECT queries.
const dealSelectCols = "id, listing_id, lessor_id, lessee_id, channel_id, type, duration, price, escrow_amount, details, message, " +
	"lessor_signature, lessee_signature, status, escrow_address, escrow_release_time, " +
	"lessor_payout_address, lessee_payout_address, created_at, updated_at"

type database interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (context.Context, error)
	EndTx(ctx context.Context, err error, source string) error
}

type repository struct {
	db database
}

func New(db database) *repository {
	return &repository{db: db}
}

func (r *repository) CreateDeal(ctx context.Context, d *entity.Deal) error {
	rows, err := r.db.Query(ctx, `
		INSERT INTO market.deal (listing_id, lessor_id, lessee_id, channel_id, type, duration, price, escrow_amount, details, message, status,
		                         lessor_payout_address, lessee_payout_address, lessor_signature, lessee_signature)
		VALUES (@listing_id, @lessor_id, @lessee_id, @channel_id, @type, @duration, @price, @escrow_amount, @details, @message, @status,
		        @lessor_payout_address, @lessee_payout_address, @lessor_signature, @lessee_signature)
		RETURNING id, created_at, updated_at`,
		pgx.NamedArgs{
			"listing_id":            d.ListingID,
			"lessor_id":             d.LessorID,
			"lessee_id":             d.LesseeID,
			"channel_id":            d.ChannelID,
			"type":                  d.Type,
			"duration":              d.Duration,
			"price":                 d.Price,
			"escrow_amount":         d.EscrowAmount,
			"details":               d.Details,
			"message":               d.Message,
			"status":                d.Status,
			"lessor_payout_address": d.LessorPayoutAddress,
			"lessee_payout_address": d.LesseePayoutAddress,
			"lessor_signature":      d.LessorSignature,
			"lessee_signature":      d.LesseeSignature,
		})
	if err != nil {
		return fmt.Errorf("create deal: %w", err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.DealReturnRow])
	if err != nil {
		return fmt.Errorf("create deal: scan row: %w", err)
	}
	d.ID = row.ID
	d.CreatedAt = row.CreatedAt
	d.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *repository) GetDealByID(ctx context.Context, id int64) (*entity.Deal, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+dealSelectCols+" FROM market.deal WHERE id = @id",
		pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, fmt.Errorf("get deal by id %d: %w", id, err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.DealRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get deal by id %d: %w", id, err)
	}
	return model.DealRowToEntity(row), nil
}

func (r *repository) ListDealsApprovedWithoutEscrow(ctx context.Context) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+dealSelectCols+" FROM market.deal WHERE status = @status AND escrow_address IS NULL ORDER BY id ASC",
		pgx.NamedArgs{"status": string(entity.DealStatusApproved)})
	if err != nil {
		return nil, fmt.Errorf("list deals approved without escrow: %w", err)
	}
	defer rows.Close()

	list, err := pgxutil.CollectAndConvert(rows, model.DealRowToEntity)
	if err != nil {
		return nil, fmt.Errorf("list deals approved without escrow: %w", err)
	}
	return list, nil
}

func (r *repository) GetDealsByListingID(ctx context.Context, listingID int64) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+dealSelectCols+" FROM market.deal WHERE listing_id = @listing_id ORDER BY updated_at DESC",
		pgx.NamedArgs{"listing_id": listingID})
	if err != nil {
		return nil, fmt.Errorf("get deals by listing id %d: %w", listingID, err)
	}
	defer rows.Close()

	list, err := pgxutil.CollectAndConvert(rows, model.DealRowToEntity)
	if err != nil {
		return nil, fmt.Errorf("get deals by listing id %d: %w", listingID, err)
	}
	return list, nil
}

func (r *repository) GetDealsByListingIDForUser(ctx context.Context, listingID int64, userID int64) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+dealSelectCols+
			" FROM market.deal WHERE listing_id = @listing_id AND (lessor_id = @user_id OR lessee_id = @user_id) ORDER BY updated_at DESC",
		pgx.NamedArgs{"listing_id": listingID, "user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("get deals by listing id %d for user %d: %w", listingID, userID, err)
	}
	defer rows.Close()

	list, err := pgxutil.CollectAndConvert(rows, model.DealRowToEntity)
	if err != nil {
		return nil, fmt.Errorf("get deals by listing id %d for user %d: %w", listingID, userID, err)
	}
	return list, nil
}

func (r *repository) ListDealsWaitingEscrowRelease(ctx context.Context) ([]*entity.Deal, error) {
	return r.listDealsByStatus(ctx, entity.DealStatusWaitingEscrowRelease)
}

func (r *repository) ListDealsWaitingEscrowRefund(ctx context.Context) ([]*entity.Deal, error) {
	return r.listDealsByStatus(ctx, entity.DealStatusWaitingEscrowRefund)
}

func (r *repository) listDealsByStatus(ctx context.Context, status entity.DealStatus) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+dealSelectCols+" FROM market.deal WHERE status = @status ORDER BY id ASC",
		pgx.NamedArgs{"status": string(status)})
	if err != nil {
		return nil, fmt.Errorf("list deals by status %s: %w", status, err)
	}
	defer rows.Close()

	list, err := pgxutil.CollectAndConvert(rows, model.DealRowToEntity)
	if err != nil {
		return nil, fmt.Errorf("list deals by status %s: %w", status, err)
	}
	return list, nil
}

func (r *repository) ListDealsEscrowDepositConfirmedWithoutPostMessage(ctx context.Context) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.id, d.listing_id, d.lessor_id, d.lessee_id, d.channel_id, d.type, d.duration, d.price, d.escrow_amount, d.details, d.message,
		       d.lessor_signature, d.lessee_signature, d.status, d.escrow_address, d.escrow_release_time, d.lessor_payout_address, d.lessee_payout_address, d.created_at, d.updated_at
		FROM market.deal d
		LEFT JOIN market.deal_post_message dpm ON dpm.deal_id = d.id
		WHERE d.status = @status AND dpm.id IS NULL
		ORDER BY d.id
		LIMIT 50`,
		pgx.NamedArgs{"status": string(entity.DealStatusEscrowDepositConfirmed)})
	if err != nil {
		return nil, fmt.Errorf("list deals escrow deposit confirmed without post message: %w", err)
	}
	defer rows.Close()

	list, err := pgxutil.CollectAndConvert(rows, model.DealRowToEntity)
	if err != nil {
		return nil, fmt.Errorf("list deals escrow deposit confirmed without post message: %w", err)
	}
	return list, nil
}

func (r *repository) ListDealsByUserID(ctx context.Context, userID int64) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+dealSelectCols+
			" FROM market.deal WHERE lessor_id = @user_id OR lessee_id = @user_id ORDER BY updated_at DESC",
		pgx.NamedArgs{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("list deals by user id %d: %w", userID, err)
	}
	defer rows.Close()

	list, err := pgxutil.CollectAndConvert(rows, model.DealRowToEntity)
	if err != nil {
		return nil, fmt.Errorf("list deals by user id %d: %w", userID, err)
	}
	return list, nil
}

// UpdateDealDetailsAndClearSignatures updates only the details JSONB and clears both signatures.
// Only works in draft status.
func (r *repository) UpdateDealDetailsAndClearSignatures(ctx context.Context, dealID int64, details json.RawMessage) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal
		SET details = @details, lessor_signature = NULL, lessee_signature = NULL, updated_at = NOW()
		WHERE id = @id AND status = @status_draft`,
		pgx.NamedArgs{
			"id":           dealID,
			"details":      details,
			"status_draft": string(entity.DealStatusDraft),
		})
	if err != nil {
		return fmt.Errorf("update deal details for deal %d: %w", dealID, err)
	}
	return nil
}

func (r *repository) UpdateDealDraftFieldsAndClearSignatures(ctx context.Context, d *entity.Deal) (*entity.Deal, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE market.deal
		SET type = @type, duration = @duration, price = @price, escrow_amount = @escrow_amount, details = @details,
		    lessor_signature = NULL, lessee_signature = NULL, updated_at = NOW()
		WHERE id = @id AND status = @status_draft
		RETURNING `+dealSelectCols,
		pgx.NamedArgs{
			"id":            d.ID,
			"type":          d.Type,
			"duration":      d.Duration,
			"price":         d.Price,
			"escrow_amount": d.EscrowAmount,
			"details":       d.Details,
			"status_draft":  string(entity.DealStatusDraft),
		})
	if err != nil {
		return nil, fmt.Errorf("update deal draft fields for deal %d: %w", d.ID, err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.DealRow])
	if err != nil {
		return nil, fmt.Errorf("update deal draft fields for deal %d: %w", d.ID, err)
	}
	return model.DealRowToEntity(row), nil
}

func (r *repository) SetDealLessorSignature(ctx context.Context, dealID int64, sig string) error {
	_, err := r.db.Exec(ctx, `UPDATE market.deal SET lessor_signature = @sig, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{"sig": sig, "id": dealID})
	if err != nil {
		return fmt.Errorf("set lessor signature for deal %d: %w", dealID, err)
	}
	return nil
}

func (r *repository) SetDealLesseeSignature(ctx context.Context, dealID int64, sig string) error {
	_, err := r.db.Exec(ctx, `UPDATE market.deal SET lessee_signature = @sig, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{"sig": sig, "id": dealID})
	if err != nil {
		return fmt.Errorf("set lessee signature for deal %d: %w", dealID, err)
	}
	return nil
}

func (r *repository) SetDealPayoutAddress(ctx context.Context, dealID int64, userID int64, payoutAddressRaw string) (*entity.Deal, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE market.deal
		SET lessor_payout_address = CASE WHEN @user_id = lessor_id THEN @payout ELSE lessor_payout_address END,
		    lessee_payout_address = CASE WHEN @user_id = lessee_id THEN @payout ELSE lessee_payout_address END,
		    updated_at = NOW()
		WHERE id = @deal_id AND status = @status_draft AND (@user_id = lessor_id OR @user_id = lessee_id)
		RETURNING `+dealSelectCols,
		pgx.NamedArgs{"deal_id": dealID, "user_id": userID, "payout": payoutAddressRaw, "status_draft": string(entity.DealStatusDraft)})
	if err != nil {
		return nil, fmt.Errorf("set payout address for deal %d user %d: %w", dealID, userID, err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.DealRow])
	if err != nil {
		return nil, fmt.Errorf("set payout address for deal %d user %d: %w", dealID, userID, err)
	}
	return model.DealRowToEntity(row), nil
}

func (r *repository) SetDealStatusApproved(ctx context.Context, dealID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{
			"id":     dealID,
			"status": string(entity.DealStatusApproved),
		})
	if err != nil {
		return fmt.Errorf("set deal status approved for deal %d: %w", dealID, err)
	}
	return nil
}

func (r *repository) SignDealInTx(ctx context.Context, dealID int64, userID int64, sig string) (d *entity.Deal, err error) {
	txCtx, beginErr := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if beginErr != nil {
		return nil, fmt.Errorf("sign deal in tx %d: begin transaction: %w", dealID, beginErr)
	}
	defer func() {
		_ = r.db.EndTx(txCtx, err, "SignDealInTx")
	}()

	existing, err := r.GetDealByID(txCtx, dealID)
	if err != nil {
		return nil, fmt.Errorf("sign deal in tx %d: %w", dealID, err)
	}
	if existing == nil {
		return nil, marketerrors.ErrNotFound
	}
	if existing.Status != entity.DealStatusDraft {
		return nil, marketerrors.ErrDealNotDraft
	}
	if userID != existing.LessorID && userID != existing.LesseeID {
		return nil, marketerrors.ErrUnauthorizedSide
	}

	if userID == existing.LessorID {
		if err = r.SetDealLessorSignature(txCtx, dealID, sig); err != nil {
			return nil, fmt.Errorf("sign deal in tx %d: %w", dealID, err)
		}
		existing.LessorSignature = &sig
	} else {
		if err = r.SetDealLesseeSignature(txCtx, dealID, sig); err != nil {
			return nil, fmt.Errorf("sign deal in tx %d: %w", dealID, err)
		}
		existing.LesseeSignature = &sig
	}

	if entity.DealSignaturesMatch(existing) {
		if err = r.SetDealStatusApproved(txCtx, dealID); err != nil {
			return nil, err
		}
		existing.Status = entity.DealStatusApproved
	}
	return existing, nil
}

func (r *repository) SetDealEscrowAddress(ctx context.Context, dealID int64, address string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal
		SET escrow_address = @address, status = @status_waiting_escrow_deposit, updated_at = NOW()
		WHERE id = @id AND status = @status_approved`,
		pgx.NamedArgs{
			"address":                       address,
			"id":                            dealID,
			"status_approved":               string(entity.DealStatusApproved),
			"status_waiting_escrow_deposit": string(entity.DealStatusWaitingEscrowDeposit),
		})
	if err != nil {
		return fmt.Errorf("set escrow address for deal %d: %w", dealID, err)
	}
	return nil
}

func (r *repository) GetDealByEscrowAddress(ctx context.Context, escrowAddress string) (*entity.Deal, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+dealSelectCols+" FROM market.deal WHERE escrow_address = @escrow_address AND status = @status",
		pgx.NamedArgs{
			"escrow_address": escrowAddress,
			"status":         string(entity.DealStatusWaitingEscrowDeposit),
		})
	if err != nil {
		return nil, fmt.Errorf("get deal by escrow address: %w", err)
	}
	defer rows.Close()
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.DealRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get deal by escrow address: %w", err)
	}
	return model.DealRowToEntity(row), nil
}

func (r *repository) SetDealStatusExpiredByEscrowAddress(ctx context.Context, escrowAddress string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW()
		WHERE escrow_address = @escrow_address AND status = @status_waiting`,
		pgx.NamedArgs{
			"escrow_address": escrowAddress,
			"status":         string(entity.DealStatusExpired),
			"status_waiting": string(entity.DealStatusWaitingEscrowDeposit),
		})
	if err != nil {
		return fmt.Errorf("set deal status expired by escrow address: %w", err)
	}
	return nil
}

func (r *repository) ListDealsWaitingEscrowDepositOlderThan(ctx context.Context, before time.Time) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+dealSelectCols+" FROM market.deal WHERE status = @status AND updated_at < @before ORDER BY id ASC",
		pgx.NamedArgs{
			"status": string(entity.DealStatusWaitingEscrowDeposit),
			"before": before,
		})
	if err != nil {
		return nil, fmt.Errorf("list deals waiting escrow deposit older than %v: %w", before, err)
	}
	defer rows.Close()

	list, err := pgxutil.CollectAndConvert(rows, model.DealRowToEntity)
	if err != nil {
		return nil, fmt.Errorf("list deals waiting escrow deposit older than %v: %w", before, err)
	}
	return list, nil
}

func (r *repository) SetDealStatusExpiredByDealID(ctx context.Context, dealID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW()
		WHERE id = @id AND status = @status_waiting`,
		pgx.NamedArgs{
			"id":             dealID,
			"status":         string(entity.DealStatusExpired),
			"status_waiting": string(entity.DealStatusWaitingEscrowDeposit),
		})
	if err != nil {
		return fmt.Errorf("set deal status expired for deal %d: %w", dealID, err)
	}
	return nil
}

func (r *repository) ListDealsEscrowConfirmedToComplete(ctx context.Context) ([]*entity.Deal, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+dealSelectCols+" FROM market.deal WHERE status = @s1 OR status = @s2 ORDER BY id ASC",
		pgx.NamedArgs{
			"s1": string(entity.DealStatusEscrowReleaseConfirmed),
			"s2": string(entity.DealStatusEscrowRefundConfirmed),
		})
	if err != nil {
		return nil, fmt.Errorf("list deals escrow confirmed to complete: %w", err)
	}
	defer rows.Close()

	list, err := pgxutil.CollectAndConvert(rows, model.DealRowToEntity)
	if err != nil {
		return nil, fmt.Errorf("list deals escrow confirmed to complete: %w", err)
	}
	return list, nil
}

func (r *repository) SetDealStatusCompleted(ctx context.Context, dealID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW()
		WHERE id = @id AND (status = @s1 OR status = @s2)`,
		pgx.NamedArgs{
			"id":     dealID,
			"status": string(entity.DealStatusCompleted),
			"s1":     string(entity.DealStatusEscrowReleaseConfirmed),
			"s2":     string(entity.DealStatusEscrowRefundConfirmed),
		})
	if err != nil {
		return fmt.Errorf("set deal status completed for deal %d: %w", dealID, err)
	}
	return nil
}

func (r *repository) SetDealStatusEscrowDepositConfirmed(ctx context.Context, dealID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW()
		WHERE id = @id AND status = @status_waiting`,
		pgx.NamedArgs{
			"id":             dealID,
			"status":         string(entity.DealStatusEscrowDepositConfirmed),
			"status_waiting": string(entity.DealStatusWaitingEscrowDeposit),
		})
	if err != nil {
		return fmt.Errorf("set deal status escrow deposit confirmed for deal %d: %w", dealID, err)
	}
	return nil
}

func (r *repository) SetDealStatusEscrowReleaseConfirmed(ctx context.Context, dealID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW()
		WHERE id = @id AND status = @status_waiting`,
		pgx.NamedArgs{
			"id":             dealID,
			"status":         string(entity.DealStatusEscrowReleaseConfirmed),
			"status_waiting": string(entity.DealStatusWaitingEscrowRelease),
		})
	if err != nil {
		return fmt.Errorf("set deal status escrow release confirmed for deal %d: %w", dealID, err)
	}
	return nil
}

func (r *repository) SetDealStatusEscrowRefundConfirmed(ctx context.Context, dealID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW()
		WHERE id = @id AND status = @status_waiting`,
		pgx.NamedArgs{
			"id":             dealID,
			"status":         string(entity.DealStatusEscrowRefundConfirmed),
			"status_waiting": string(entity.DealStatusWaitingEscrowRefund),
		})
	if err != nil {
		return fmt.Errorf("set deal status escrow refund confirmed for deal %d: %w", dealID, err)
	}
	return nil
}

func (r *repository) RejectDealsByUserAndChannel(ctx context.Context, userID, channelID int64) ([]model.RejectedDealRow, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE market.deal SET status = @status_rejected, updated_at = NOW()
		WHERE channel_id = @channel_id AND lessor_id = @user_id
		  AND (status = @status_draft OR status = @status_approved)
		RETURNING id, lessor_id, lessee_id`,
		pgx.NamedArgs{
			"channel_id":      channelID,
			"user_id":         userID,
			"status_rejected": string(entity.DealStatusRejected),
			"status_draft":    string(entity.DealStatusDraft),
			"status_approved": string(entity.DealStatusApproved),
		})
	if err != nil {
		return nil, fmt.Errorf("reject deals by user %d channel %d: %w", userID, channelID, err)
	}
	defer rows.Close()
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.RejectedDealRow])
	if err != nil {
		return nil, fmt.Errorf("reject deals by user %d channel %d: %w", userID, channelID, err)
	}
	return result, nil
}

func (r *repository) RejectDealsByChannel(ctx context.Context, channelID int64) ([]model.RejectedDealRow, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE market.deal SET status = @status_rejected, updated_at = NOW()
		WHERE channel_id = @channel_id
		  AND (status = @status_draft OR status = @status_approved)
		RETURNING id, lessor_id, lessee_id`,
		pgx.NamedArgs{
			"channel_id":      channelID,
			"status_rejected": string(entity.DealStatusRejected),
			"status_draft":    string(entity.DealStatusDraft),
			"status_approved": string(entity.DealStatusApproved),
		})
	if err != nil {
		return nil, fmt.Errorf("reject deals by channel %d: %w", channelID, err)
	}
	defer rows.Close()
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.RejectedDealRow])
	if err != nil {
		return nil, fmt.Errorf("reject deals by channel %d: %w", channelID, err)
	}
	return result, nil
}

func (r *repository) SetDealStatusRejected(ctx context.Context, dealID int64) (*entity.Deal, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE market.deal SET status = @status, updated_at = NOW()
		WHERE id = @id AND status = @status_draft
		RETURNING `+dealSelectCols,
		pgx.NamedArgs{
			"id":           dealID,
			"status":       string(entity.DealStatusRejected),
			"status_draft": string(entity.DealStatusDraft),
		})
	if err != nil {
		return nil, fmt.Errorf("set deal status rejected for deal %d: %w", dealID, err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.DealRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("set deal status rejected for deal %d: %w", dealID, err)
	}
	return model.DealRowToEntity(row), nil
}
