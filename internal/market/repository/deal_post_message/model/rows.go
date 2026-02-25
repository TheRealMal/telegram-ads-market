package model

import (
	"time"

	"ads-mrkt/internal/market/domain/entity"
)

type DealPostMessageRow struct {
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

type DealPostMessageReturnRow struct {
	ID        int64     `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func DealPostMessageRowToEntity(row DealPostMessageRow) *entity.DealPostMessage {
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
