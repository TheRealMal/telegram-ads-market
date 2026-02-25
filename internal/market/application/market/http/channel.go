package http

import (
	"errors"
	"net/http"

	apperrors "ads-mrkt/internal/errors"
	marketerrors "ads-mrkt/internal/market/domain/errors"
	_ "ads-mrkt/internal/market/domain/entity"
	_ "ads-mrkt/internal/server/templates/response"
)

// @Security	JWT
// @Tags		Market
// @Summary	List channels where current user is admin
// @Produce	json
// @Success	200	{object}	response.Template{data=[]entity.Channel}	"List of my channels"
// @Failure	401	{object}	response.Template{data=string}				"Unauthorized"
// @Router		/market/my-channels [get]
func (h *handler) ListMyChannels(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	list, err := h.channelService.ListMyChannels(r.Context(), userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	return list, nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Refresh (get) channel data by id; user must be admin of the channel
// @Produce	json
// @Param		id	path		int										true	"Channel ID"
// @Success	200	{object}	response.Template{data=entity.Channel}	"Channel"
// @Failure	400	{object}	response.Template{data=string}			"Bad request"
// @Failure	401	{object}	response.Template{data=string}			"Unauthorized"
// @Failure	403	{object}	response.Template{data=string}			"Forbidden"
// @Failure	404	{object}	response.Template{data=string}			"Not found"
// @Router		/market/channels/{id}/refresh [get]
func (h *handler) RefreshChannel(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}
	ch, err := h.channelService.RequestStatsRefresh(r.Context(), id, userID)
	if err != nil {
		var tooSoon *marketerrors.ErrStatsRefreshTooSoon
		if errors.As(err, &tooSoon) {
			return nil, apperrors.ServiceError{
				Err:     err,
				Message: tooSoon.Error(),
				Code:    apperrors.ErrorCodeTooManyRequests,
				Data:   map[string]string{"next_available_at": tooSoon.NextAvailableAt.Format("15:04")},
			}
		}
		return nil, toServiceError(err)
	}
	if ch == nil {
		return nil, apperrors.ServiceError{Err: nil, Message: "not found", Code: apperrors.ErrorCodeNotFound}
	}
	return ch, nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Get channel statistics. Allowed for channel admins or users who have a listing with this channel.
// @Produce	json
// @Param		id	path		int	true	"Channel ID"
// @Success	200	{object}	response.Template{data=object}	"Channel stats (Telegram broadcast stats shape)"
// @Failure	400	{object}	response.Template{data=string}	"Bad request"
// @Failure	401	{object}	response.Template{data=string}	"Unauthorized"
// @Failure	403	{object}	response.Template{data=string}	"Forbidden"
// @Failure	404	{object}	response.Template{data=string}	"Not found"
// @Router		/market/channels/{id}/stats [get]
func (h *handler) GetChannelStats(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}
	stats, err := h.channelService.GetChannelStats(r.Context(), id, userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	if stats == nil {
		return nil, apperrors.ServiceError{Err: nil, Message: "not found", Code: apperrors.ErrorCodeNotFound}
	}
	return stats, nil
}
