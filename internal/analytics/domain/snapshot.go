package domain

import "encoding/json"

// Snapshot is one hourly analytics snapshot.
type Snapshot struct {
	ID                      int64              `json:"id"`
	RecordedAt              string             `json:"recorded_at"` // RFC3339
	ListingsCount           int64              `json:"listings_count"`
	DealsCount              int64              `json:"deals_count"`
	DealsByStatus           map[string]int64   `json:"deals_by_status"`            // status -> count
	DealAmountsByStatusTON  map[string]float64 `json:"deal_amounts_by_status_ton"` // status -> sum in TON
	CommissionEarnedNanoton int64              `json:"commission_earned_nanoton"`
	UsersCount              int64              `json:"users_count"`
	AvgListingsPerUser      float64            `json:"avg_listings_per_user"`
}

// DealsByStatusJSON returns deals_by_status as jsonb-ready value.
func (s *Snapshot) DealsByStatusJSON() (json.RawMessage, error) {
	if s.DealsByStatus == nil {
		return json.Marshal(map[string]int64{})
	}
	return json.Marshal(s.DealsByStatus)
}

// DealAmountsByStatusTONJSON returns deal_amounts_by_status_ton as jsonb-ready value.
func (s *Snapshot) DealAmountsByStatusTONJSON() (json.RawMessage, error) {
	if s.DealAmountsByStatusTON == nil {
		return json.Marshal(map[string]float64{})
	}
	return json.Marshal(s.DealAmountsByStatusTON)
}
