package domain

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrDealDetailsInvalid = errors.New("deal details must contain only \"message\" (string) and optional \"posted_at\" (RFC3339 datetime)")

// ValidateDealDetails parses raw as JSON and ensures it has only "message" (string) and optional "posted_at" (RFC3339).
// Returns canonical JSON for storage. Empty or null input becomes {}.
func ValidateDealDetails(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage("{}"), nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, ErrDealDetailsInvalid
	}
	for k := range m {
		if k != "message" && k != "posted_at" {
			return nil, ErrDealDetailsInvalid
		}
	}
	var message string
	if msg, ok := m["message"]; ok && msg != nil {
		msgStr, ok := msg.(string)
		if !ok {
			return nil, ErrDealDetailsInvalid
		}
		message = msgStr
	}
	var postedAt string
	if pa, ok := m["posted_at"]; ok && pa != nil {
		paStr, ok := pa.(string)
		if !ok {
			return nil, ErrDealDetailsInvalid
		}
		if paStr != "" {
			if _, err := time.Parse(time.RFC3339, paStr); err != nil {
				return nil, ErrDealDetailsInvalid
			}
			postedAt = paStr
		}
	}
	canon := make(map[string]string)
	if message != "" {
		canon["message"] = message
	}
	if postedAt != "" {
		canon["posted_at"] = postedAt
	}
	return json.Marshal(canon)
}

// GetMessageFromDetails returns the "message" field from deal details JSON, or empty string.
func GetMessageFromDetails(details json.RawMessage) string {
	if len(details) == 0 || string(details) == "null" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(details, &m); err != nil {
		return ""
	}
	if msg, ok := m["message"]; ok && msg != nil {
		if s, ok := msg.(string); ok {
			return s
		}
	}
	return ""
}
