package model

import (
	"encoding/json"
	"time"

	"ads-mrkt/internal/market/domain/entity"
)

type DealRow struct {
	ID                  int64           `db:"id"`
	ListingID           int64           `db:"listing_id"`
	LessorID            int64           `db:"lessor_id"`
	LesseeID            int64           `db:"lessee_id"`
	ChannelID           *int64          `db:"channel_id"`
	Type                string          `db:"type"`
	Duration            int64           `db:"duration"`
	Price               int64           `db:"price"`
	EscrowAmount        int64           `db:"escrow_amount"`
	Details             json.RawMessage `db:"details"`
	LessorSignature     *string         `db:"lessor_signature"`
	LesseeSignature     *string         `db:"lessee_signature"`
	Status              string          `db:"status"`
	EscrowAddress       *string         `db:"escrow_address"`
	EscrowPrivateKey    *string         `db:"escrow_private_key"`
	EscrowReleaseTime   *time.Time      `db:"escrow_release_time"`
	LessorPayoutAddress *string         `db:"lessor_payout_address"`
	LesseePayoutAddress *string         `db:"lessee_payout_address"`
	CreatedAt           time.Time       `db:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at"`
}

type DealReturnRow struct {
	ID        int64     `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func DealRowToEntity(row DealRow) *entity.Deal {
	return &entity.Deal{
		ID:                  row.ID,
		ListingID:           row.ListingID,
		LessorID:            row.LessorID,
		LesseeID:            row.LesseeID,
		ChannelID:           row.ChannelID,
		Type:                row.Type,
		Duration:            row.Duration,
		Price:               row.Price,
		EscrowAmount:        row.EscrowAmount,
		Details:             row.Details,
		LessorSignature:     row.LessorSignature,
		LesseeSignature:     row.LesseeSignature,
		Status:              entity.DealStatus(row.Status),
		EscrowAddress:       row.EscrowAddress,
		EscrowPrivateKey:    row.EscrowPrivateKey,
		EscrowReleaseTime:   row.EscrowReleaseTime,
		LessorPayoutAddress: row.LessorPayoutAddress,
		LesseePayoutAddress: row.LesseePayoutAddress,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}
