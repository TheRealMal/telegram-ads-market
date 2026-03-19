package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	apperrors "ads-mrkt/internal/errors"
	"ads-mrkt/internal/market/application/market/http/model"
	dealservice "ads-mrkt/internal/market/service/deal"
	_ "ads-mrkt/internal/server/templates/response"
)

// @Security	JWT
// @Tags		Market
// @Summary	Create deal
// @Accept		json
// @Produce	json
// @Param		request	body		model.CreateDealRequest						true	"deal body"
// @Success	200		{object}	response.Template{data=model.DealResponse}	"Created deal"
// @Failure	400		{object}	response.Template{data=string}				"Bad request"
// @Failure	401		{object}	response.Template{data=string}				"Unauthorized"
// @Failure	403		{object}	response.Template{data=string}				"Forbidden"
// @Failure	404		{object}	response.Template{data=string}				"Not found"
// @Router		/market/deals [post]
func (h *handler) CreateDeal(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}

	var req model.CreateDealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid body", Code: apperrors.ErrorCodeBadRequest}
	}

	d, listingType, err := h.dealService.CreateDealFromRequest(r.Context(), userID, dealservice.CreateDealInput{
		ListingID: req.ListingID,
		ChannelID: req.ChannelID,
		Type:      req.Type,
		Duration:  req.Duration,
		PriceTON:  req.Price,
		Message:   req.Message,
		Details:   req.Details,
	})
	if err != nil {
		return nil, toServiceError(err)
	}

	// Create forum topics eagerly (best-effort, don't fail the deal creation)
	if err := h.dealChatService.CreateDealForumTopics(r.Context(), d, listingType); err != nil {
		slog.Error("create deal forum topics", "deal_id", d.ID, "error", err)
	}

	return model.DealToResponse(d), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Get deal by ID (only if caller is lessor or lessee)
