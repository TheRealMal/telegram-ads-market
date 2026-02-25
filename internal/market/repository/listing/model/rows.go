package model

import (
	"encoding/json"
	"time"

	"ads-mrkt/internal/market/domain/entity"
)

type ListingRow struct {
	ID          int64           `db:"id"`
	Status      string          `db:"status"`
	UserID      int64           `db:"user_id"`
	ChannelID   *int64          `db:"channel_id"`
	Type        string          `db:"type"`
	Prices      json.RawMessage `db:"prices"`
	Categories  json.RawMessage `db:"categories"`
	Description *string         `db:"description"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
}

type ListingWithChannelRow struct {
	ListingRow
	ChannelTitle     *string `db:"channel_title"`
	ChannelUsername  *string `db:"channel_username"`
	ChannelPhoto     *string `db:"channel_photo"`
	ChannelFollowers *int64  `db:"channel_followers"`
}

type ListingReturnRow struct {
	ID        int64     `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type ListingExistsRow struct {
	One int `db:"one"`
}

func stringFromPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ListingRowToEntity(row ListingRow) *entity.Listing {
	l := &entity.Listing{
		ID:          row.ID,
		Status:      entity.ListingStatus(row.Status),
		UserID:      row.UserID,
		ChannelID:   row.ChannelID,
		Type:        entity.ListingType(row.Type),
		Prices:      row.Prices,
		Description: stringFromPtr(row.Description),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if len(row.Categories) > 0 {
		l.Categories = row.Categories
	}
	return l
}

func ListingWithChannelRowToEntity(row ListingWithChannelRow) *entity.Listing {
	l := ListingRowToEntity(row.ListingRow)
	l.ChannelTitle = row.ChannelTitle
	l.ChannelUsername = row.ChannelUsername
	l.ChannelPhoto = row.ChannelPhoto
	l.ChannelFollowers = row.ChannelFollowers
	return l
}
