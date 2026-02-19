package entity

import "time"

// DealForumTopic stores one topic per side: lessor_chat_id/lessee_chat_id are Telegram user IDs (each user's chat with the bot); message_thread_id is the topic in that chat. Messages mirrored via copyMessage.
type DealForumTopic struct {
	DealID                int64     `json:"deal_id"`
	LessorChatID          int64     `json:"lessor_chat_id"`
	LesseeChatID          int64     `json:"lessee_chat_id"`
	LessorMessageThreadID int64     `json:"lessor_message_thread_id"`
	LesseeMessageThreadID int64     `json:"lessee_message_thread_id"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}
