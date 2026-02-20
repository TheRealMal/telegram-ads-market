package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// ComputeDealSignature returns a deterministic signature for deal terms and payout addresses.
// lessorPayoutRaw and lesseePayoutRaw are TON addresses in raw format; use "" when not set.
// price is in nanoton.
func ComputeDealSignature(dealType string, duration int64, priceNanoton int64, details json.RawMessage, userID int64, lessorPayoutRaw, lesseePayoutRaw string) string {
	h := sha256.New()
	h.Write([]byte(dealType))
	h.Write([]byte(fmt.Sprintf("%d", duration)))
	h.Write([]byte(fmt.Sprintf("%d", priceNanoton)))
	h.Write(details)
	h.Write([]byte(fmt.Sprintf("%d", userID)))
	h.Write([]byte(lessorPayoutRaw))
	h.Write([]byte(lesseePayoutRaw))
	return hex.EncodeToString(h.Sum(nil))
}
