package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"ads-mrkt/internal/market/domain/entity"
)

// ListingPricesFormat is the required JSON format: array of [ad_type, duration_string, price_number].
// Example: [["post", "24hr", 100], ["instant_post", "48hr", 200]].
// ad_type must be one of "post", "instant_post". Duration must match \<number_of_hours>hr (e.g. "24hr", "1hr").
var durationRegex = regexp.MustCompile(`^\d+hr$`)

var validAdTypes = map[string]bool{
	string(entity.AdTypePost):        true,
	string(entity.AdTypeInstantPost): true,
}

// DealPriceMatchesListing checks that the deal's ad type, duration, and price correspond to an option in the listing's prices.
// listingPrices must be a JSON array of [adType, durationStr, priceNanoton] triples (prices stored in nanoton). Returns false if no match.
func DealPriceMatchesListing(listingPrices json.RawMessage, dealType string, dealDuration int64, dealPriceNanoton int64) bool {
	if len(listingPrices) == 0 {
		return false
	}
	var slots []json.RawMessage
	if err := json.Unmarshal(listingPrices, &slots); err != nil {
		return false
	}
	for _, slot := range slots {
		var triple []interface{}
		if err := json.Unmarshal(slot, &triple); err != nil || len(triple) != 3 {
			continue
		}
		adType, ok := triple[0].(string)
		if !ok || !validAdTypes[adType] {
			continue
		}
		durStr, ok := triple[1].(string)
		if !ok || !durationRegex.MatchString(durStr) {
			continue
		}
		price, ok := parsePriceAsInt64(triple[2])
		if !ok {
			continue
		}
		if adType != dealType || price != dealPriceNanoton {
			continue
		}
		entryHours := parseDurationHours(durStr)
		if entryHours >= 0 && entryHours != dealDuration {
			continue
		}
		return true
	}
	return false
}

// ListingHasAdType checks whether the listing's prices contain any entry with the given ad type.
func ListingHasAdType(listingPrices json.RawMessage, adType string) bool {
	if len(listingPrices) == 0 {
		return false
	}
	var slots []json.RawMessage
	if err := json.Unmarshal(listingPrices, &slots); err != nil {
		return false
	}
	for _, slot := range slots {
		var triple []interface{}
		if err := json.Unmarshal(slot, &triple); err != nil || len(triple) != 3 {
			continue
		}
		at, ok := triple[0].(string)
		if ok && at == adType {
			return true
		}
	}
	return false
}

func parsePriceNumber(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func parseDurationHours(durStr string) int64 {
	n := durationRegex.FindString(durStr)
	if n == "" || len(n) < 3 {
		return -1
	}
	h, _ := strconv.ParseInt(n[:len(n)-2], 10, 64)
	return h
}

// ValidateListingPrices checks that raw is a JSON array of triples [["<ad_type>", "<n>hr", price], ...].
func ValidateListingPrices(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var slots []json.RawMessage
	if err := json.Unmarshal(raw, &slots); err != nil {
		return fmt.Errorf("prices: must be a JSON array: %w", err)
	}
	for i, slot := range slots {
		var triple []interface{}
		if err := json.Unmarshal(slot, &triple); err != nil {
			return fmt.Errorf("prices[%d]: must be a 3-element array [ad_type, duration, price]: %w", i, err)
		}
		if len(triple) != 3 {
			return fmt.Errorf("prices[%d]: must have exactly 3 elements [ad_type, duration, price]", i)
		}
		adType, ok := triple[0].(string)
		if !ok {
			return fmt.Errorf("prices[%d][0]: ad_type must be a string (e.g. \"post\")", i)
		}
		if !validAdTypes[adType] {
			return fmt.Errorf("prices[%d][0]: ad_type must be one of \"post\", \"instant_post\"", i)
		}
		durStr, ok := triple[1].(string)
		if !ok {
			return fmt.Errorf("prices[%d][1]: duration must be a string (e.g. \"24hr\")", i)
		}
		if !durationRegex.MatchString(durStr) {
			return fmt.Errorf("prices[%d][1]: duration must match <number>hr (e.g. \"24hr\")", i)
		}
		switch v := triple[2].(type) {
		case float64:
			if v < 0 || (v != v) {
				return fmt.Errorf("prices[%d][2]: price must be a non-negative number", i)
			}
		case json.Number:
			f, err := v.Float64()
			if err != nil || v.String() == "" || f < 0 || (f != f) {
				return fmt.Errorf("prices[%d][2]: price must be a non-negative number", i)
			}
			_ = f
		default:
			return fmt.Errorf("prices[%d][2]: price must be a number", i)
		}
	}
	return nil
}
