package http

import (
	"context"
	"encoding/json"
	"net/http"

	apperrors "ads-mrkt/internal/errors"
	"ads-mrkt/internal/market/application/market/http/model"
	"ads-mrkt/internal/market/domain"
	"ads-mrkt/internal/market/domain/entity"
	_ "ads-mrkt/internal/server/templates/response"
)

func resolveLessorLessee(listing *entity.Listing, userID int64) (lessorID, lesseeID int64, err error) {
	switch listing.Type {
	case entity.ListingTypeLessor:
		return listing.UserID, userID, nil
	case entity.ListingTypeLessee:
		return userID, listing.UserID, nil
	default:
		return 0, 0, apperrors.ServiceError{Err: nil, Message: "invalid listing type", Code: apperrors.ErrorCodeBadRequest}
	}
}

func resolveDealChannelID(listing *entity.Listing, req *model.CreateDealRequest) (*int64, error) {
	switch listing.Type {
	case entity.ListingTypeLessor:
		return listing.ChannelID, nil
	case entity.ListingTypeLessee:
		if req.ChannelID == nil {
			return nil, apperrors.ServiceError{Err: nil, Message: "channel_id is required when applying to a lessee listing", Code: apperrors.ErrorCodeBadRequest}
		}
		return req.ChannelID, nil
	default:
		return nil, nil
	}
}

func buildDealFromCreateRequest(req *model.CreateDealRequest, lessorID, lesseeID int64, dealChannelID *int64, canonDetails json.RawMessage) *entity.Deal {
	return &entity.Deal{
		ListingID: req.ListingID,
		LessorID:  lessorID,
		LesseeID:  lesseeID,
		ChannelID: dealChannelID,
		Type:      req.Type,
		Duration:  req.Duration,
		Price:     domain.TONToNanoton(req.Price),
		Details:   canonDetails,
	}
}

func applyUpdateDealDraftRequest(existing *entity.Deal, id int64, req *model.UpdateDealDraftRequest) (*entity.Deal, error) {
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
	return &d, nil
}

func (h *handler) validateDealPriceMatchesListing(ctx context.Context, existing, d *entity.Deal, req *model.UpdateDealDraftRequest) error {
	if req.Type == nil && req.Duration == nil && req.Price == nil {
		return nil
	}
	listing, err := h.listingService.GetListing(ctx, existing.ListingID)
	if err != nil {
		return toServiceError(err)
	}
	if listing == nil {
		return apperrors.ServiceError{Err: nil, Message: "listing not found", Code: apperrors.ErrorCodeNotFound}
	}
	if !domain.DealPriceMatchesListing(listing.Prices, d.Type, d.Duration, d.Price) {
		return apperrors.ServiceError{Err: nil, Message: "type, duration and price must match one of the listing's price options", Code: apperrors.ErrorCodeBadRequest}
	}
	return nil
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
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}

	var req model.CreateDealRequest
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

	lessorID, lesseeID, err := resolveLessorLessee(listing, userID)
	if err != nil {
		return nil, err
	}
	dealChannelID, err := resolveDealChannelID(listing, &req)
	if err != nil {
		return nil, err
	}

	details := req.Details
	if details == nil {
		details = json.RawMessage("{}")
	}
	canonDetails, err := domain.ValidateDealDetails(details)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeBadRequest}
	}

	d := buildDealFromCreateRequest(&req, lessorID, lesseeID, dealChannelID, canonDetails)
	if err := h.dealService.CreateDeal(r.Context(), d, listing.UserID); err != nil {
		return nil, toServiceError(err)
	}
	return model.DealToResponse(d), nil
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
// @Param		listing_id	path		int										true	"Listing ID"
// @Success	200			{object}	response.Template{data=[]entity.Deal}	"List of deals"
// @Failure	400			{object}	response.Template{data=string}			"Bad request"
// @Failure	401			{object}	response.Template{data=string}			"Unauthorized"
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
// @Success	200	{object}	response.Template{data=[]entity.Deal}	"List of user's deals"
// @Failure	401	{object}	response.Template{data=string}			"Unauthorized"
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
// @Param		id		path		int									true	"Deal ID"
// @Param		request	body		UpdateDealDraftRequest				true	"fields to update"
// @Success	200		{object}	response.Template{data=entity.Deal}	"Updated deal"
// @Failure	400		{object}	response.Template{data=string}		"Bad request"
// @Failure	401		{object}	response.Template{data=string}		"Unauthorized"
// @Failure	403		{object}	response.Template{data=string}		"Forbidden"
// @Failure	404		{object}	response.Template{data=string}		"Not found"
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

	existing, err := h.dealService.GetDeal(r.Context(), id)
	if err != nil || existing == nil {
		return nil, toServiceError(err)
	}

	var req model.UpdateDealDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid body", Code: apperrors.ErrorCodeBadRequest}
	}

	d, err := applyUpdateDealDraftRequest(existing, id, &req)
	if err != nil {
		return nil, err
	}
	if err := h.validateDealPriceMatchesListing(r.Context(), existing, d, &req); err != nil {
		return nil, err
	}

	if err := h.dealService.UpdateDealDraft(r.Context(), userID, d); err != nil {
		return nil, toServiceError(err)
	}
	updated, _ := h.dealService.GetDeal(r.Context(), id)
	return model.DealToResponse(updated), nil
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
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}

	if err := h.dealService.SignDeal(r.Context(), userID, id); err != nil {
		return nil, toServiceError(err)
	}
	updated, err := h.dealService.GetDeal(r.Context(), id)
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
// @Param		id	path		int									true	"Deal ID"
// @Param		request	body		SetDealPayoutRequest				true	"wallet_address (raw)"
// @Success	200	{object}	response.Template{data=entity.Deal}	"Updated deal"
// @Failure	400	{object}	response.Template{data=string}		"Bad request"
// @Failure	401	{object}	response.Template{data=string}		"Unauthorized"
// @Failure	404	{object}	response.Template{data=string}		"Not found"
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

	if err := h.dealService.SetDealPayoutAddress(r.Context(), userID, id, req.WalletAddress); err != nil {
		return nil, toServiceError(err)
	}
	updated, _ := h.dealService.GetDeal(r.Context(), id)
	return model.DealToResponse(updated), nil
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
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}

	if err := h.dealService.RejectDeal(r.Context(), userID, id); err != nil {
		return nil, toServiceError(err)
	}
	updated, err := h.dealService.GetDeal(r.Context(), id)
	if err != nil {
		return nil, toServiceError(err)
	}
	return model.DealToResponse(updated), nil
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
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}

	chatLink, err := h.dealChatService.GetOrCreateDealForumChat(r.Context(), id, userID)
	if err != nil {
		return nil, toServiceError(err)
	}
	return &model.DealChatLinkResponse{ChatLink: chatLink}, nil
}
