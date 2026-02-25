package domain

import "encoding/json"

type Snapshot struct {
	ID                      int64              `json:"id"`
	RecordedAt              string             `json:"recorded_at"`
	ListingsCount           int64              `json:"listings_count"`
	DealsCount              int64              `json:"deals_count"`
	DealsByStatus           map[string]int64   `json:"deals_by_status"`
	DealAmountsByStatusTON  map[string]float64 `json:"deal_amounts_by_status_ton"`
	CommissionEarnedNanoton int64              `json:"commission_earned_nanoton"`
	UsersCount              int64              `json:"users_count"`
	AvgListingsPerUser      float64            `json:"avg_listings_per_user"`
}

func (s *Snapshot) DealsByStatusJSON() (json.RawMessage, error) {
	if s.DealsByStatus == nil {
		return json.Marshal(map[string]int64{})
	}
	return json.Marshal(s.DealsByStatus)
}

func (s *Snapshot) DealAmountsByStatusTONJSON() (json.RawMessage, error) {
	if s.DealAmountsByStatusTON == nil {
		return json.Marshal(map[string]float64{})
	}
	return json.Marshal(s.DealAmountsByStatusTON)
}
