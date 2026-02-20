package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
)

// ListingPricesFormat is the required JSON format: array of [duration_string, price_number].
// Example: [["24hr", 100], ["48hr", 200]]. Duration must match \<number_of_hours>hr (e.g. "24hr", "1hr").
var durationRegex = regexp.MustCompile(`^\d+hr$`)

// DealPriceMatchesListing checks that the deal's type, duration, and price correspond to an option in the listing's prices.
// listingPrices must be a JSON array of [durationStr, priceNanoton] pairs (prices stored in nanoton). Returns false if no match.
func DealPriceMatchesListing(listingPrices json.RawMessage, dealType string, dealDuration int64, dealPriceNanoton int64) bool {
	if len(listingPrices) == 0 {
		return false
	}
	var slots []json.RawMessage
	if err := json.Unmarshal(listingPrices, &slots); err != nil {
		return false
	}
	dealTypeNorm := normalizeDurationType(dealType)
	for _, slot := range slots {
		var pair []interface{}
		if err := json.Unmarshal(slot, &pair); err != nil || len(pair) != 2 {
			continue
		}
		durStr, ok := pair[0].(string)
		if !ok || !durationRegex.MatchString(durStr) {
			continue
		}
		price, ok := parsePriceAsInt64(pair[1])
		if !ok {
			continue
		}
		if normalizeDurationType(durStr) != dealTypeNorm || price != dealPriceNanoton {
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

func normalizeDurationType(s string) string {
	s = durationRegex.FindString(s)
	if s == "" {
		return ""
	}
	return s
}

func parseDurationHours(durStr string) int64 {
	n := durationRegex.FindString(durStr)
	if n == "" || len(n) < 3 {
		return -1
	}
	h, _ := strconv.ParseInt(n[:len(n)-2], 10, 64)
	return h
}

// ValidateListingPrices checks that raw is a JSON array of pairs [["<n>hr", price], ...].
func ValidateListingPrices(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var slots []json.RawMessage
	if err := json.Unmarshal(raw, &slots); err != nil {
		return fmt.Errorf("prices: must be a JSON array: %w", err)
	}
	for i, slot := range slots {
		var pair []interface{}
		if err := json.Unmarshal(slot, &pair); err != nil {
			return fmt.Errorf("prices[%d]: must be a 2-element array [duration, price]: %w", i, err)
		}
		if len(pair) != 2 {
			return fmt.Errorf("prices[%d]: must have exactly 2 elements [duration, price]", i)
		}
		durStr, ok := pair[0].(string)
		if !ok {
			return fmt.Errorf("prices[%d][0]: duration must be a string (e.g. \"24hr\")", i)
		}
		if !durationRegex.MatchString(durStr) {
			return fmt.Errorf("prices[%d][0]: duration must match <number>hr (e.g. \"24hr\")", i)
		}
		switch v := pair[1].(type) {
		case float64:
			if v < 0 || (v != v) {
				return fmt.Errorf("prices[%d][1]: price must be a non-negative number", i)
			}
		case json.Number:
			f, err := v.Float64()
			if err != nil || v.String() == "" || f < 0 || (f != f) {
				return fmt.Errorf("prices[%d][1]: price must be a non-negative number", i)
			}
			_ = f
		default:
			return fmt.Errorf("prices[%d][1]: price must be a number", i)
		}
	}
	return nil
}
