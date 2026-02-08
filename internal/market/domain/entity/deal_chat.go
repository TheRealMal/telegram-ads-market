package entity

import "time"

// DealChat represents a deal chat invitation message sent to a user.
// reply_to_chat_id/reply_to_message_id are the sent message; replied_message is set when the user replies.
type DealChat struct {
	DealID           int64     `json:"deal_id"`
	ReplyToChatID    int64     `json:"reply_to_chat_id"`
	ReplyToMessageID int64     `json:"-"`
	RepliedMessage   *string   `json:"replied_message,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
