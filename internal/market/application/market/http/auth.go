package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	apperrors "ads-mrkt/internal/errors"
	"ads-mrkt/pkg/auth"

	_ "ads-mrkt/internal/server/templates/response"
)

type AuthUserRequest struct {
	Referrer int64 `json:"referrer"` // optional, for future use
}

type SetWalletRequest struct {
	WalletAddress string `json:"wallet_address"` // TON address in raw format
}

// @Security
// @Tags		Market
// @Summary	Authenticate user
// @Accept		json
// @Produce	json
// @Param		request				body		AuthUserRequest					true	"request body"
// @Param		X-Telegram-InitData	header		string							true	"Telegram init data"
// @Success	200					{object}	response.Template{data=string}	"JWT token"
// @Failure	401					{object}	response.Template{data=string}	"Unauthorized"
// @Router		/market/auth [post]
func (h *Handler) AuthUser(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	initDataStr := r.Header.Get("X-Telegram-InitData")
	if initDataStr == "" {
		return nil, apperrors.ServiceError{
			Err:     fmt.Errorf("init data is required"),
			Message: "X-Telegram-InitData header is empty",
			Code:    apperrors.ErrorCodeInitDataRequired,
		}
	}

	var req AuthUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = AuthUserRequest{Referrer: 0}
	}

	user, err := h.userService.AuthUser(r.Context(), initDataStr, req.Referrer)
	if err != nil {
		return nil, apperrors.ServiceError{
			Err:     err,
			Message: "failed to authenticate user",
			Code:    apperrors.ErrorCodeUnauthorized,
		}
	}

	token, err := h.jwtManager.GenerateToken(user.ID)
	if err != nil {
		return nil, apperrors.ServiceError{
			Err:     err,
			Message: "failed to generate JWT token",
			Code:    apperrors.ErrorCodeUnauthorized,
		}
	}

	return token, nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Set current user's TON wallet (raw format) for deal payouts.
// @Accept		json
// @Produce	json
// @Param		request	body		SetWalletRequest				true	"wallet_address (raw)"
// @Success	200		{object}	response.Template{data=string}	"ok"
// @Failure	400		{object}	response.Template{data=string}	"Bad request"
// @Failure	401		{object}	response.Template{data=string}	"Unauthorized"
// @Router		/market/me/wallet [put]
func (h *Handler) SetWallet(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	var req SetWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid body", Code: apperrors.ErrorCodeBadRequest}
	}
	if req.WalletAddress == "" {
		return nil, apperrors.ServiceError{Err: nil, Message: "wallet_address is required", Code: apperrors.ErrorCodeBadRequest}
	}

	if err := h.userService.SetWallet(r.Context(), userID, req.WalletAddress); err != nil {
		return nil, toServiceError(err)
	}
	return map[string]string{"status": "ok"}, nil
}
