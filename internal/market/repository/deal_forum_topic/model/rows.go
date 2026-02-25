package model

import (
	"time"

	"ads-mrkt/internal/market/domain/entity"
)

type DealForumTopicRow struct {
	DealID                int64     `db:"deal_id"`
	LessorChatID          int64     `db:"lessor_chat_id"`
	LesseeChatID          int64     `db:"lessee_chat_id"`
	LessorMessageThreadID int64     `db:"lessor_message_thread_id"`
	LesseeMessageThreadID int64     `db:"lessee_message_thread_id"`
	CreatedAt             time.Time `db:"created_at"`
	UpdatedAt             time.Time `db:"updated_at"`
}

func DealForumTopicRowToEntity(row DealForumTopicRow) *entity.DealForumTopic {
	return &entity.DealForumTopic{
		DealID:                row.DealID,
		LessorChatID:          row.LessorChatID,
		LesseeChatID:          row.LesseeChatID,
		LessorMessageThreadID: row.LessorMessageThreadID,
		LesseeMessageThreadID: row.LesseeMessageThreadID,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
}
