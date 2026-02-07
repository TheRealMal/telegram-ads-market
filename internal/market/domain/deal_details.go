package domain

import (
	"encoding/json"
	"errors"
)

var ErrDealDetailsInvalid = errors.New("deal details must contain only a \"message\" field (string)")

// ValidateDealDetails parses raw as JSON and ensures it has at most one key "message" (string).
// Returns canonical JSON for storage. Empty or null input becomes {}.
func ValidateDealDetails(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage("{}"), nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, ErrDealDetailsInvalid
	}
	// No extra keys: only "message" allowed
	for k := range m {
		if k != "message" {
			return nil, ErrDealDetailsInvalid
		}
	}
	msg, _ := m["message"]
	if msg == nil {
		return json.RawMessage("{}"), nil
	}
	msgStr, ok := msg.(string)
	if !ok {
		return nil, ErrDealDetailsInvalid
	}
	canon := map[string]string{"message": msgStr}
	return json.Marshal(canon)
}
