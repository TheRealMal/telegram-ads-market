package entity

import (
	"encoding/json"
	"time"
)

type DealStatus string

const (
	DealStatusDraft                   DealStatus = "draft"
	DealStatusApproved                DealStatus = "approved"
	DealStatusWaitingEscrowDeposit    DealStatus = "waiting_escrow_deposit"
	DealStatusEscrowDepositConfirmed  DealStatus = "escrow_deposit_confirmed"
	DealStatusInProgress              DealStatus = "in_progress"
	DealStatusWaitingEscrowRelease    DealStatus = "waiting_escrow_release"
	DealStatusEscrowReleaseConfirmed  DealStatus = "escrow_release_confirmed"
	DealStatusCompleted               DealStatus = "completed"
	DealStatusExpired                  DealStatus = "expired"
	DealStatusRejected                DealStatus = "rejected"
)

// Deal represents a deal between lessor and lessee. In draft, both can edit type, duration, price, details;
// any edit clears both signatures. When both signatures are valid for current [type, duration, price, details],
// status becomes approved.
type Deal struct {
	ID               int64          `json:"id"`
	ListingID        int64          `json:"listing_id"`
	LessorID         int64          `json:"lessor_id"`
	LesseeID         int64          `json:"lessee_id"`
	Type             string         `json:"type"`
	Duration         int64          `json:"duration"`
	Price            int64          `json:"price"`
	Details          json.RawMessage `json:"details"`
	LessorSignature  *string        `json:"lessor_signature,omitempty"`
	LesseeSignature  *string        `json:"lessee_signature,omitempty"`
	Status           DealStatus     `json:"status"`
	EscrowAddress    *string        `json:"escrow_address,omitempty"`
	EscrowPrivateKey *string        `json:"-"` // never expose
	CreatedAt        time.Time      `json:"created_at,omitempty"`
	UpdatedAt        time.Time      `json:"updated_at,omitempty"`
}
