package model

import (
	"encoding/json"
	"time"

	"ads-mrkt/internal/analytics/domain"
)

type CountRow struct {
	Count int64 `db:"count"`
}

type StatusCountRow struct {
	Status string `db:"status"`
	Count  int64  `db:"count"`
}

type StatusSumRow struct {
	Status string `db:"status"`
	Sum    int64  `db:"sum"`
}

type CommissionRow struct {
	Commission int64 `db:"commission"`
}

type ExistsRow struct {
	Exists bool `db:"exists"`
}

type SnapshotRow struct {
	ID                      int64           `db:"id"`
	RecordedAt              time.Time       `db:"recorded_at"`
	ListingsCount           int64           `db:"listings_count"`
	DealsCount              int64           `db:"deals_count"`
	DealsByStatus           json.RawMessage `db:"deals_by_status"`
	DealAmountsByStatusTon  json.RawMessage `db:"deal_amounts_by_status_ton"`
	CommissionEarnedNanoton int64           `db:"commission_earned_nanoton"`
	UsersCount              int64           `db:"users_count"`
	AvgListingsPerUser      float64         `db:"avg_listings_per_user"`
}

func SnapshotRowToDomain(row SnapshotRow) *domain.Snapshot {
	s := &domain.Snapshot{
		ID:                      row.ID,
		RecordedAt:              row.RecordedAt.Format(time.RFC3339),
		ListingsCount:           row.ListingsCount,
		DealsCount:              row.DealsCount,
		CommissionEarnedNanoton: row.CommissionEarnedNanoton,
		UsersCount:              row.UsersCount,
		AvgListingsPerUser:      row.AvgListingsPerUser,
	}
	s.DealsByStatus = make(map[string]int64)
	s.DealAmountsByStatusTON = make(map[string]float64)
	_ = json.Unmarshal(row.DealsByStatus, &s.DealsByStatus)
	_ = json.Unmarshal(row.DealAmountsByStatusTon, &s.DealAmountsByStatusTON)
	return s
}
