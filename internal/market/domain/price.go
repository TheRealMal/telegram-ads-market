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

// ConvertListingPricesTONToNanoton converts prices JSON from TON to nanoton. Input: [["post", "24hr", 99.5], ...], output: [["post", "24hr", 99500000000], ...].
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
		var triple []interface{}
		if err := json.Unmarshal(slot, &triple); err != nil || len(triple) != 3 {
			continue
		}
		ton, ok := ParsePriceNumber(triple[2])
		if !ok || ton < 0 {
			continue
		}
		out = append(out, []interface{}{triple[0], triple[1], TONToNanoton(ton)})
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
		var triple []interface{}
		if err := json.Unmarshal(slot, &triple); err != nil || len(triple) != 3 {
			continue
		}
		n, ok := ParsePriceAsInt64(triple[2])
		if !ok {
			continue
		}
		out = append(out, []interface{}{triple[0], triple[1], NanotonToTON(n)})
	}
	return json.Marshal(out)
}

func ParsePriceAsInt64(v interface{}) (int64, bool) {
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
