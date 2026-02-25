package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"ads-mrkt/internal/market/domain/entity"
)

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

func dealPayoutAddresses(d *entity.Deal) (lessorPayout, lesseePayout string) {
	if d.LessorPayoutAddress != nil {
		lessorPayout = *d.LessorPayoutAddress
	}
	if d.LesseePayoutAddress != nil {
		lesseePayout = *d.LesseePayoutAddress
	}
	return lessorPayout, lesseePayout
}

func DealSignaturesMatch(d *entity.Deal) bool {
	if d.LessorSignature == nil || d.LesseeSignature == nil {
		return false
	}
	lessorPayout, lesseePayout := dealPayoutAddresses(d)
	expectedLessor := ComputeDealSignature(d.Type, d.Duration, d.Price, d.Details, d.LessorID, lessorPayout, lesseePayout)
	expectedLessee := ComputeDealSignature(d.Type, d.Duration, d.Price, d.Details, d.LesseeID, lessorPayout, lesseePayout)
	return *d.LessorSignature == expectedLessor && *d.LesseeSignature == expectedLessee
}
