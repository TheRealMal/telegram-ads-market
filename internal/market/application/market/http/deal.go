package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	apperrors "ads-mrkt/internal/errors"
	"ads-mrkt/internal/market/domain"
	"ads-mrkt/internal/market/domain/entity"
	_ "ads-mrkt/internal/server/templates/response"
	"ads-mrkt/pkg/auth"
)

// CreateDealRequest is the body for POST /api/v1/market/deals. LessorID/LesseeID are derived from listing + token.
type CreateDealRequest struct {
	ListingID int64           `json:"listing_id"`
	Type      string          `json:"type"`
	Duration  int64           `json:"duration"`
	Price     int64           `json:"price"`
	Details   json.RawMessage `json:"details"`
}

// @Security	JWT
// @Tags		Market
// @Summary	Create deal
// @Accept		json
// @Produce	json
// @Param		request	body		CreateDealRequest					true	"deal body"
// @Success	200		{object}	response.Template{data=entity.Deal}	"Created deal"
// @Failure	400		{object}	response.Template{data=string}		"Bad request"
// @Failure	401		{object}	response.Template{data=string}		"Unauthorized"
// @Failure	403		{object}	response.Template{data=string}		"Forbidden"
// @Failure	404		{object}	response.Template{data=string}		"Not found"
// @Router		/market/deals [post]
func (h *Handler) CreateDeal(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	var req CreateDealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid body", Code: apperrors.ErrorCodeBadRequest}
	}

	listing, err := h.listingService.GetListing(r.Context(), req.ListingID)
	if err != nil {
		return nil, toServiceError(err)
	}
	if listing == nil {
		return nil, apperrors.ServiceError{Err: nil, Message: "listing not found", Code: apperrors.ErrorCodeNotFound}
	}
	if listing.UserID == userID {
		return nil, apperrors.ServiceError{Err: nil, Message: "cannot create deal on your own listing", Code: apperrors.ErrorCodeForbidden}
	}

	var lessorID, lesseeID int64
	switch listing.Type {
	case entity.ListingTypeLessor:
		lessorID = listing.UserID
		lesseeID = userID
	case entity.ListingTypeLessee:
		lessorID = userID
		lesseeID = listing.UserID
	default:
		return nil, apperrors.ServiceError{Err: nil, Message: "invalid listing type", Code: apperrors.ErrorCodeBadRequest}
	}

	details := req.Details
	if details == nil {
		details = json.RawMessage("{}")
	}
	canonDetails, err := domain.ValidateDealDetails(details)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeBadRequest}
	}
	d := &entity.Deal{
		ListingID: req.ListingID,
		LessorID:  lessorID,
		LesseeID:  lesseeID,
		Type:      req.Type,
		Duration:  req.Duration,
		Price:     req.Price,
		Details:   canonDetails,
	}
	if err := h.dealService.CreateDeal(r.Context(), d); err != nil {
		return nil, toServiceError(err)
	}
	return d, nil
}

// @Tags		Market
// @Summary	Get deal by ID
// @Produce	json
// @Param		id	path		int									true	"Deal ID"
// @Success	200	{object}	response.Template{data=entity.Deal}	"Deal"
// @Failure	400	{object}	response.Template{data=string}		"Bad request"
// @Failure	404	{object}	response.Template{data=string}		"Not found"
// @Router		/market/deals/{id} [get]
func (h *Handler) GetDeal(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid id", Code: apperrors.ErrorCodeBadRequest}
	}

	d, err := h.dealService.GetDeal(r.Context(), id)
	if err != nil {
		return nil, toServiceError(err)
	}
	if d == nil {
		return nil, apperrors.ServiceError{Err: nil, Message: "not found", Code: apperrors.ErrorCodeNotFound}
	}
	return d, nil
}

// @Tags		Market
// @Summary	List deals by listing ID
// @Produce	json
// @Param		listing_id	path		int										true	"Listing ID"
// @Success	200			{object}	response.Template{data=[]entity.Deal}	"List of deals"
// @Failure	400			{object}	response.Template{data=string}			"Bad request"
// @Router		/market/listings/{listing_id}/deals [get]
func (h *Handler) ListDealsByListingID(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	listingIDStr := r.PathValue("listing_id")
	if listingIDStr == "" {
		return nil, apperrors.ServiceError{Err: nil, Message: "listing_id required", Code: apperrors.ErrorCodeBadRequest}
	}
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid listing_id", Code: apperrors.ErrorCodeBadRequest}
	}

	list, err := h.dealService.GetDealsByListingID(r.Context(), listingID)
	if err != nil {
		return nil, toServiceError(err)
	}
	return list, nil
}

// @Security	JWT
// @Tags		Market
// @Summary	List deals for the current user (as lessor or lessee)
// @Produce	json
// @Success	200	{object}	response.Template{data=[]entity.Deal}	"List of user's deals"
// @Failure	401	{object}	response.Template{data=string}			"Unauthorized"
// @Router		/market/my-deals [get]
func (h *Handler) ListMyDeals(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}
	list, err := h.dealService.GetDealsByUserID(r.Context(), userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	return list, nil
}

