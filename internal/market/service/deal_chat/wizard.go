package deal_chat

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	wizardStepText    = "text"
	wizardStepMedia   = "media"
	wizardStepButtons = "buttons"

	wizardKeyPrefix = "craft"
	wizardTTL       = 1 * time.Hour
)

type wizardState struct {
	Step    string          `json:"step"`
	DealID  int64           `json:"deal_id"`
	AdType  string          `json:"ad_type,omitempty"`
	Details json.RawMessage `json:"details"`
}

func wizardKey(userID, threadID int64) string {
	return fmt.Sprintf("%s:%d:%d", wizardKeyPrefix, userID, threadID)
}
