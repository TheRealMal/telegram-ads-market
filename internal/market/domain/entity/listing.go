package entity

import (
	"encoding/json"
	"fmt"
	"strconv"
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

// ListingPriceEntry represents a single price option in a listing.
// Duration is in the format "<n>hr" (e.g. "24hr"). Price is in nanoton.
type ListingPriceEntry struct {
	AdType   AdType
	Duration string
	Price    int64 // nanoton
}

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
	PreparedPost     json.RawMessage `json:"prepared_post,omitempty"` // JSON object, e.g. {"message": "ad text"}; extensible
	CreatedAt        time.Time       `json:"created_at,omitempty"`
	UpdatedAt        time.Time       `json:"updated_at,omitempty"`
}

// ParseCategories unmarshals the JSON-encoded category list into a string slice.
func (l *Listing) ParseCategories() ([]string, error) {
	if len(l.Categories) == 0 {
		return nil, nil
	}
	var cats []string
	if err := json.Unmarshal(l.Categories, &cats); err != nil {
		return nil, fmt.Errorf("parse listing categories: %w", err)
	}
	return cats, nil
}

// ParsePrices unmarshals the JSON-encoded price list into typed entries.
// Each entry is a [ad_type, duration, price] triple where price is in nanoton.
func (l *Listing) ParsePrices() ([]ListingPriceEntry, error) {
	if len(l.Prices) == 0 {
		return nil, nil
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(l.Prices, &raw); err != nil {
		return nil, fmt.Errorf("parse listing prices: %w", err)
	}
	entries := make([]ListingPriceEntry, 0, len(raw))
	for i, slot := range raw {
		var triple []interface{}
		if err := json.Unmarshal(slot, &triple); err != nil || len(triple) != 3 {
			return nil, fmt.Errorf("parse listing prices[%d]: expected 3-element array", i)
		}
		adType, ok := triple[0].(string)
		if !ok {
			return nil, fmt.Errorf("parse listing prices[%d]: ad_type must be a string", i)
		}
		duration, ok := triple[1].(string)
		if !ok {
			return nil, fmt.Errorf("parse listing prices[%d]: duration must be a string", i)
		}
		price, err := parseListingPriceNumber(triple[2])
		if err != nil {
			return nil, fmt.Errorf("parse listing prices[%d]: %w", i, err)
		}
		entries = append(entries, ListingPriceEntry{AdType: AdType(adType), Duration: duration, Price: price})
	}
	return entries, nil
}

func parseListingPriceNumber(v interface{}) (int64, error) {
	switch x := v.(type) {
	case float64:
		return int64(x), nil
	case json.Number:
		return strconv.ParseInt(x.String(), 10, 64)
	default:
		return 0, fmt.Errorf("price must be a number")
	}
}

// IsActive reports whether the listing status is active.
func (l *Listing) IsActive() bool {
	return l.Status == ListingStatusActive
}

// HasAdType reports whether the listing's prices include at least one entry for the given ad type.
func (l *Listing) HasAdType(adType AdType) bool {
	entries, err := l.ParsePrices()
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.AdType == adType {
			return true
		}
	}
	return false
}

// GetPrice returns the nanoton price for the given ad type and duration string (e.g. "24hr").
// Returns (0, false) if no matching entry is found.
func (l *Listing) GetPrice(adType AdType, duration string) (int64, bool) {
	entries, err := l.ParsePrices()
	if err != nil {
		return 0, false
	}
	for _, e := range entries {
		if e.AdType == adType && e.Duration == duration {
			return e.Price, true
		}
	}
	return 0, false
}
