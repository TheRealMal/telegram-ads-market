package model

import (
	"encoding/json"
	"time"

	"ads-mrkt/internal/market/domain"
	"ads-mrkt/internal/market/domain/entity"
)

type DealResponse struct {
	ID                  int64             `json:"id"`
	ListingID           int64             `json:"listing_id"`
	LessorID            int64             `json:"lessor_id"`
	LesseeID            int64             `json:"lessee_id"`
	ChannelID           *int64            `json:"channel_id,omitempty"`
	Type                string            `json:"type"`
	Duration            int64             `json:"duration"`
	Price               float64           `json:"price"`
	EscrowAmount        int64             `json:"escrow_amount"`
	Details             json.RawMessage   `json:"details"`
	LessorSignature     *string           `json:"lessor_signature,omitempty"`
	LesseeSignature     *string           `json:"lessee_signature,omitempty"`
	Status              entity.DealStatus `json:"status"`
	EscrowAddress       *string           `json:"escrow_address,omitempty"`
	EscrowReleaseTime   *time.Time        `json:"escrow_release_time,omitempty"`
	LessorPayoutAddress *string           `json:"lessor_payout_address,omitempty"`
	LesseePayoutAddress *string           `json:"lessee_payout_address,omitempty"`
	CreatedAt           time.Time         `json:"created_at,omitempty"`
	UpdatedAt           time.Time         `json:"updated_at,omitempty"`
}

func DealToResponse(d *entity.Deal) *DealResponse {
	if d == nil {
		return nil
	}
	return &DealResponse{
		ID:                  d.ID,
		ListingID:           d.ListingID,
		LessorID:            d.LessorID,
		LesseeID:            d.LesseeID,
		ChannelID:           d.ChannelID,
		Type:                d.Type,
		Duration:            d.Duration,
		Price:               domain.NanotonToTON(d.Price),
		EscrowAmount:        d.EscrowAmount,
		Details:             d.Details,
		LessorSignature:     d.LessorSignature,
		LesseeSignature:     d.LesseeSignature,
		Status:              d.Status,
		EscrowAddress:       d.EscrowAddress,
		EscrowReleaseTime:   d.EscrowReleaseTime,
		LessorPayoutAddress: d.LessorPayoutAddress,
		LesseePayoutAddress: d.LesseePayoutAddress,
		CreatedAt:           d.CreatedAt,
		UpdatedAt:           d.UpdatedAt,
	}
}

func DealsToResponses(list []*entity.Deal) []*DealResponse {
	out := make([]*DealResponse, len(list))
	for i, d := range list {
		out[i] = DealToResponse(d)
	}
	return out
}

type CreateDealRequest struct {
	ListingID int64           `json:"listing_id"`
	ChannelID *int64          `json:"channel_id,omitempty"`
	Type      string          `json:"type"`
	Duration  int64           `json:"duration"`
	Price     float64         `json:"price"`
	Details   json.RawMessage `json:"details"`
}

type UpdateDealDraftRequest struct {
	Type     *string         `json:"type,omitempty"`
	Duration *int64          `json:"duration,omitempty"`
	Price    *float64        `json:"price,omitempty"`
	Details  json.RawMessage `json:"details,omitempty"`
}

type SetDealPayoutRequest struct {
	WalletAddress string `json:"wallet_address"`
}

type DealChatLinkResponse struct {
	ChatLink string `json:"chat_link"`
}
