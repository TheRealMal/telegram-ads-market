package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	apperrors "ads-mrkt/internal/errors"
	"ads-mrkt/internal/market/domain"
	"ads-mrkt/internal/market/domain/entity"
	_ "ads-mrkt/internal/server/templates/response"
	"ads-mrkt/pkg/auth"
)

// dealResponse is the API shape for a deal; price is in TON (float) for clients.
type dealResponse struct {
	ID                  int64             `json:"id"`
	ListingID           int64             `json:"listing_id"`
	LessorID            int64             `json:"lessor_id"`
	LesseeID            int64             `json:"lessee_id"`
	ChannelID           *int64            `json:"channel_id,omitempty"`
	Type                string            `json:"type"`
	Duration            int64             `json:"duration"`
	Price               float64           `json:"price"`
	EscrowAmount        int64             `json:"escrow_amount"`
	Details             json.RawMessage   `json:"details"`
	LessorSignature     *string           `json:"lessor_signature,omitempty"`
	LesseeSignature     *string           `json:"lessee_signature,omitempty"`
	Status              entity.DealStatus `json:"status"`
	EscrowAddress       *string           `json:"escrow_address,omitempty"`
	EscrowReleaseTime   *time.Time        `json:"escrow_release_time,omitempty"`
	LessorPayoutAddress *string           `json:"lessor_payout_address,omitempty"`
	LesseePayoutAddress *string           `json:"lessee_payout_address,omitempty"`
	CreatedAt           time.Time         `json:"created_at,omitempty"`
	UpdatedAt           time.Time         `json:"updated_at,omitempty"`
}

func dealToResponse(d *entity.Deal) *dealResponse {
	if d == nil {
		return nil
	}
	return &dealResponse{
		ID:                  d.ID,
		ListingID:           d.ListingID,
		LessorID:            d.LessorID,
		LesseeID:            d.LesseeID,
		ChannelID:           d.ChannelID,
		Type:                d.Type,
		Duration:            d.Duration,
		Price:               domain.NanotonToTON(d.Price),
		EscrowAmount:        d.EscrowAmount,
		Details:             d.Details,
		LessorSignature:     d.LessorSignature,
		LesseeSignature:     d.LesseeSignature,
		Status:              d.Status,
		EscrowAddress:       d.EscrowAddress,
		EscrowReleaseTime:   d.EscrowReleaseTime,
		LessorPayoutAddress: d.LessorPayoutAddress,
		LesseePayoutAddress: d.LesseePayoutAddress,
		CreatedAt:           d.CreatedAt,
		UpdatedAt:           d.UpdatedAt,
	}
}

func dealsToResponses(list []*entity.Deal) []*dealResponse {
	out := make([]*dealResponse, len(list))
	for i, d := range list {
		out[i] = dealToResponse(d)
	}
	return out
}

