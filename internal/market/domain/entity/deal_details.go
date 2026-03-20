package entity

import (
	"encoding/json"
	"errors"
	"net/url"
	"time"
)

var ErrDealDetailsInvalid = errors.New("deal details contains invalid or disallowed fields")

var ValidButtonStyles = map[string]bool{"danger": true, "success": true, "primary": true, "default": true}

// DealDetailsButton represents an inline button on the ad post.
type DealDetailsButton struct {
	Text  string `json:"text"`
	URL   string `json:"url"`
	Style string `json:"style"` // "danger", "success", "primary", "default"
	Emoji string `json:"emoji"` // emoji string, or "" for none
}

// IsValid reports whether the button has a valid style and a valid http/https URL if set.
func (b *DealDetailsButton) IsValid() bool {
	if !ValidButtonStyles[b.Style] {
		return false
	}
	if b.URL != "" {
		parsed, err := url.ParseRequestURI(b.URL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return false
		}
	}
	return true
}

// ValidateDealDetails parses raw as JSON and ensures it contains only allowed keys.
// Returns canonical JSON for storage. Empty or null input becomes {}.
func ValidateDealDetails(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage("{}"), nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, ErrDealDetailsInvalid
	}
	allowedKeys := map[string]bool{
		"message":          true,
		"posted_at":        true,
		"media_type":       true,
		"media_file_id":    true,
		"button":           true,
		"buttons":          true, // array of DealDetailsButton
		"entities":         true,
		"caption_entities": true,
	}
	for k := range m {
		if !allowedKeys[k] {
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
	var mediaType, mediaFileID string
	if mt, ok := m["media_type"]; ok && mt != nil {
		mtStr, ok := mt.(string)
		if !ok {
			return nil, ErrDealDetailsInvalid
		}
		if mtStr != "photo" && mtStr != "video" && mtStr != "animation" {
			return nil, ErrDealDetailsInvalid
		}
		mediaType = mtStr
	}
	if mf, ok := m["media_file_id"]; ok && mf != nil {
		mfStr, ok := mf.(string)
		if !ok {
			return nil, ErrDealDetailsInvalid
		}
		mediaFileID = mfStr
	}
	if mediaType != "" && mediaFileID == "" {
		return nil, ErrDealDetailsInvalid
	}
	var button *DealDetailsButton
	if btnRaw, ok := m["button"]; ok && btnRaw != nil {
		btnBytes, err := json.Marshal(btnRaw)
		if err != nil {
			return nil, ErrDealDetailsInvalid
		}
		var btn DealDetailsButton
		if err := json.Unmarshal(btnBytes, &btn); err != nil {
			return nil, ErrDealDetailsInvalid
		}
		if !btn.IsValid() {
			return nil, ErrDealDetailsInvalid
		}
		button = &btn
	}

	var buttons []DealDetailsButton
	if btnsRaw, ok := m["buttons"]; ok && btnsRaw != nil {
		btnsBytes, err := json.Marshal(btnsRaw)
		if err != nil {
			return nil, ErrDealDetailsInvalid
		}
		if err := json.Unmarshal(btnsBytes, &buttons); err != nil {
			return nil, ErrDealDetailsInvalid
		}
		for _, btn := range buttons {
			if !btn.IsValid() {
				return nil, ErrDealDetailsInvalid
			}
		}
	}

	var entities []interface{}
	if ent, ok := m["entities"]; ok && ent != nil {
		entArr, ok := ent.([]interface{})
		if !ok {
			return nil, ErrDealDetailsInvalid
		}
		if err := validateEntityURLs(entArr); err != nil {
			return nil, err
		}
		entities = entArr
	}
	var captionEntities []interface{}
	if ce, ok := m["caption_entities"]; ok && ce != nil {
		ceArr, ok := ce.([]interface{})
		if !ok {
			return nil, ErrDealDetailsInvalid
		}
		if err := validateEntityURLs(ceArr); err != nil {
			return nil, err
		}
		captionEntities = ceArr
	}

	canon := make(map[string]interface{})
	if message != "" {
		canon["message"] = message
	}
	if postedAt != "" {
		canon["posted_at"] = postedAt
	}
	if mediaType != "" {
		canon["media_type"] = mediaType
		canon["media_file_id"] = mediaFileID
	}
	if button != nil {
		canon["button"] = button
	}
	if len(buttons) > 0 {
		canon["buttons"] = buttons
	}
	if len(entities) > 0 {
		canon["entities"] = entities
	}
	if len(captionEntities) > 0 {
		canon["caption_entities"] = captionEntities
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

// GetPostedAtFromDetails returns the "posted_at" field from deal details (RFC3339), or (zero time, false) if not set or invalid.
func GetPostedAtFromDetails(details json.RawMessage) (time.Time, bool) {
	if len(details) == 0 || string(details) == "null" {
		return time.Time{}, false
	}
	var m map[string]interface{}
	if err := json.Unmarshal(details, &m); err != nil {
		return time.Time{}, false
	}
	pa, ok := m["posted_at"]
	if !ok || pa == nil {
		return time.Time{}, false
	}
	s, ok := pa.(string)
	if !ok || s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// GetMediaFromDetails returns (mediaType, mediaFileID) from details. mediaType is "photo" or "video", or "" if none.
func GetMediaFromDetails(details json.RawMessage) (mediaType string, mediaFileID string) {
	if len(details) == 0 || string(details) == "null" {
		return "", ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(details, &m); err != nil {
		return "", ""
	}
	if mt, ok := m["media_type"].(string); ok {
		mediaType = mt
	}
	if mf, ok := m["media_file_id"].(string); ok {
		mediaFileID = mf
	}
	return mediaType, mediaFileID
}

// GetButtonFromDetails returns the button config from details, or nil if not set.
func GetButtonFromDetails(details json.RawMessage) *DealDetailsButton {
	if len(details) == 0 || string(details) == "null" {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(details, &m); err != nil {
		return nil
	}
	btnRaw, ok := m["button"]
	if !ok || btnRaw == nil {
		return nil
	}
	btnBytes, err := json.Marshal(btnRaw)
	if err != nil {
		return nil
	}
	var btn DealDetailsButton
	if err := json.Unmarshal(btnBytes, &btn); err != nil {
		return nil
	}
	return &btn
}

// SetMessageInDetails updates the "message" key in details JSON. Preserves other keys.
func SetMessageInDetails(details json.RawMessage, message string) (json.RawMessage, error) {
	m := parseDetailsMap(details)
	m["message"] = message
	return json.Marshal(m)
}

// SetMediaInDetails updates/removes the "media_type" and "media_file_id" keys.
// Pass empty strings to remove media.
func SetMediaInDetails(details json.RawMessage, mediaType string, mediaFileID string) (json.RawMessage, error) {
	m := parseDetailsMap(details)
	if mediaType == "" || mediaFileID == "" {
		delete(m, "media_type")
		delete(m, "media_file_id")
	} else {
		m["media_type"] = mediaType
		m["media_file_id"] = mediaFileID
	}
	return json.Marshal(m)
}

// SetButtonInDetails updates/removes the "button" key. Pass nil to remove.
func SetButtonInDetails(details json.RawMessage, button *DealDetailsButton) (json.RawMessage, error) {
	m := parseDetailsMap(details)
	if button == nil {
		delete(m, "button")
	} else {
		m["button"] = button
	}
	return json.Marshal(m)
}

// validateEntityURLs checks that any text_link entity in the array has an http/https URL.
func validateEntityURLs(entities []interface{}) error {
	for _, e := range entities {
		em, ok := e.(map[string]interface{})
		if !ok {
			return ErrDealDetailsInvalid
		}
		eType, _ := em["type"].(string)
		if eType == "text_link" {
			eURL, _ := em["url"].(string)
			if eURL == "" {
				return ErrDealDetailsInvalid
			}
			parsed, err := url.ParseRequestURI(eURL)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return ErrDealDetailsInvalid
			}
		}
	}
	return nil
}

// parseDetailsMap unmarshals details into a map, returning empty map on error.
func parseDetailsMap(details json.RawMessage) map[string]interface{} {
	if len(details) == 0 || string(details) == "null" {
		return make(map[string]interface{})
	}
	var m map[string]interface{}
	if err := json.Unmarshal(details, &m); err != nil {
		return make(map[string]interface{})
	}
	return m
}

// GetRawEntitiesFromDetails returns the raw JSON "entities" array from details, or nil.
func GetRawEntitiesFromDetails(details json.RawMessage) json.RawMessage {
	return getRawField(details, "entities")
}

// GetRawCaptionEntitiesFromDetails returns the raw JSON "caption_entities" array from details, or nil.
func GetRawCaptionEntitiesFromDetails(details json.RawMessage) json.RawMessage {
	return getRawField(details, "caption_entities")
}

func getRawField(details json.RawMessage, key string) json.RawMessage {
	if len(details) == 0 || string(details) == "null" {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(details, &m); err != nil {
		return nil
	}
	raw, ok := m[key]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return raw
}

// SetRawEntitiesInDetails sets/removes the "entities" key using raw JSON. Pass nil to remove.
func SetRawEntitiesInDetails(details json.RawMessage, entitiesJSON json.RawMessage) (json.RawMessage, error) {
	return setRawField(details, "entities", entitiesJSON)
}

// SetRawCaptionEntitiesInDetails sets/removes the "caption_entities" key using raw JSON. Pass nil to remove.
func SetRawCaptionEntitiesInDetails(details json.RawMessage, entitiesJSON json.RawMessage) (json.RawMessage, error) {
	return setRawField(details, "caption_entities", entitiesJSON)
}

func setRawField(details json.RawMessage, key string, raw json.RawMessage) (json.RawMessage, error) {
	m := parseDetailsMap(details)
	if len(raw) == 0 || string(raw) == "null" {
		delete(m, key)
	} else {
		var val interface{}
		if err := json.Unmarshal(raw, &val); err != nil {
			return nil, err
		}
		m[key] = val
	}
	return json.Marshal(m)
}

// GetButtonsFromDetails returns all buttons from deal details. Supports both the new "buttons"
// array format and the legacy single "button" format for backward compatibility.
func GetButtonsFromDetails(details json.RawMessage) []DealDetailsButton {
	if len(details) == 0 || string(details) == "null" {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(details, &m); err != nil {
		return nil
	}
	// Try "buttons" first (new array format)
	if btnsRaw, ok := m["buttons"]; ok && btnsRaw != nil {
		btnsBytes, err := json.Marshal(btnsRaw)
		if err != nil {
			return nil
		}
		var btns []DealDetailsButton
		if err := json.Unmarshal(btnsBytes, &btns); err != nil {
			return nil
		}
		return btns
	}
	// Fallback: "button" (old single-button format)
	if btnRaw, ok := m["button"]; ok && btnRaw != nil {
		btnBytes, err := json.Marshal(btnRaw)
		if err != nil {
			return nil
		}
		var btn DealDetailsButton
		if err := json.Unmarshal(btnBytes, &btn); err != nil {
			return nil
		}
		return []DealDetailsButton{btn}
	}
	return nil
}

// StripNonMediaFields removes message, buttons, button, entities, and caption_entities from details,
// keeping only media_type, media_file_id, and posted_at. Used to enforce story deals as media-only.
func StripNonMediaFields(details json.RawMessage) json.RawMessage {
	m := parseDetailsMap(details)
	delete(m, "message")
	delete(m, "button")
	delete(m, "buttons")
	delete(m, "entities")
	delete(m, "caption_entities")
	b, _ := json.Marshal(m)
	return b
}

// SetButtonsInDetails sets the "buttons" array in details JSON, removing the legacy "button" key.
// Pass an empty slice to remove all buttons.
func SetButtonsInDetails(details json.RawMessage, buttons []DealDetailsButton) (json.RawMessage, error) {
	m := parseDetailsMap(details)
	delete(m, "button") // remove old single-button key
	if len(buttons) == 0 {
		delete(m, "buttons")
	} else {
		m["buttons"] = buttons
	}
	return json.Marshal(m)
}
