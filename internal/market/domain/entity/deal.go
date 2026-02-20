package entity

import (
	"encoding/json"
	"time"
)

type DealStatus string

const (
	DealStatusDraft                    DealStatus = "draft"
	DealStatusApproved                 DealStatus = "approved"
	DealStatusWaitingEscrowDeposit     DealStatus = "waiting_escrow_deposit"
	DealStatusEscrowDepositConfirmed   DealStatus = "escrow_deposit_confirmed"
	DealStatusInProgress               DealStatus = "in_progress"
	DealStatusWaitingEscrowRelease     DealStatus = "waiting_escrow_release"
	DealStatusEscrowReleaseConfirmed   DealStatus = "escrow_release_confirmed"
	DealStatusCompleted                DealStatus = "completed"
	DealStatusWaitingEscrowRefund      DealStatus = "waiting_escrow_refund"
	DealStatusEscrowRefundConfirmed    DealStatus = "escrow_refund_confirmed"
	DealStatusExpired                  DealStatus = "expired"
	DealStatusRejected                 DealStatus = "rejected"
)

// Deal represents a deal between lessor and lessee. In draft, both can edit type, duration, price, details;
// any edit clears both signatures. When both signatures are valid for current [type, duration, price, details],
// status becomes approved.
type Deal struct {
	ID                   int64           `json:"id"`
	ListingID            int64           `json:"listing_id"`
	LessorID             int64           `json:"lessor_id"`
	LesseeID             int64           `json:"lessee_id"`
	ChannelID            *int64          `json:"channel_id,omitempty"` // from listing; channel where ad is posted (validated at deal creation)
	Type                 string          `json:"type"`
	Duration             int64           `json:"duration"`
	Price                float64         `json:"price"`
	EscrowAmount         int64           `json:"escrow_amount"` // price + transaction gas + commission
	Details              json.RawMessage `json:"details"`
	LessorSignature     *string         `json:"lessor_signature,omitempty"`
	LesseeSignature     *string         `json:"lessee_signature,omitempty"`
	Status               DealStatus      `json:"status"`
	EscrowAddress       *string         `json:"escrow_address,omitempty"`
	EscrowPrivateKey    *string         `json:"-"` // never expose
	EscrowReleaseTime   *time.Time      `json:"escrow_release_time,omitempty"`
	LessorPayoutAddress *string         `json:"lessor_payout_address,omitempty"`
	LesseePayoutAddress *string         `json:"lessee_payout_address,omitempty"`
	CreatedAt           time.Time       `json:"created_at,omitempty"`
	UpdatedAt           time.Time       `json:"updated_at,omitempty"`
}
