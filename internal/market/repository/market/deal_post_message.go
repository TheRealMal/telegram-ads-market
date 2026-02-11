package repository

import (
	"context"
	"errors"
	"time"

	"ads-mrkt/internal/market/domain/entity"

	"github.com/jackc/pgx/v5"
)

func (r *repository) CreateDealPostMessage(ctx context.Context, m *entity.DealPostMessage) error {
	rows, err := r.db.Query(ctx, `
		INSERT INTO market.deal_post_message (deal_id, channel_id, message_id, post_message, status, next_check, until_ts)
		VALUES (@deal_id, @channel_id, @message_id, @post_message, @status, @next_check, @until_ts)
		ON CONFLICT (deal_id) DO NOTHING
		RETURNING id, created_at, updated_at`,
		pgx.NamedArgs{
			"deal_id":      m.DealID,
			"channel_id":   m.ChannelID,
			"message_id":   m.MessageID,
			"post_message": m.PostMessage,
			"status":       string(m.Status),
			"next_check":   m.NextCheck,
			"until_ts":     m.UntilTs,
		})
	if err != nil {
		return err
	}
	defer rows.Close()
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[struct {
		ID        int64     `db:"id"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // conflict, row already exists
		}
		return err
	}
	m.ID = row.ID
	m.CreatedAt = row.CreatedAt
	m.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *repository) UpdateDealPostMessageStatus(ctx context.Context, id int64, status entity.DealPostMessageStatus) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal_post_message SET status = @status, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{"id": id, "status": string(status)})
	return err
}

func (r *repository) UpdateDealPostMessageStatusAndNextCheck(ctx context.Context, id int64, status entity.DealPostMessageStatus, nextCheck time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal_post_message SET status = @status, next_check = @next_check, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{"id": id, "status": string(status), "next_check": nextCheck})
	return err
}

func (r *repository) ListDealPostMessageExistsWithNextCheckBefore(ctx context.Context, before time.Time) ([]*entity.DealPostMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, deal_id, channel_id, message_id, post_message, status, next_check, until_ts, created_at, updated_at
		FROM market.deal_post_message
		WHERE status = 'exists' AND next_check <= @before
		ORDER BY id`,
		pgx.NamedArgs{"before": before})
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*entity.DealPostMessage, error) {
		var id, dealID, channelID, messageID int64
		var postMessage, status string
		var nextCheck, untilTs, createdAt, updatedAt time.Time
		err := row.Scan(&id, &dealID, &channelID, &messageID, &postMessage, &status, &nextCheck, &untilTs, &createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}
		return &entity.DealPostMessage{
			ID:          id,
			DealID:      dealID,
			ChannelID:   channelID,
			MessageID:   messageID,
			PostMessage: postMessage,
			Status:      entity.DealPostMessageStatus(status),
			NextCheck:   nextCheck,
			UntilTs:     untilTs,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}, nil
	})
}

func (r *repository) ListDealPostMessageByStatus(ctx context.Context, status entity.DealPostMessageStatus) ([]*entity.DealPostMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, deal_id, channel_id, message_id, post_message, status, next_check, until_ts, created_at, updated_at
		FROM market.deal_post_message
		WHERE status = @status
		ORDER BY id`,
		pgx.NamedArgs{"status": string(status)})
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*entity.DealPostMessage, error) {
		var id, dealID, channelID, messageID int64
		var postMessage, st string
		var nextCheck, untilTs, createdAt, updatedAt time.Time
		err := row.Scan(&id, &dealID, &channelID, &messageID, &postMessage, &st, &nextCheck, &untilTs, &createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}
		return &entity.DealPostMessage{
			ID:          id,
			DealID:      dealID,
			ChannelID:   channelID,
			MessageID:   messageID,
			PostMessage: postMessage,
			Status:      entity.DealPostMessageStatus(st),
			NextCheck:   nextCheck,
			UntilTs:     untilTs,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}, nil
	})
}

// CompleteDealPostMessagesAndSetDealsWaitingEscrowRelease updates given deal_post_message rows to completed and their deals to waiting_escrow_release in one tx.
func (r *repository) CompleteDealPostMessagesAndSetDealsWaitingEscrowRelease(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	txCtx, beginErr := r.db.BeginTx(ctx, pgx.TxOptions{})
	if beginErr != nil {
		return beginErr
	}
	var err error
	defer func() { _ = r.db.EndTx(txCtx, err, "CompleteDealPostMessagesAndSetDealsWaitingEscrowRelease") }()
	_, err = r.db.Exec(txCtx, `
		UPDATE market.deal_post_message SET status = 'completed', updated_at = NOW() WHERE id = ANY(@ids)`,
		pgx.NamedArgs{"ids": ids})
	if err != nil {
		return err
	}
	_, err = r.db.Exec(txCtx, `
		UPDATE market.deal SET status = 'waiting_escrow_release', updated_at = NOW()
		WHERE id IN (SELECT deal_id FROM market.deal_post_message WHERE id = ANY(@ids))`)
	if err != nil {
		return err
	}
	return nil
}

// FailDealPostMessagesAndSetDealsWaitingEscrowRefund updates given deal_post_message rows to failed and their deals to waiting_escrow_refund in one tx.
func (r *repository) FailDealPostMessagesAndSetDealsWaitingEscrowRefund(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	txCtx, beginErr := r.db.BeginTx(ctx, pgx.TxOptions{})
	if beginErr != nil {
		return beginErr
	}
	var err error
	defer func() { _ = r.db.EndTx(txCtx, err, "FailDealPostMessagesAndSetDealsWaitingEscrowRefund") }()
	_, err = r.db.Exec(txCtx, `
		UPDATE market.deal_post_message SET status = 'failed', updated_at = NOW() WHERE id = ANY(@ids)`,
		pgx.NamedArgs{"ids": ids})
	if err != nil {
		return err
	}
	_, err = r.db.Exec(txCtx, `
		UPDATE market.deal SET status = 'waiting_escrow_refund', updated_at = NOW()
		WHERE id IN (SELECT deal_id FROM market.deal_post_message WHERE id = ANY(@ids))`,
		pgx.NamedArgs{"ids": ids})
	if err != nil {
		return err
	}
	return nil
}
