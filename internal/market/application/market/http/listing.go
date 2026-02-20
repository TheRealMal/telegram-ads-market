package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	apperrors "ads-mrkt/internal/errors"
	"ads-mrkt/internal/market/domain"
	"ads-mrkt/internal/market/domain/entity"
	_ "ads-mrkt/internal/server/templates/response"
	"ads-mrkt/pkg/auth"
)

func splitComma(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func categoriesToRaw(categories []string) json.RawMessage {
	if len(categories) == 0 {
		return json.RawMessage("[]")
	}
	b, _ := json.Marshal(categories)
	return b
}

// CreateListingRequest is the body for POST /api/v1/market/listings.
// Prices must be JSON array of [duration, price] pairs, e.g. [["24hr", 100], ["48hr", 200]].
// Categories must be from the predefined set (see domain.ListingCategories).
type CreateListingRequest struct {
	Status      string          `json:"status"` // active | inactive
	ChannelID   *int64          `json:"channel_id,omitempty"`
	Type        string          `json:"type"`   // lessor | lessee
	Prices      json.RawMessage `json:"prices"` // [["<number_of_hours>hr", <price>], ...]
	Categories  []string        `json:"categories,omitempty"`
	Description string          `json:"description,omitempty"`
}

// @Security	JWT
// @Tags		Market
// @Summary	Create listing
// @Accept		json
// @Produce	json
// @Param		request	body		CreateListingRequest					true	"listing body"
// @Success	200		{object}	response.Template{data=entity.Listing}	"Created listing"
// @Failure	400		{object}	response.Template{data=string}			"Bad request"
// @Failure	401		{object}	response.Template{data=string}			"Unauthorized"
// @Failure	403		{object}	response.Template{data=string}			"Forbidden"
// @Router		/market/listings [post]
func (h *handler) CreateListing(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	var req CreateListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid body", Code: apperrors.ErrorCodeBadRequest}
	}
	if err := domain.ValidateListingPrices(req.Prices); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeBadRequest}
	}
	if err := domain.ValidateListingCategories(req.Categories); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeBadRequest}
	}
	pricesNanoton, err := domain.ConvertListingPricesTONToNanoton(req.Prices)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid prices", Code: apperrors.ErrorCodeBadRequest}
	}

	l := &entity.Listing{
		Status:      entity.ListingStatus(req.Status),
		ChannelID:   req.ChannelID,
		Type:        entity.ListingType(req.Type),
		Prices:      pricesNanoton,
		Categories:  categoriesToRaw(req.Categories),
		Description: req.Description,
	}
	if l.Status == "" {
		l.Status = entity.ListingStatusInactive
	}
	if err := h.listingService.CreateListing(r.Context(), userID, l); err != nil {
		return nil, toServiceError(err)
	}
	return listingWithPricesInTON(l), nil
}

// @Tags		Market
// @Summary	Get listing by ID
// @Produce	json
// @Param		id	path		int										true	"Listing ID"
// @Success	200	{object}	response.Template{data=entity.Listing}	"Listing"
// @Failure	400	{object}	response.Template{data=string}			"Bad request"
// @Failure	404	{object}	response.Template{data=string}			"Not found"
// @Router		/market/listings/{id} [get]
func (h *handler) GetListing(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid id", Code: apperrors.ErrorCodeBadRequest}
	}

	l, err := h.listingService.GetListing(r.Context(), id)
	if err != nil {
		return nil, toServiceError(err)
	}
	if l == nil {
		return nil, apperrors.ServiceError{Err: nil, Message: "not found", Code: apperrors.ErrorCodeNotFound}
	}
	return listingWithPricesInTON(l), nil
}

// listingWithPricesInTON returns a copy of the listing with Prices JSON converted from nanoton to TON for API.
func listingWithPricesInTON(l *entity.Listing) *entity.Listing {
	if l == nil {
		return nil
	}
	converted, _ := domain.ConvertListingPricesNanotonToTON(l.Prices)
	out := *l
	out.Prices = converted
	return &out
}

// @Tags		Market
// @Summary	List all listings with optional type, categories, and min_followers filter (public, no auth)
// @Produce	json
// @Param		type	query		string										false	"Filter by type: lessor | lessee"
// @Param		categories	query		string									false	"Comma-separated categories (e.g. Tech,Crypto)"
// @Param		min_followers	query		int									false	"Min channel followers (only lessor listings with stats)"
// @Success	200		{object}	response.Template{data=[]entity.Listing}	"List of listings"
// @Router		/market/listings [get]
func (h *handler) ListListings(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var typ *entity.ListingType
	if t := r.URL.Query().Get("type"); t != "" {
		tp := entity.ListingType(t)
		typ = &tp
	}
	var categories []string
	if c := r.URL.Query().Get("categories"); c != "" {
		for _, s := range splitComma(c) {
			if s != "" {
				categories = append(categories, s)
			}
		}
	}
	var minFollowers *int64
	if m := r.URL.Query().Get("min_followers"); m != "" {
		if n, err := strconv.ParseInt(m, 10, 64); err == nil && n >= 0 {
			minFollowers = &n
		}
	}
	list, err := h.listingService.ListListingsAll(r.Context(), typ, categories, minFollowers)
	if err != nil {
		return nil, toServiceError(err)
	}
	return listingsWithPricesInTON(list), nil
}

