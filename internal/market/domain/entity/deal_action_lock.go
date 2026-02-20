package entity

import "time"

// DealActionType is the type of action protected by the lock.
type DealActionType string

const (
	DealActionTypeEscrowRelease DealActionType = "escrow_release"
	DealActionTypeEscrowRefund  DealActionType = "escrow_refund"
	DealActionTypePostMessage   DealActionType = "post_message"
)

// DealActionLockStatus is the status of a deal action lock.
type DealActionLockStatus string

const (
	DealActionLockStatusLocked    DealActionLockStatus = "locked"
	DealActionLockStatusCompleted DealActionLockStatus = "completed"
	DealActionLockStatusFailed    DealActionLockStatus = "failed"
)

// DealActionLock represents a short-lived lock for a deal action (escrow release/refund or post message).
// Used for concurrency safety and recovery: expire_at allows retry after service restart.
type DealActionLock struct {
	ID         string               `json:"id"`
	DealID     int64                `json:"deal_id"`
	ActionType DealActionType       `json:"action_type"`
	Status     DealActionLockStatus `json:"status"`
	ExpireAt   time.Time            `json:"expire_at"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}
