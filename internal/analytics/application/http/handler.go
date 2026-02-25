package http

import (
	"context"
	"net/http"
	"time"

	"ads-mrkt/internal/analytics/application/http/model"
	"ads-mrkt/internal/analytics/domain"
	apperrors "ads-mrkt/internal/errors"
	_ "ads-mrkt/internal/server/templates/response"
)

type AnalyticsService interface {
	GetLatestSnapshot(ctx context.Context) (*domain.Snapshot, error)
	ListSnapshots(ctx context.Context, from, to time.Time) ([]*domain.Snapshot, error)
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

func NewHandler(svc AnalyticsService) *handler {
	return &handler{svc: svc}
}

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
		return &model.LatestSnapshotResponse{Snapshot: nil}, nil
	}
	return &model.LatestSnapshotResponse{
		Snapshot: &model.SnapshotResponse{
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

	resp := &model.HistoryResponse{
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
