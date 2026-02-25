package model

type SnapshotResponse struct {
	ID                     int64              `json:"id"`
	RecordedAt             string             `json:"recorded_at"`
	ListingsCount          int64              `json:"listings_count"`
	DealsCount             int64              `json:"deals_count"`
	DealsByStatus          map[string]int64   `json:"deals_by_status"`
	DealAmountsByStatusTON map[string]float64 `json:"deal_amounts_by_status_ton"`
	CommissionEarnedTON    float64            `json:"commission_earned_ton"`
	UsersCount             int64              `json:"users_count"`
	AvgListingsPerUser     float64            `json:"avg_listings_per_user"`
}

type LatestSnapshotResponse struct {
	Snapshot *SnapshotResponse `json:"snapshot"`
}

type HistoryResponse struct {
	Period                 string               `json:"period"`
	From                   string               `json:"from"`
	To                     string               `json:"to"`
	Timestamps             []int64              `json:"timestamps"`
	ListingsCount          []int64              `json:"listings_count"`
	DealsCount             []int64              `json:"deals_count"`
	UsersCount             []int64              `json:"users_count"`
	CommissionEarnedTON    []float64            `json:"commission_earned_ton"`
	AvgListingsPerUser     []float64            `json:"avg_listings_per_user"`
	DealsByStatus          map[string][]int64   `json:"deals_by_status"`
	DealAmountsByStatusTON map[string][]float64 `json:"deal_amounts_by_status_ton"`
}
