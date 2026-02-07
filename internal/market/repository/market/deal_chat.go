package repository

import (
	"context"
	"errors"
	"time"

	"ads-mrkt/internal/market/domain/entity"

	"github.com/jackc/pgx/v5"
)

type dealChatRow struct {
	DealID           int64     `db:"deal_id"`
	ReplyToChatID    int64     `db:"reply_to_chat_id"`
	ReplyToMessageID int64     `db:"reply_to_message_id"`
	RepliedMessage   *string   `db:"replied_message"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

func (r *repository) InsertDealChat(ctx context.Context, dc *entity.DealChat) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO market.deal_chat (deal_id, reply_to_chat_id, reply_to_message_id, replied_message)
		VALUES (@deal_id, @reply_to_chat_id, @reply_to_message_id, @replied_message)`,
		pgx.NamedArgs{
			"deal_id":             dc.DealID,
			"reply_to_chat_id":    dc.ReplyToChatID,
			"reply_to_message_id": dc.ReplyToMessageID,
			"replied_message":     dc.RepliedMessage,
		})
	return err
}

// GetDealChatByReply returns the deal_chat row for the given sent message (chat_id, message_id), if any.
func (r *repository) GetDealChatByReply(ctx context.Context, chatID, messageID int64) (*entity.DealChat, error) {
	rows, err := r.db.Query(ctx, `
		SELECT deal_id, reply_to_chat_id, reply_to_message_id, replied_message, created_at, updated_at
		FROM market.deal_chat
		WHERE reply_to_chat_id = @chat_id AND reply_to_message_id = @message_id`,
		pgx.NamedArgs{"chat_id": chatID, "message_id": messageID})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dealChatRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &entity.DealChat{
		DealID:           row.DealID,
		ReplyToChatID:    row.ReplyToChatID,
		ReplyToMessageID: row.ReplyToMessageID,
		RepliedMessage:   row.RepliedMessage,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func (r *repository) UpdateDealChatRepliedMessage(ctx context.Context, dealID, replyToChatID, replyToMessageID int64, repliedMessage string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal_chat
		SET replied_message = @replied_message, updated_at = NOW()
		WHERE deal_id = @deal_id AND reply_to_chat_id = @reply_to_chat_id AND reply_to_message_id = @reply_to_message_id`,
		pgx.NamedArgs{
			"deal_id":             dealID,
			"reply_to_chat_id":    replyToChatID,
			"reply_to_message_id": replyToMessageID,
			"replied_message":     repliedMessage,
		})
	return err
}

// ListDealChatsByDealIDWhereReplied returns deal_chat rows for the deal that have replied_message NOT NULL, in chronological order (created_at).
func (r *repository) ListDealChatsByDealIDWhereReplied(ctx context.Context, dealID int64) ([]*entity.DealChat, error) {
	rows, err := r.db.Query(ctx, `
		SELECT deal_id, reply_to_chat_id, reply_to_message_id, replied_message, created_at, updated_at
		FROM market.deal_chat
		WHERE deal_id = @deal_id AND replied_message IS NOT NULL
		ORDER BY created_at ASC`,
		pgx.NamedArgs{"deal_id": dealID})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[dealChatRow])
	if err != nil {
		return nil, err
	}
	list := make([]*entity.DealChat, 0, len(slice))
	for _, row := range slice {
		list = append(list, &entity.DealChat{
			DealID:           row.DealID,
			ReplyToChatID:    row.ReplyToChatID,
			ReplyToMessageID: row.ReplyToMessageID,
			RepliedMessage:   row.RepliedMessage,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		})
	}
	return list, nil
}

// HasActiveDealChatForUser returns true if there is a deal_chat row for this deal and user (reply_to_chat_id) with replied_message IS NULL.
func (r *repository) HasActiveDealChatForUser(ctx context.Context, dealID, userID int64) (bool, error) {
	rows, err := r.db.Query(ctx, `
		SELECT 1 AS one FROM market.deal_chat
		WHERE deal_id = @deal_id AND reply_to_chat_id = @user_id AND replied_message IS NULL
		LIMIT 1`,
		pgx.NamedArgs{"deal_id": dealID, "user_id": userID})
	if err != nil {
		return false, err
	}
	defer rows.Close()
	type oneRow struct {
		One int `db:"one"`
	}
	_, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[oneRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
