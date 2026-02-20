package domain

import (
	"encoding/json"
	"math"
)

const NanotonPerTON = 1e9

// TONToNanoton converts TON (e.g. 99.5) to nanoton (integer). Rounds to nearest.
func TONToNanoton(ton float64) int64 {
	return int64(math.Round(ton * NanotonPerTON))
}

// NanotonToTON converts nanoton to TON for API display.
func NanotonToTON(nanoton int64) float64 {
	return float64(nanoton) / NanotonPerTON
}

// ConvertListingPricesTONToNanoton converts prices JSON from TON to nanoton. Input: [["24hr", 99.5], ...], output: [["24hr", 99500000000], ...].
func ConvertListingPricesTONToNanoton(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var slots []json.RawMessage
	if err := json.Unmarshal(raw, &slots); err != nil {
		return nil, err
	}
	out := make([][]interface{}, 0, len(slots))
	for _, slot := range slots {
		var pair []interface{}
		if err := json.Unmarshal(slot, &pair); err != nil || len(pair) != 2 {
			continue
		}
		ton, ok := parsePriceNumber(pair[1])
		if !ok || ton < 0 {
			continue
		}
		out = append(out, []interface{}{pair[0], TONToNanoton(ton)})
	}
	return json.Marshal(out)
}

// ConvertListingPricesNanotonToTON converts prices JSON to TON for API.
func ConvertListingPricesNanotonToTON(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var slots []json.RawMessage
	if err := json.Unmarshal(raw, &slots); err != nil {
		return nil, err
	}
	out := make([][]interface{}, 0, len(slots))
	for _, slot := range slots {
		var pair []interface{}
		if err := json.Unmarshal(slot, &pair); err != nil || len(pair) != 2 {
			continue
		}
		n, ok := parsePriceAsInt64(pair[1])
		if !ok {
			continue
		}
		out = append(out, []interface{}{pair[0], NanotonToTON(n)})
	}
	return json.Marshal(out)
}

func parsePriceAsInt64(v interface{}) (int64, bool) {
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case json.Number:
		n, err := x.Int64()
		return n, err == nil
	default:
		return 0, false
	}
}
