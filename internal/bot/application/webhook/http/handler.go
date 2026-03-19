package handler

import (
	"context"
	"fmt"
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
func (h *handler) HandleUpdate(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read webhook body: %w", err)
	}

	if err := h.updatesService.HandleUpdate(r.Context(), bodyBytes); err != nil {
		slog.Error("handle update", "error", err, "request_body", string(bodyBytes))
	}

	return nil, nil
}
