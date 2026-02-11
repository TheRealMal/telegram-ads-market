package entity

import "time"

type DealPostMessageStatus string

const (
	DealPostMessageStatusExists    DealPostMessageStatus = "exists"
	DealPostMessageStatusDeleted   DealPostMessageStatus = "deleted"
	DealPostMessageStatusPassed    DealPostMessageStatus = "passed"
	DealPostMessageStatusCompleted DealPostMessageStatus = "completed"
	DealPostMessageStatusFailed    DealPostMessageStatus = "failed"
)

type DealPostMessage struct {
	ID          int64                `json:"id"`
	DealID      int64                `json:"deal_id"`
	ChannelID   int64                `json:"channel_id"`
	MessageID   int64                `json:"message_id"`
	PostMessage string               `json:"post_message"`
	Status      DealPostMessageStatus `json:"status"`
	NextCheck   time.Time            `json:"next_check"`
	UntilTs     time.Time            `json:"until_ts"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}
