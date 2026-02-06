package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// ListingPricesFormat is the required JSON format: array of [duration_string, price_number].
// Example: [["24hr", 100], ["48hr", 200]]. Duration must match \<number_of_hours>hr (e.g. "24hr", "1hr").
var durationRegex = regexp.MustCompile(`^\d+hr$`)

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
			if v < 0 || v != float64(int64(v)) {
				return fmt.Errorf("prices[%d][1]: price must be a non-negative integer", i)
			}
		case json.Number:
			if _, err := v.Int64(); err != nil || v.String() == "" {
				return fmt.Errorf("prices[%d][1]: price must be an integer", i)
			}
		default:
			return fmt.Errorf("prices[%d][1]: price must be a number", i)
		}
	}
	return nil
}
