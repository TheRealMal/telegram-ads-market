package entity

import (
	"encoding/json"
	"time"
)

type AdType string

const (
	AdTypePost        AdType = "post"
	AdTypeInstantPost AdType = "instant_post"
)

type DealStatus string

const (
	DealStatusDraft                  DealStatus = "draft"
	DealStatusApproved               DealStatus = "approved"
	DealStatusWaitingEscrowDeposit   DealStatus = "waiting_escrow_deposit"
	DealStatusEscrowDepositConfirmed DealStatus = "escrow_deposit_confirmed"
	DealStatusInProgress             DealStatus = "in_progress"
	DealStatusWaitingEscrowRelease   DealStatus = "waiting_escrow_release"
	DealStatusEscrowReleaseConfirmed DealStatus = "escrow_release_confirmed"
	DealStatusCompleted              DealStatus = "completed"
	DealStatusWaitingEscrowRefund    DealStatus = "waiting_escrow_refund"
	DealStatusEscrowRefundConfirmed  DealStatus = "escrow_refund_confirmed"
	DealStatusExpired                DealStatus = "expired"
	DealStatusRejected               DealStatus = "rejected"
)

// Deal represents a deal between lessor and lessee. In draft, both can edit type, duration, price, details;
// any edit clears both signatures. When both signatures are valid for current [type, duration, price, details],
// status becomes approved.
type Deal struct {
	ID                  int64           `json:"id"`
	ListingID           int64           `json:"listing_id"`
	LessorID            int64           `json:"lessor_id"`
	LesseeID            int64           `json:"lessee_id"`
	ChannelID           *int64          `json:"channel_id,omitempty"` // from listing; channel where ad is posted (validated at deal creation)
	Type                AdType          `json:"type"`
	Duration            int64           `json:"duration"`
	Price               int64           `json:"price"`         // in nanoton; API layer converts to/from TON
	EscrowAmount        int64           `json:"escrow_amount"` // price + transaction gas + commission
	Details             json.RawMessage `json:"details"`
	Message             string          `json:"message"` // context message from deal creator (NOT ad content)
	LessorSignature     *string         `json:"lessor_signature,omitempty"`
	LesseeSignature     *string         `json:"lessee_signature,omitempty"`
	Status              DealStatus      `json:"status"`
	EscrowAddress       *string         `json:"escrow_address,omitempty"`
	EscrowReleaseTime   *time.Time      `json:"escrow_release_time,omitempty"`
	LessorPayoutAddress *string         `json:"lessor_payout_address,omitempty"`
	LesseePayoutAddress *string         `json:"lessee_payout_address,omitempty"`
	CreatedAt           time.Time       `json:"created_at,omitempty"`
	UpdatedAt           time.Time       `json:"updated_at,omitempty"`
}

// ForumTopicEmojiForStatus maps deal statuses to custom emoji IDs for forum topic icons.
var ForumTopicEmojiForStatus = map[DealStatus]string{
	DealStatusWaitingEscrowDeposit:   "5310107765874632305", // 💱
	DealStatusEscrowDepositConfirmed: "5348227245599105972", // 💼
	DealStatusCompleted:              "5237699328843200968", // ✅
	DealStatusWaitingEscrowRelease:   "5309929258443874898", // 💸
	DealStatusWaitingEscrowRefund:    "5309929258443874898", // 💸
	DealStatusExpired:                "5386395194029515402", // 🏴‍☠️
	DealStatusRejected:               "5386395194029515402", // 🏴‍☠️
}

// CanBeEdited reports whether the deal can be edited (only draft deals are editable).
func (d *Deal) CanBeEdited() bool {
	return d.Status == DealStatusDraft
}

// IsInstantPost reports whether the deal is of instant_post ad type.
func (d *Deal) IsInstantPost() bool {
	return d.Type == AdTypeInstantPost
}

// CanBeApproved reports whether both parties have set their signatures on the deal.
func (d *Deal) CanBeApproved() bool {
	return d.LessorSignature != nil && *d.LessorSignature != "" &&
		d.LesseeSignature != nil && *d.LesseeSignature != ""
}
