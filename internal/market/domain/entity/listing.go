package entity

import (
	"encoding/json"
	"time"
)

type ListingType string

const (
	ListingTypeLessor ListingType = "lessor"
	ListingTypeLessee ListingType = "lessee"
)

type ListingStatus string

const (
	ListingStatusActive   ListingStatus = "active"
	ListingStatusInactive ListingStatus = "inactive"
)

type Listing struct {
	ID               int64           `json:"id"`
	Status           ListingStatus   `json:"status"`
	UserID           int64           `json:"user_id"`
	ChannelID        *int64          `json:"channel_id,omitempty"`
	ChannelTitle     *string         `json:"channel_title,omitempty"`
	ChannelUsername  *string         `json:"channel_username,omitempty"`
	ChannelPhoto     *string         `json:"channel_photo,omitempty"`
	ChannelFollowers *int64          `json:"channel_followers,omitempty"`
	Type             ListingType     `json:"type"`
	Prices           json.RawMessage `json:"prices"`
	Categories       json.RawMessage `json:"categories,omitempty"` // JSON array of strings from predefined set
	Description      string          `json:"description,omitempty"`
	CreatedAt        time.Time       `json:"created_at,omitempty"`
	UpdatedAt        time.Time       `json:"updated_at,omitempty"`
}