func listingsWithPricesInTON(list []*entity.Listing) []*entity.Listing {
	out := make([]*entity.Listing, len(list))
	for i, l := range list {
		out[i] = listingWithPricesInTON(l)
	}
	return out
}

// @Security	JWT
// @Tags		Market
// @Summary	List current user's listings with optional type filter
// @Produce	json
// @Param		type	query		string										false	"Filter by type: lessor | lessee"
// @Success	200		{object}	response.Template{data=[]entity.Listing}	"List of my listings"
// @Failure	401		{object}	response.Template{data=string}				"Unauthorized"
// @Router		/market/my-listings [get]
func (h *handler) ListMyListings(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}
	var typ *entity.ListingType
	if t := r.URL.Query().Get("type"); t != "" {
		tp := entity.ListingType(t)
		typ = &tp
	}
	list, err := h.listingService.ListListingsByUserID(r.Context(), userID, typ)
	if err != nil {
		return nil, toServiceError(err)
	}
	return listingsWithPricesInTON(list), nil
}

// UpdateListingRequest is the body for PATCH /api/v1/market/listings/:id.
// Channel cannot be changed after creation. Prices if set must be [["<number_of_hours>hr", <price>], ...].
type UpdateListingRequest struct {
	Status      *string         `json:"status,omitempty"`
	Type        *string         `json:"type,omitempty"`
	Prices      json.RawMessage `json:"prices,omitempty"`
	Categories  *[]string       `json:"categories,omitempty"`
	Description *string         `json:"description,omitempty"`
}

// @Security	JWT
// @Tags		Market
// @Summary	Update listing
// @Accept		json
// @Produce	json
// @Param		id		path		int										true	"Listing ID"
// @Param		request	body		UpdateListingRequest					true	"fields to update"
// @Success	200		{object}	response.Template{data=entity.Listing}	"Updated listing"
// @Failure	400		{object}	response.Template{data=string}			"Bad request"
// @Failure	401		{object}	response.Template{data=string}			"Unauthorized"
// @Failure	403		{object}	response.Template{data=string}			"Forbidden"
// @Failure	404		{object}	response.Template{data=string}			"Not found"
// @Router		/market/listings/{id} [patch]
func (h *handler) UpdateListing(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid id", Code: apperrors.ErrorCodeBadRequest}
	}

	existing, err := h.listingService.GetListing(r.Context(), id)
	if err != nil || existing == nil {
		return nil, toServiceError(err)
	}

	var req UpdateListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid body", Code: apperrors.ErrorCodeBadRequest}
	}
	if len(req.Prices) > 0 {
		if err := domain.ValidateListingPrices(req.Prices); err != nil {
			return nil, apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeBadRequest}
		}
	}
	if req.Categories != nil {
		if err := domain.ValidateListingCategories(*req.Categories); err != nil {
			return nil, apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeBadRequest}
		}
	}

	l := *existing
	l.ID = id
	if req.Status != nil {
		l.Status = entity.ListingStatus(*req.Status)
	}
	// Channel cannot be changed after creation (keep existing)
	if req.Type != nil {
		l.Type = entity.ListingType(*req.Type)
	}
	if req.Prices != nil {
		pricesNanoton, err := domain.ConvertListingPricesTONToNanoton(req.Prices)
		if err != nil {
			return nil, apperrors.ServiceError{Err: err, Message: "invalid prices", Code: apperrors.ErrorCodeBadRequest}
		}
		l.Prices = pricesNanoton
	}
	if req.Categories != nil {
		l.Categories = categoriesToRaw(*req.Categories)
	}
	if req.Description != nil {
		l.Description = *req.Description
	}

	if err := h.listingService.UpdateListing(r.Context(), userID, &l); err != nil {
		return nil, toServiceError(err)
	}
	updated, _ := h.listingService.GetListing(r.Context(), id)
	return listingWithPricesInTON(updated), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Delete listing
// @Produce	json
// @Param		id	path		int										true	"Listing ID"
// @Success	200		{object}	response.Template{data=string}			"Deleted"
// @Failure	400		{object}	response.Template{data=string}			"Bad request"
// @Failure	401		{object}	response.Template{data=string}			"Unauthorized"
// @Failure	403		{object}	response.Template{data=string}			"Forbidden"
// @Failure	404		{object}	response.Template{data=string}			"Not found"
// @Router		/market/listings/{id} [delete]
func (h *handler) DeleteListing(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return nil, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid id", Code: apperrors.ErrorCodeBadRequest}
	}

	if err := h.listingService.DeleteListing(r.Context(), userID, id); err != nil {
		return nil, toServiceError(err)
	}
	return map[string]string{"status": "deleted"}, nil
}
