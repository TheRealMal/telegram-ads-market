package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	apperrors "ads-mrkt/internal/errors"
	"ads-mrkt/internal/market/application/market/http/model"
	"ads-mrkt/internal/market/domain/entity"
	listingservice "ads-mrkt/internal/market/service/listing"
	_ "ads-mrkt/internal/server/templates/response"
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

func parseListListingsQuery(r *http.Request) (typ *entity.ListingType, categories []string, minFollowers *int64) {
	if t := r.URL.Query().Get("type"); t != "" {
		tp := entity.ListingType(t)
		typ = &tp
	}
	if c := r.URL.Query().Get("categories"); c != "" {
		for _, s := range splitComma(c) {
			if s != "" {
				categories = append(categories, s)
			}
		}
	}
	if m := r.URL.Query().Get("min_followers"); m != "" {
		if n, err := strconv.ParseInt(m, 10, 64); err == nil && n >= 0 {
			minFollowers = &n
		}
	}
	return typ, categories, minFollowers
}

// @Security	JWT
// @Tags		Market
// @Summary	Create listing
// @Accept		json
// @Produce	json
// @Param		request	body		model.CreateListingRequest				true	"listing body"
// @Success	200		{object}	response.Template{data=entity.Listing}	"Created listing"
// @Failure	400		{object}	response.Template{data=string}			"Bad request"
// @Failure	401		{object}	response.Template{data=string}			"Unauthorized"
// @Failure	403		{object}	response.Template{data=string}			"Forbidden"
// @Router		/market/listings [post]
func (h *handler) CreateListing(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}

	var req model.CreateListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid body", Code: apperrors.ErrorCodeBadRequest}
	}

	l, err := h.listingService.CreateListing(r.Context(), userID, listingservice.CreateListingInput{
		Status:       req.Status,
		ChannelID:    req.ChannelID,
		Type:         req.Type,
		PricesTON:    req.Prices,
		Categories:   req.Categories,
		Description:  req.Description,
		PreparedPost: req.PreparedPost,
	})
	if err != nil {
		return nil, toServiceError(err)
	}
	return model.ListingWithPricesInTON(l), nil
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
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}

	l, err := h.listingService.GetListing(r.Context(), id)
	if err != nil {
		return nil, toServiceError(err)
	}
	if l == nil {
		return nil, apperrors.ServiceError{Err: nil, Message: "not found", Code: apperrors.ErrorCodeNotFound}
	}
	return model.ListingWithPricesInTON(l), nil
}

// @Tags		Market
// @Summary	List all listings with optional type, categories, and min_followers filter (public, no auth)
// @Produce	json
// @Param		type			query		string										false	"Filter by type: lessor | lessee"
// @Param		categories		query		string										false	"Comma-separated categories (e.g. Tech,Crypto)"
// @Param		min_followers	query		int											false	"Min channel followers (only lessor listings with stats)"
// @Success	200				{object}	response.Template{data=[]entity.Listing}	"List of listings"
// @Router		/market/listings [get]
func (h *handler) ListListings(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	typ, categories, minFollowers := parseListListingsQuery(r)
	list, err := h.listingService.ListListingsAll(r.Context(), typ, categories, minFollowers)
	if err != nil {
		return nil, toServiceError(err)
	}
	return model.ListingsWithPricesInTON(list), nil
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
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	typ, _, _ := parseListListingsQuery(r)
	list, err := h.listingService.ListListingsByUserID(r.Context(), userID, typ)
	if err != nil {
		return nil, toServiceError(err)
	}
	return model.ListingsWithPricesInTON(list), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Update listing
// @Accept		json
// @Produce	json
// @Param		id		path		int										true	"Listing ID"
// @Param		request	body		model.UpdateListingRequest				true	"fields to update"
// @Success	200		{object}	response.Template{data=entity.Listing}	"Updated listing"
// @Failure	400		{object}	response.Template{data=string}			"Bad request"
// @Failure	401		{object}	response.Template{data=string}			"Unauthorized"
// @Failure	403		{object}	response.Template{data=string}			"Forbidden"
// @Failure	404		{object}	response.Template{data=string}			"Not found"
// @Router		/market/listings/{id} [patch]
func (h *handler) UpdateListing(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}

	var req model.UpdateListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, apperrors.ServiceError{Err: err, Message: "invalid body", Code: apperrors.ErrorCodeBadRequest}
	}

	if err := h.listingService.UpdateListing(r.Context(), userID, id, listingservice.UpdateListingInput{
		Status:       req.Status,
		Type:         req.Type,
		PricesTON:    req.Prices,
		Categories:   req.Categories,
		Description:  req.Description,
		PreparedPost: req.PreparedPost,
	}); err != nil {
		return nil, toServiceError(err)
	}

	updated, err := h.listingService.GetListing(r.Context(), id)
	if err != nil {
		return nil, toServiceError(err)
	}
	if updated == nil {
		return nil, apperrors.ServiceError{Err: nil, Message: "not found", Code: apperrors.ErrorCodeNotFound}
	}
	return model.ListingWithPricesInTON(updated), nil
}

// @Security	JWT
// @Tags		Market
// @Summary	Delete listing
// @Produce	json
// @Param		id	path		int								true	"Listing ID"
// @Success	200	{object}	response.Template{data=string}	"Deleted"
// @Failure	400	{object}	response.Template{data=string}	"Bad request"
// @Failure	401	{object}	response.Template{data=string}	"Unauthorized"
// @Failure	403	{object}	response.Template{data=string}	"Forbidden"
// @Failure	404	{object}	response.Template{data=string}	"Not found"
// @Router		/market/listings/{id} [delete]
func (h *handler) DeleteListing(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID, err := requireUserID(r)
	if err != nil {
		return nil, err
	}
	id, err := parsePathID(r, "id")
	if err != nil {
		return nil, err
	}

	if err := h.listingService.DeleteListing(r.Context(), userID, id); err != nil {
		return nil, toServiceError(err)
	}
	return map[string]string{"status": "deleted"}, nil
}
