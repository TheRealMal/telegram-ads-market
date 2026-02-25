package model

import (
	"encoding/json"

	"ads-mrkt/internal/market/domain"
	"ads-mrkt/internal/market/domain/entity"
)

type CreateListingRequest struct {
	Status      string          `json:"status"`
	ChannelID   *int64          `json:"channel_id,omitempty"`
	Type        string          `json:"type"`
	Prices      json.RawMessage `json:"prices"`
	Categories  []string        `json:"categories,omitempty"`
	Description string          `json:"description,omitempty"`
}

type UpdateListingRequest struct {
	Status      *string         `json:"status,omitempty"`
	Type        *string         `json:"type,omitempty"`
	Prices      json.RawMessage `json:"prices,omitempty"`
	Categories  *[]string       `json:"categories,omitempty"`
	Description *string         `json:"description,omitempty"`
}

func ListingWithPricesInTON(l *entity.Listing) *entity.Listing {
	if l == nil {
		return nil
	}
	converted, _ := domain.ConvertListingPricesNanotonToTON(l.Prices)
	out := *l
	out.Prices = converted
	return &out
}

func ListingsWithPricesInTON(list []*entity.Listing) []*entity.Listing {
	out := make([]*entity.Listing, len(list))
	for i, l := range list {
		out[i] = ListingWithPricesInTON(l)
	}
	return out
}

func CategoriesToRaw(categories []string) json.RawMessage {
	if len(categories) == 0 {
		return json.RawMessage("[]")
	}
	b, _ := json.Marshal(categories)
	return b
}
