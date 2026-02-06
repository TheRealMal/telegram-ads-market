package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// ComputeDealSignature returns a deterministic signature for [type, duration, price, details, user_id].
// Including user_id binds the signature to the signer (lessor or lessee).
func ComputeDealSignature(dealType string, duration, price int64, details json.RawMessage, userID int64) string {
	h := sha256.New()
	h.Write([]byte(dealType))
	h.Write([]byte(fmt.Sprintf("%d", duration)))
	h.Write([]byte(fmt.Sprintf("%d", price)))
	h.Write(details)
	h.Write([]byte(fmt.Sprintf("%d", userID)))
	return hex.EncodeToString(h.Sum(nil))
}
