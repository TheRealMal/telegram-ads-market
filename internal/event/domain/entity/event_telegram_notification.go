package entity

import "strconv"

const streamKeyTelegramNotification = "events:telegram_notification"

type EventTelegramNotification struct {
	ID       string `json:"-"`
	ChatID   int64  `json:"chat_id"`
	Message  string `json:"message"`
	ThreadID int64  `json:"thread_id,omitempty"`  // forum topic message_thread_id
	Photo    string `json:"photo,omitempty"`       // photo file_id
	Video    string `json:"video,omitempty"`       // video file_id
	Entities string `json:"entities,omitempty"`    // JSON-serialized []telegram.MessageEntity
	Buttons  string `json:"buttons,omitempty"`     // JSON-serialized [][]telegram.InlineKeyboardButton
}

var _ Event = (*EventTelegramNotification)(nil)

func (e *EventTelegramNotification) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"chat_id": strconv.FormatInt(e.ChatID, 10),
		"message": e.Message,
	}
	if e.ThreadID != 0 {
		m["thread_id"] = strconv.FormatInt(e.ThreadID, 10)
	}
	if e.Photo != "" {
		m["photo"] = e.Photo
	}
	if e.Video != "" {
		m["video"] = e.Video
	}
	if e.Entities != "" {
		m["entities"] = e.Entities
	}
	if e.Buttons != "" {
		m["buttons"] = e.Buttons
	}
	return m
}

func (e *EventTelegramNotification) FromMap(m map[string]interface{}) {
	e.ChatID = int64FromMap(m, "chat_id")
	e.Message = stringFromMap(m, "message")
	e.ThreadID = int64FromMap(m, "thread_id")
	e.Photo = stringFromMap(m, "photo")
	e.Video = stringFromMap(m, "video")
	e.Entities = stringFromMap(m, "entities")
	e.Buttons = stringFromMap(m, "buttons")
}

func (e *EventTelegramNotification) StreamKey() string {
	return streamKeyTelegramNotification
}
