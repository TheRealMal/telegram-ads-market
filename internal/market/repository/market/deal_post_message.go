package repository

import (
	"context"
	"errors"
	"time"

	"ads-mrkt/internal/market/domain/entity"

	"github.com/jackc/pgx/v5"
)

type dealPostMessageRow struct {
	ID          int64     `db:"id"`
	DealID      int64     `db:"deal_id"`
	ChannelID   int64     `db:"channel_id"`
	MessageID   int64     `db:"message_id"`
	PostMessage string    `db:"post_message"`
	Status      string    `db:"status"`
	NextCheck   time.Time `db:"next_check"`
	UntilTs     time.Time `db:"until_ts"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func dealPostMessageRowToEntity(row dealPostMessageRow) *entity.DealPostMessage {
	return &entity.DealPostMessage{
		ID:          row.ID,
		DealID:      row.DealID,
		ChannelID:   row.ChannelID,
		MessageID:   row.MessageID,
		PostMessage: row.PostMessage,
		Status:      entity.DealPostMessageStatus(row.Status),
		NextCheck:   row.NextCheck,
		UntilTs:     row.UntilTs,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

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

// CreateDealPostMessageAndSetDealInProgress inserts the deal_post_message and sets the deal status to in_progress
// (when current status is escrow_deposit_confirmed) in a single transaction.
func (r *repository) CreateDealPostMessageAndSetDealInProgress(ctx context.Context, m *entity.DealPostMessage) error {
	txCtx, beginErr := r.db.BeginTx(ctx, pgx.TxOptions{})
	if beginErr != nil {
		return beginErr
	}
	var err error
	defer func() { _ = r.db.EndTx(txCtx, err, "CreateDealPostMessageAndSetDealInProgress") }()

	rows, err := r.db.Query(txCtx, `
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

	_, err = r.db.Exec(txCtx, `
		UPDATE market.deal SET status = @status, updated_at = NOW()
		WHERE id = @deal_id AND status = @status_escrow_deposit_confirmed`,
		pgx.NamedArgs{
			"status":                          string(entity.DealStatusInProgress),
			"deal_id":                         m.DealID,
			"status_escrow_deposit_confirmed": string(entity.DealStatusEscrowDepositConfirmed),
		})
	return err
}

func (r *repository) UpdateDealPostMessageStatus(ctx context.Context, id int64, status entity.DealPostMessageStatus) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal_post_message SET status = @status, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{
			"id":     id,
			"status": string(status),
		},
	)
	return err
}

func (r *repository) UpdateDealPostMessageStatusAndNextCheck(ctx context.Context, id int64, status entity.DealPostMessageStatus, nextCheck time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal_post_message SET status = @status, next_check = @next_check, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{
			"id":         id,
			"status":     string(status),
			"next_check": nextCheck,
		},
	)
	return err
}

func (r *repository) ListDealPostMessageExistsWithNextCheckBefore(ctx context.Context, before time.Time) ([]*entity.DealPostMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, deal_id, channel_id, message_id, post_message, status, next_check, until_ts, created_at, updated_at
		FROM market.deal_post_message
		WHERE status = 'exists' AND next_check <= @before
		ORDER BY id`,
		pgx.NamedArgs{
			"before": before,
		},
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[dealPostMessageRow])
	if err != nil {
		return nil, err
	}
	list := make([]*entity.DealPostMessage, 0, len(slice))
	for _, row := range slice {
		list = append(list, dealPostMessageRowToEntity(row))
	}
	return list, nil
}

func (r *repository) ListDealPostMessageByStatus(ctx context.Context, status entity.DealPostMessageStatus) ([]*entity.DealPostMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, deal_id, channel_id, message_id, post_message, status, next_check, until_ts, created_at, updated_at
		FROM market.deal_post_message
		WHERE status = @status
		ORDER BY id`,
		pgx.NamedArgs{
			"status": string(status),
		},
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[dealPostMessageRow])
	if err != nil {
		return nil, err
	}
	list := make([]*entity.DealPostMessage, 0, len(slice))
	for _, row := range slice {
		list = append(list, dealPostMessageRowToEntity(row))
	}
	return list, nil
}

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
		pgx.NamedArgs{
			"ids": ids,
		},
	)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(txCtx, `
		UPDATE market.deal SET status = 'waiting_escrow_release', updated_at = NOW()
		WHERE id IN (SELECT deal_id FROM market.deal_post_message WHERE id = ANY(@ids))`,
		pgx.NamedArgs{
			"ids": ids,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

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
		pgx.NamedArgs{
			"ids": ids,
		},
	)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(txCtx, `
		UPDATE market.deal SET status = 'waiting_escrow_refund', updated_at = NOW()
		WHERE id IN (SELECT deal_id FROM market.deal_post_message WHERE id = ANY(@ids))`,
		pgx.NamedArgs{
			"ids": ids,
		},
	)
	if err != nil {
		return err
	}
	return nil
}
