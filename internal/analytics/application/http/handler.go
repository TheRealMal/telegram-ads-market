package http

import (
	"context"
	"net/http"
	"time"

	"ads-mrkt/internal/analytics/domain"
	apperrors "ads-mrkt/internal/errors"
	_ "ads-mrkt/internal/server/templates/response"
)

// AnalyticsService is the service layer interface used by the application handler.
type AnalyticsService interface {
	GetLatestSnapshot(ctx context.Context) (*domain.Snapshot, error)
	ListSnapshots(ctx context.Context, from, to time.Time) ([]*domain.Snapshot, error)
}

// SnapshotResponse is the API snapshot with commission in TON.
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

// LatestSnapshotResponse is the response for GET /api/v1/analytics/snapshot/latest.
type LatestSnapshotResponse struct {
	Snapshot *SnapshotResponse `json:"snapshot"`
}

// HistoryResponse is the response for GET /api/v1/analytics/snapshot/history.
// Each field is a slice of values; index i corresponds to timestamps[i] for all series.
type HistoryResponse struct {
	Period                 string               `json:"period"`     // week, month, year
	From                   string               `json:"from"`      // RFC3339
	To                     string               `json:"to"`        // RFC3339
	Timestamps             []int64              `json:"timestamps"` // Unix seconds; index i = time for all value series at i
	ListingsCount          []int64              `json:"listings_count"`
	DealsCount             []int64              `json:"deals_count"`
	UsersCount             []int64              `json:"users_count"`
	CommissionEarnedTON    []float64            `json:"commission_earned_ton"`
	AvgListingsPerUser     []float64            `json:"avg_listings_per_user"`
	DealsByStatus          map[string][]int64   `json:"deals_by_status"`            // status -> series of counts
	DealAmountsByStatusTON map[string][]float64 `json:"deal_amounts_by_status_ton"` // status -> series of sums in TON
}

const (
	periodWeek  = "week"
	periodMonth = "month"
	periodYear  = "year"
)

var periodToDays = map[string]int{
	periodWeek:  7,
	periodMonth: 30,
	periodYear:  365,
}

type handler struct {
	svc AnalyticsService
}

// NewHandler creates the analytics HTTP handler. It depends on the analytics service layer only.
func NewHandler(svc AnalyticsService) *handler {
	return &handler{svc: svc}
}

// GetLatestSnapshot returns the most recent analytics snapshot for the frontend (current state).
//
// @Tags		Analytics
// @Summary	Get latest analytics snapshot
// @Produce	json
// @Success	200	{object}	response.Template{data=LatestSnapshotResponse}	"Latest snapshot (snapshot field null when none)"
// @Failure	500	{object}	response.Template{data=string}				"Internal error"
// @Router		/analytics/snapshot/latest [get]
const nanotonPerTON = 1e9

func (h *handler) GetLatestSnapshot(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	ctx := r.Context()
	snap, err := h.svc.GetLatestSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	if snap == nil {
		return &LatestSnapshotResponse{Snapshot: nil}, nil
	}
	return &LatestSnapshotResponse{
		Snapshot: &SnapshotResponse{
			ID:                     snap.ID,
			RecordedAt:             snap.RecordedAt,
			ListingsCount:          snap.ListingsCount,
			DealsCount:             snap.DealsCount,
			DealsByStatus:          snap.DealsByStatus,
			DealAmountsByStatusTON: snap.DealAmountsByStatusTON,
			CommissionEarnedTON:    float64(snap.CommissionEarnedNanoton) / nanotonPerTON,
			UsersCount:             snap.UsersCount,
			AvgListingsPerUser:     snap.AvgListingsPerUser,
		},
	}, nil
}

// GetSnapshotHistory returns time-series for line charts: value slices plus timestamps (index i = time for all series at i).
//
// @Tags		Analytics
// @Summary	Get analytics snapshot history for charts
// @Produce	json
// @Param		period	query		string	false	"Time range: week (7d), month (30d), year (365d)"	Enums(week, month, year)
// @Success	200	{object}	response.Template{data=HistoryResponse}	"Timestamps (Unix) and metric series"
// @Failure	400	{object}	response.Template{data=string}				"Bad request (invalid period)"
// @Failure	500	{object}	response.Template{data=string}				"Internal error"
// @Router		/analytics/snapshot/history [get]
func (h *handler) GetSnapshotHistory(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	ctx := r.Context()
	period := r.URL.Query().Get("period")
	if period == "" {
		period = periodWeek
	}
	days, ok := periodToDays[period]
	if !ok {
		return nil, apperrors.ServiceError{Code: apperrors.ErrorCodeBadRequest, Message: "period must be week, month, or year"}
	}
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -days)
	to := now

	list, err := h.svc.ListSnapshots(ctx, from, to)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []*domain.Snapshot{}
	}

	resp := &HistoryResponse{
		Period:                 period,
		From:                   from.Format(time.RFC3339),
		To:                     to.Format(time.RFC3339),
		Timestamps:             make([]int64, 0, len(list)),
		ListingsCount:          make([]int64, 0, len(list)),
		DealsCount:             make([]int64, 0, len(list)),
		UsersCount:             make([]int64, 0, len(list)),
		CommissionEarnedTON:    make([]float64, 0, len(list)),
		AvgListingsPerUser:     make([]float64, 0, len(list)),
		DealsByStatus:          make(map[string][]int64),
		DealAmountsByStatusTON: make(map[string][]float64),
	}

	// Collect all status keys for map-based series
	statusKeys := make(map[string]struct{})
	amountStatusKeys := make(map[string]struct{})
	for _, snap := range list {
		for k := range snap.DealsByStatus {
			statusKeys[k] = struct{}{}
		}
		for k := range snap.DealAmountsByStatusTON {
			amountStatusKeys[k] = struct{}{}
		}
	}
	for k := range statusKeys {
		resp.DealsByStatus[k] = make([]int64, 0, len(list))
	}
	for k := range amountStatusKeys {
		resp.DealAmountsByStatusTON[k] = make([]float64, 0, len(list))
	}

	for _, snap := range list {
		t, _ := time.Parse(time.RFC3339, snap.RecordedAt)
		resp.Timestamps = append(resp.Timestamps, t.Unix())
		resp.ListingsCount = append(resp.ListingsCount, snap.ListingsCount)
		resp.DealsCount = append(resp.DealsCount, snap.DealsCount)
		resp.UsersCount = append(resp.UsersCount, snap.UsersCount)
		resp.CommissionEarnedTON = append(resp.CommissionEarnedTON, float64(snap.CommissionEarnedNanoton)/nanotonPerTON)
		resp.AvgListingsPerUser = append(resp.AvgListingsPerUser, snap.AvgListingsPerUser)
		for k := range statusKeys {
			v := int64(0)
			if snap.DealsByStatus != nil {
				v = snap.DealsByStatus[k]
			}
			resp.DealsByStatus[k] = append(resp.DealsByStatus[k], v)
		}
		for k := range amountStatusKeys {
			v := 0.0
			if snap.DealAmountsByStatusTON != nil {
				v = snap.DealAmountsByStatusTON[k]
			}
			resp.DealAmountsByStatusTON[k] = append(resp.DealAmountsByStatusTON[k], v)
		}
	}

	return resp, nil
}
