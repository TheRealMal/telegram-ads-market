package entity

import "strconv"

const streamKeyTelegramNotification = "events:telegram_notification"

type EventTelegramNotification struct {
	ID      string `json:"-"`
	ChatID  int64  `json:"chat_id"`
	Message string `json:"message"`
}

var _ Event = (*EventTelegramNotification)(nil)

func (e *EventTelegramNotification) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"chat_id": strconv.FormatInt(e.ChatID, 10),
		"message": e.Message,
	}
}

func (e *EventTelegramNotification) FromMap(m map[string]interface{}) {
	e.ChatID = int64FromMap(m, "chat_id")
	e.Message = stringFromMap(m, "message")
}

func (e *EventTelegramNotification) StreamKey() string {
	return streamKeyTelegramNotification
}