// CreateDealRequest is the body for POST /api/v1/market/deals. LessorID/LesseeID are derived from listing + token.
// For lessee-type listings, channel_id is required (channel where ad will be posted; must be owned by the deal creator).
type CreateDealRequest struct {
	ListingID int64           `json:"listing_id"`
	ChannelID *int64          `json:"channel_id,omitempty"` // Required when listing type is lessee; ignored when lessor (taken from listing).
	Type      string          `json:"type"`
	Duration  int64           `json:"duration"`
	Price     float64         `json:"price"`
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
func (h *handler) CreateDeal(w http.ResponseWriter, r *http.Request) (interface{}, error) {
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
	if !domain.DealPriceMatchesListing(listing.Prices, req.Type, req.Duration, domain.TONToNanoton(req.Price)) {
		return nil, apperrors.ServiceError{Err: nil, Message: "type, duration and price must match one of the listing's price options", Code: apperrors.ErrorCodeBadRequest}
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

	// Deal channel: lessor listing → from listing (listing owner = lessor = channel owner); lessee listing → from request (deal creator = lessor = channel owner).
	var dealChannelID *int64
	switch listing.Type {
	case entity.ListingTypeLessor:
		if listing.ChannelID != nil {
			dealChannelID = listing.ChannelID
		}
	case entity.ListingTypeLessee:
		if req.ChannelID == nil {
			return nil, apperrors.ServiceError{Err: nil, Message: "channel_id is required when applying to a lessee listing", Code: apperrors.ErrorCodeBadRequest}
		}
		dealChannelID = req.ChannelID
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
		ChannelID: dealChannelID,
		Type:      req.Type,
		Duration:  req.Duration,
		Price:     domain.TONToNanoton(req.Price),
		Details:   canonDetails,
	}
	if err := h.dealService.CreateDeal(r.Context(), d, listing.UserID); err != nil {
		return nil, toServiceError(err)
	}
	return dealToResponse(d), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Get deal by ID (only if caller is lessor or lessee)
// @Produce	json
// @Param		id	path		int									true	"Deal ID"
// @Success	200	{object}	response.Template{data=entity.Deal}	"Deal"
// @Failure	400	{object}	response.Template{data=string}		"Bad request"
// @Failure	401	{object}	response.Template{data=string}		"Unauthorized"
// @Failure	404	{object}	response.Template{data=string}		"Not found"
// @Router		/market/deals/{id} [get]
func (h *handler) GetDeal(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid id", Code: apperrors.ErrorCodeBadRequest}
	}

	d, err := h.dealService.GetDealForUser(r.Context(), id, userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	if d == nil {
		return nil, apperrors.ServiceError{Err: nil, Message: "not found", Code: apperrors.ErrorCodeNotFound}
	}
	return dealToResponse(d), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	List deals by listing ID (only deals where caller is lessor or lessee)
// @Produce	json
// @Param		listing_id	path		int										true	"Listing ID"
// @Success	200			{object}	response.Template{data=[]entity.Deal}	"List of deals"
// @Failure	400			{object}	response.Template{data=string}			"Bad request"
// @Failure	401			{object}	response.Template{data=string}			"Unauthorized"
// @Router		/market/listings/{listing_id}/deals [get]
func (h *handler) ListDealsByListingID(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	listingIDStr := r.PathValue("listing_id")
	if listingIDStr == "" {
		return nil, apperrors.ServiceError{Err: nil, Message: "listing_id required", Code: apperrors.ErrorCodeBadRequest}
	}
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid listing_id", Code: apperrors.ErrorCodeBadRequest}
	}

	list, err := h.dealService.GetDealsByListingIDForUser(r.Context(), listingID, userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	return dealsToResponses(list), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	List deals for the current user (as lessor or lessee)
// @Produce	json
// @Success	200	{object}	response.Template{data=[]entity.Deal}	"List of user's deals"
// @Failure	401	{object}	response.Template{data=string}			"Unauthorized"
// @Router		/market/my-deals [get]
func (h *handler) ListMyDeals(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}
	list, err := h.dealService.GetDealsByUserID(r.Context(), userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	return dealsToResponses(list), nil
}

// UpdateDealDraftRequest is the body for PATCH /api/v1/market/deals/:id (draft only)
type UpdateDealDraftRequest struct {
	Type     *string         `json:"type,omitempty"`
	Duration *int64          `json:"duration,omitempty"`
	Price    *float64        `json:"price,omitempty"`
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
func (h *handler) UpdateDealDraft(w http.ResponseWriter, r *http.Request) (interface{}, error) {
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
		d.Price = domain.TONToNanoton(*req.Price)
	}
	if req.Details != nil {
		canonDetails, err := domain.ValidateDealDetails(req.Details)
		if err != nil {
			return nil, apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeBadRequest}
		}
		d.Details = canonDetails
	}

	if req.Type != nil || req.Duration != nil || req.Price != nil {
		listing, listErr := h.listingService.GetListing(r.Context(), existing.ListingID)
		if listErr != nil {
			return nil, toServiceError(listErr)
		}
		if listing == nil {
			return nil, apperrors.ServiceError{Err: nil, Message: "listing not found", Code: apperrors.ErrorCodeNotFound}
		}
		if !domain.DealPriceMatchesListing(listing.Prices, d.Type, d.Duration, d.Price) {
			return nil, apperrors.ServiceError{Err: nil, Message: "type, duration and price must match one of the listing's price options", Code: apperrors.ErrorCodeBadRequest}
		}
	}

	if err := h.dealService.UpdateDealDraft(r.Context(), userID, &d); err != nil {
		return nil, toServiceError(err)
	}
	updated, _ := h.dealService.GetDeal(r.Context(), id)
	return dealToResponse(updated), nil
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
func (h *handler) SignDeal(w http.ResponseWriter, r *http.Request) (interface{}, error) {
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
	updated, err := h.dealService.GetDeal(r.Context(), id)
	if err != nil {
		return nil, toServiceError(err)
	}
	return dealToResponse(updated), nil
}

// SetDealPayoutRequest is the body for PUT /api/v1/market/deals/{id}/payout-address.
type SetDealPayoutRequest struct {
	WalletAddress string `json:"wallet_address"` // TON address in raw format
}

// @Security	JWT
// @Tags		Market
// @Summary	Set your payout address on the deal (lessor or lessee). Required before signing. Draft only.
// @Accept		json
// @Produce	json
// @Param		id	path		int									true	"Deal ID"
// @Param		request	body		SetDealPayoutRequest				true	"wallet_address (raw)"
// @Success	200	{object}	response.Template{data=entity.Deal}	"Updated deal"
// @Failure	400	{object}	response.Template{data=string}		"Bad request"
// @Failure	401	{object}	response.Template{data=string}		"Unauthorized"
// @Failure	404	{object}	response.Template{data=string}		"Not found"
// @Router		/market/deals/{id}/payout-address [put]
func (h *handler) SetDealPayoutAddress(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid id", Code: apperrors.ErrorCodeBadRequest}
	}

	var req SetDealPayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid body", Code: apperrors.ErrorCodeBadRequest}
	}
	if req.WalletAddress == "" {
		return nil, apperrors.ServiceError{Err: nil, Message: "wallet_address is required", Code: apperrors.ErrorCodeBadRequest}
	}

	if err := h.dealService.SetDealPayoutAddress(r.Context(), userID, id, req.WalletAddress); err != nil {
		return nil, toServiceError(err)
	}
	updated, _ := h.dealService.GetDeal(r.Context(), id)
	return dealToResponse(updated), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Reject deal. Only allowed when deal status is draft; caller must be lessor or lessee.
// @Produce	json
// @Param		id	path		int									true	"Deal ID"
// @Success	200	{object}	response.Template{data=entity.Deal}	"Updated deal"
// @Failure	400	{object}	response.Template{data=string}		"Bad request"
// @Failure	401	{object}	response.Template{data=string}		"Unauthorized"
// @Failure	403	{object}	response.Template{data=string}		"Forbidden (deal not draft or not a side)"
// @Failure	404	{object}	response.Template{data=string}		"Not found"
// @Router		/market/deals/{id}/reject [post]
func (h *handler) RejectDeal(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid id", Code: apperrors.ErrorCodeBadRequest}
	}

	if err := h.dealService.RejectDeal(r.Context(), userID, id); err != nil {
		return nil, toServiceError(err)
	}
	updated, err := h.dealService.GetDeal(r.Context(), id)
	if err != nil {
		return nil, toServiceError(err)
	}
	return dealToResponse(updated), nil
}

// DealChatLinkResponse is the response for get-or-create deal forum chat.
type DealChatLinkResponse struct {
	ChatLink string `json:"chat_link"`
}

// @Security	JWT
// @Tags		Market
// @Summary	Get or create deal forum chat and return link to open the topic. Caller must be lessor or lessee.
// @Produce	json
// @Param		id	path		int	true	"Deal ID"
// @Success	200	{object}	response.Template{data=DealChatLinkResponse}	"Chat link to open in Telegram"
// @Failure	400	{object}	response.Template{data=string}	"Bad request"
// @Failure	401	{object}	response.Template{data=string}	"Unauthorized"
// @Failure	403	{object}	response.Template{data=string}	"Forbidden"
// @Failure	404	{object}	response.Template{data=string}	"Not found"
// @Router		/market/deals/{id}/chat-link [post]
func (h *handler) GetOrCreateDealChatLink(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid id", Code: apperrors.ErrorCodeBadRequest}
	}

	chatLink, err := h.dealChatService.GetOrCreateDealForumChat(r.Context(), id, userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	return &DealChatLinkResponse{ChatLink: chatLink}, nil
}