// UpdateDealDraftRequest is the body for PATCH /api/v1/market/deals/:id (draft only)
type UpdateDealDraftRequest struct {
	Type     *string         `json:"type,omitempty"`
	Duration *int64          `json:"duration,omitempty"`
	Price    *int64          `json:"price,omitempty"`
	Details  json.RawMessage `json:"details,omitempty"`
}

// @Security	JWT
// @Tags		Market
// @Summary	Update deal draft (type, duration, price, details). Clears both signatures.
// @Accept		json
// @Produce	json
// @Param		id		path		int									true	"Deal ID"
// @Param		request	body		UpdateDealDraftRequest				true	"fields to update"
// @Success	200		{object}	response.Template{data=entity.Deal}	"Updated deal"
// @Failure	400		{object}	response.Template{data=string}		"Bad request"
// @Failure	401		{object}	response.Template{data=string}		"Unauthorized"
// @Failure	403		{object}	response.Template{data=string}		"Forbidden"
// @Failure	404		{object}	response.Template{data=string}		"Not found"
// @Router		/market/deals/{id} [patch]
func (h *Handler) UpdateDealDraft(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid id", Code: apperrors.ErrorCodeBadRequest}
	}

	existing, err := h.dealService.GetDeal(r.Context(), id)
	if err != nil || existing == nil {
		return nil, toServiceError(err)
	}

	var req UpdateDealDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid body", Code: apperrors.ErrorCodeBadRequest}
	}

	d := *existing
	d.ID = id
	if req.Type != nil {
		d.Type = *req.Type
	}
	if req.Duration != nil {
		d.Duration = *req.Duration
	}
	if req.Price != nil {
		d.Price = *req.Price
	}
	if req.Details != nil {
		canonDetails, err := domain.ValidateDealDetails(req.Details)
		if err != nil {
			return nil, apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeBadRequest}
		}
		d.Details = canonDetails
	}

	if err := h.dealService.UpdateDealDraft(r.Context(), userID, &d); err != nil {
		return nil, toServiceError(err)
	}
	updated, _ := h.dealService.GetDeal(r.Context(), id)
	return updated, nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Sign deal (lessor or lessee). When both have signed same terms, status becomes approved.
// @Produce	json
// @Param		id	path		int									true	"Deal ID"
// @Success	200	{object}	response.Template{data=entity.Deal}	"Deal (possibly approved)"
// @Failure	400	{object}	response.Template{data=string}		"Bad request"
// @Failure	401	{object}	response.Template{data=string}		"Unauthorized"
// @Failure	403	{object}	response.Template{data=string}		"Forbidden"
// @Failure	404	{object}	response.Template{data=string}		"Not found"
// @Router		/market/deals/{id}/sign [post]
func (h *Handler) SignDeal(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid id", Code: apperrors.ErrorCodeBadRequest}
	}

	if err := h.dealService.SignDeal(r.Context(), userID, id); err != nil {
		return nil, toServiceError(err)
	}
	updated, _ := h.dealService.GetDeal(r.Context(), id)
	return updated, nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Send deal chat invite message to the current user. Sends "Reply to this message to chat with the other side." and stores the message for reply tracking.
// @Produce	json
// @Param		id	path		int										true	"Deal ID"
// @Success	200	{object}	response.Template{data=entity.DealChat}	"Created deal chat row"
// @Failure	400	{object}	response.Template{data=string}			"Bad request"
// @Failure	401	{object}	response.Template{data=string}			"Unauthorized"
// @Failure	403	{object}	response.Template{data=string}			"Forbidden"
// @Failure	404	{object}	response.Template{data=string}			"Not found"
// @Router		/market/deals/{id}/send-chat-message [post]
func (h *Handler) SendDealChatMessage(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid id", Code: apperrors.ErrorCodeBadRequest}
	}

	dc, err := h.dealChatService.SendDealChatMessage(r.Context(), id, userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	return dc, nil
}

// @Security	JWT
// @Tags		Market
// @Summary	List deal messages (chat invite + replies) for the deal in chronological order. Caller must be lessor or lessee.
// @Produce	json
// @Param		id	path		int											true	"Deal ID"
// @Success	200	{object}	response.Template{data=[]entity.DealChat}	"List of deal chat messages"
// @Failure	400	{object}	response.Template{data=string}				"Bad request"
// @Failure	401	{object}	response.Template{data=string}				"Unauthorized"
// @Failure	403	{object}	response.Template{data=string}				"Forbidden"
// @Failure	404	{object}	response.Template{data=string}				"Not found"
// @Router		/market/deals/{id}/messages [get]
func (h *Handler) ListDealMessages(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid id", Code: apperrors.ErrorCodeBadRequest}
	}

	list, err := h.dealChatService.ListDealMessages(r.Context(), id, userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	return list, nil
}