// @Produce	json
// @Param		id	path		int											true	"Deal ID"
// @Success	200	{object}	response.Template{data=model.DealResponse}	"Deal"
// @Failure	400	{object}	response.Template{data=string}				"Bad request"
// @Failure	401	{object}	response.Template{data=string}				"Unauthorized"
// @Failure	404	{object}	response.Template{data=string}				"Not found"
// @Router		/market/deals/{id} [get]
func (h *handler) GetDeal(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}

	d, err := h.dealService.GetDealForUser(r.Context(), id, userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	if d == nil {
		return nil, apperrors.ServiceError{Err: nil, Message: "not found", Code: apperrors.ErrorCodeNotFound}
	}
	return model.DealToResponse(d), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	List deals by listing ID (only deals where caller is lessor or lessee)
// @Produce	json
// @Param		listing_id	path		int												true	"Listing ID"
// @Success	200			{object}	response.Template{data=[]model.DealResponse}	"List of deals"
// @Failure	400			{object}	response.Template{data=string}					"Bad request"
// @Failure	401			{object}	response.Template{data=string}					"Unauthorized"
// @Router		/market/listings/{listing_id}/deals [get]
func (h *handler) ListDealsByListingID(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	listingID, err := parsePathID(r, "listing_id")
	if err != nil {
		return nil, err
	}

	list, err := h.dealService.GetDealsByListingIDForUser(r.Context(), listingID, userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	return model.DealsToResponses(list), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	List deals for the current user (as lessor or lessee)
// @Produce	json
// @Success	200	{object}	response.Template{data=[]model.DealResponse}	"List of user's deals"
// @Failure	401	{object}	response.Template{data=string}					"Unauthorized"
// @Router		/market/my-deals [get]
func (h *handler) ListMyDeals(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	list, err := h.dealService.GetDealsByUserID(r.Context(), userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	return model.DealsToResponses(list), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Update deal draft (type, duration, price, details). Clears both signatures.
// @Accept		json
// @Produce	json
// @Param		id		path		int											true	"Deal ID"
// @Param		request	body		model.UpdateDealDraftRequest				true	"fields to update"
// @Success	200		{object}	response.Template{data=model.DealResponse}	"Updated deal"
// @Failure	400		{object}	response.Template{data=string}				"Bad request"
// @Failure	401		{object}	response.Template{data=string}				"Unauthorized"
// @Failure	403		{object}	response.Template{data=string}				"Forbidden"
// @Failure	404		{object}	response.Template{data=string}				"Not found"
// @Router		/market/deals/{id} [patch]
func (h *handler) UpdateDealDraft(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}

	var req model.UpdateDealDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid body", Code: apperrors.ErrorCodeBadRequest}
	}

	updated, err := h.dealService.UpdateDealDraft(r.Context(), userID, id, dealservice.UpdateDealDraftInput{
		Type:     req.Type,
		Duration: req.Duration,
		PriceTON: req.Price,
		Details:  req.Details,
	})
	if err != nil {
		return nil, toServiceError(err)
	}
	return model.DealToResponse(updated), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Sign deal (lessor or lessee). When both have signed same terms, status becomes approved.
// @Produce	json
// @Param		id	path		int											true	"Deal ID"
// @Success	200	{object}	response.Template{data=model.DealResponse}	"Deal (possibly approved)"
// @Failure	400	{object}	response.Template{data=string}				"Bad request"
// @Failure	401	{object}	response.Template{data=string}				"Unauthorized"
// @Failure	403	{object}	response.Template{data=string}				"Forbidden"
// @Failure	404	{object}	response.Template{data=string}				"Not found"
// @Router		/market/deals/{id}/sign [post]
func (h *handler) SignDeal(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}

	updated, err := h.dealService.SignDeal(r.Context(), userID, id)
	if err != nil {
		return nil, toServiceError(err)
	}
	return model.DealToResponse(updated), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Set your payout address on the deal (lessor or lessee). Required before signing. Draft only.
// @Accept		json
// @Produce	json
// @Param		id		path		int											true	"Deal ID"
// @Param		request	body		model.SetDealPayoutRequest					true	"wallet_address (raw)"
// @Success	200		{object}	response.Template{data=model.DealResponse}	"Updated deal"
// @Failure	400		{object}	response.Template{data=string}				"Bad request"
// @Failure	401		{object}	response.Template{data=string}				"Unauthorized"
// @Failure	404		{object}	response.Template{data=string}				"Not found"
// @Router		/market/deals/{id}/payout-address [put]
func (h *handler) SetDealPayoutAddress(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}

	var req model.SetDealPayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid body", Code: apperrors.ErrorCodeBadRequest}
	}
	if req.WalletAddress == "" {
		return nil, apperrors.ServiceError{Err: nil, Message: "wallet_address is required", Code: apperrors.ErrorCodeBadRequest}
	}

	updated, err := h.dealService.SetDealPayoutAddress(r.Context(), userID, id, req.WalletAddress)
	if err != nil {
		return nil, toServiceError(err)
	}
	return model.DealToResponse(updated), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Reject deal. Only allowed when deal status is draft; caller must be lessor or lessee.
// @Produce	json
// @Param		id	path		int											true	"Deal ID"
// @Success	200	{object}	response.Template{data=model.DealResponse}	"Updated deal"
// @Failure	400	{object}	response.Template{data=string}				"Bad request"
// @Failure	401	{object}	response.Template{data=string}				"Unauthorized"
// @Failure	403	{object}	response.Template{data=string}				"Forbidden (deal not draft or not a side)"
// @Failure	404	{object}	response.Template{data=string}				"Not found"
// @Router		/market/deals/{id}/reject [post]
func (h *handler) RejectDeal(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}

	updated, err := h.dealService.RejectDeal(r.Context(), userID, id)
	if err != nil {
		return nil, toServiceError(err)
	}
	return model.DealToResponse(updated), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Get deal forum chat link. Caller must be lessor or lessee.
// @Produce	json
// @Param		id	path		int													true	"Deal ID"
// @Success	200	{object}	response.Template{data=model.DealChatLinkResponse}	"Chat link to open in Telegram"
// @Failure	400	{object}	response.Template{data=string}						"Bad request"
// @Failure	401	{object}	response.Template{data=string}						"Unauthorized"
// @Failure	403	{object}	response.Template{data=string}						"Forbidden"
// @Failure	404	{object}	response.Template{data=string}						"Not found"
// @Router		/market/deals/{id}/chat-link [post]
func (h *handler) GetDealChatLink(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}

	chatLink, err := h.dealChatService.GetDealChatLink(r.Context(), id, userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	return &model.DealChatLinkResponse{ChatLink: chatLink}, nil
}
