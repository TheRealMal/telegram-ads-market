package entity

import (
	"ads-mrkt/internal/helpers/telegram"
	"encoding/json"
)

const (
	streamKeyTelegramUpdate = "events:telegram_update"
)

type EventTelegramUpdate struct {
	ID        string           `json:"-"`
	Update    *telegram.Update `json:"update"`
	Timestamp int64            `json:"timestamp"`
}

var _ Event = (*EventTelegramUpdate)(nil)

func (e *EventTelegramUpdate) ToMap() map[string]interface{} {
	raw, err := json.Marshal(e.Update)
	if err != nil {
		return nil
	}
	return map[string]interface{}{
		"update":    string(raw),
		"timestamp": e.Timestamp,
	}
}

func (e *EventTelegramUpdate) FromMap(m map[string]interface{}) {
	raw, ok := m["update"].(string)
	if !ok {
		return
	}
	e.Update = &telegram.Update{}
	err := json.Unmarshal([]byte(raw), e.Update)
	if err != nil {
		return
	}
	e.Timestamp = mustParseInt64(m["timestamp"])
}

func (e *EventTelegramUpdate) StreamKey() string {
	return streamKeyTelegramUpdate
}
