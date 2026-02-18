package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
)

type updatesService interface {
	HandleUpdate(ctx context.Context, raw []byte) error
}

type handler struct {
	updatesService updatesService
}

func NewHandler(updatesService updatesService) *handler {
	return &handler{
		updatesService: updatesService,
	}
}

// @Security
// @Tags		Telegram-Internal
// @Summary	Handle update
// @Accept		json
// @Param		request	body		[]byte	true	"request body"
// @Success	200		{object}	string
// @Router		/telegram/webhook [post]
func (h *handler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("ServiceError", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}
	defer r.Body.Close()

	err = h.updatesService.HandleUpdate(r.Context(), bodyBytes)
	if err != nil {
		slog.Error("ServiceError", "error", err, "request_body", string(bodyBytes))
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
}
