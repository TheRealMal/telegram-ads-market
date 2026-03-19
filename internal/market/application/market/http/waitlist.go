package http

import (
	"net/http"

	"ads-mrkt/internal/market/application/market/http/model"
	_ "ads-mrkt/internal/server/templates/response"
)

// @Security	JWT
// @Tags		Market
// @Summary	Get waitlist status for current user
// @Produce	json
// @Success	200	{object}	response.Template{data=model.WaitlistResponse}	"Waitlist status with channel count"
// @Failure	401	{object}	response.Template{data=string}					"Unauthorized"
// @Router		/market/waitlist [get]
func (h *handler) GetWaitlist(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}

	list, err := h.channelService.ListMyChannels(r.Context(), userID)
	if err != nil {
		return nil, toServiceError(err)
	}

	referrals, err := h.userService.GetReferrals(r.Context(), userID)
	if err != nil {
		return nil, toServiceError(err)
	}

	return &model.WaitlistResponse{
		ChannelsConnected: list,
		Referrals:         referrals,
	}, nil
}
