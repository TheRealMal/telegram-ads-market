package http

import (
	"net/http"
)

func (h *handler) GetWaitlist(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	count, err := h.channelService.CountMyChannels(r.Context(), userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	return map[string]int64{"channel_count": count}, nil
}
