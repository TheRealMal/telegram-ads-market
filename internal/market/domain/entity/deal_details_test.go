package entity

import (
	"encoding/json"
	"testing"
)

// buildDetails is a helper to create a JSON object from a map for test input.
func buildDetails(m map[string]any) json.RawMessage {
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(b)
}

// --- ValidateDealDetails: 9 valid combinations ---

func TestValidateDealDetails_ValidCombinations(t *testing.T) {
	tests := []struct {
		name          string
		input         map[string]any
		wantMessage   string
		wantMediaType string
		wantMediaFile string
		wantButton    *DealDetailsButton
	}{
		{
			name:        "text only",
			input:       map[string]any{"message": "Hello world"},
			wantMessage: "Hello world",
		},
		{
			name: "text + image",
			input: map[string]any{
				"message":       "Check this out",
				"media_type":    "photo",
				"media_file_id": "file_abc123",
			},
			wantMessage:   "Check this out",
			wantMediaType: "photo",
			wantMediaFile: "file_abc123",
		},
		{
			name: "text + video",
			input: map[string]any{
				"message":       "Watch this",
				"media_type":    "video",
				"media_file_id": "file_vid456",
			},
			wantMessage:   "Watch this",
			wantMediaType: "video",
			wantMediaFile: "file_vid456",
		},
		{
			name: "text + image + button",
			input: map[string]any{
				"message":       "Ad with photo and button",
				"media_type":    "photo",
				"media_file_id": "file_photo789",
				"button": map[string]any{
					"text":  "Click me",
					"url":   "https://example.com",
					"style": "primary",
					"emoji": "👆",
				},
			},
			wantMessage:   "Ad with photo and button",
			wantMediaType: "photo",
			wantMediaFile: "file_photo789",
			wantButton:    &DealDetailsButton{Text: "Click me", URL: "https://example.com", Style: "primary", Emoji: "👆"},
		},
		{
			name: "text + video + button",
			input: map[string]any{
				"message":       "Ad with video and button",
				"media_type":    "video",
				"media_file_id": "file_vid999",
				"button": map[string]any{
					"text":  "Learn more",
					"url":   "https://example.com/promo",
					"style": "success",
					"emoji": "",
				},
			},
			wantMessage:   "Ad with video and button",
			wantMediaType: "video",
			wantMediaFile: "file_vid999",
			wantButton:    &DealDetailsButton{Text: "Learn more", URL: "https://example.com/promo", Style: "success", Emoji: ""},
		},
		{
			name: "image + button",
			input: map[string]any{
				"media_type":    "photo",
				"media_file_id": "file_photo_btn",
				"button": map[string]any{
					"text":  "Buy now",
					"url":   "https://shop.example.com",
					"style": "danger",
					"emoji": "🛒",
				},
			},
			wantMediaType: "photo",
			wantMediaFile: "file_photo_btn",
			wantButton:    &DealDetailsButton{Text: "Buy now", URL: "https://shop.example.com", Style: "danger", Emoji: "🛒"},
		},
		{
			name: "image only",
			input: map[string]any{
				"media_type":    "photo",
				"media_file_id": "file_photo_only",
			},
			wantMediaType: "photo",
			wantMediaFile: "file_photo_only",
		},
		{
			name: "video + button",
			input: map[string]any{
				"media_type":    "video",
				"media_file_id": "file_vid_btn",
				"button": map[string]any{
					"text":  "Watch trailer",
					"url":   "http://cdn.example.com/video",
					"style": "default",
					"emoji": "",
				},
			},
			wantMediaType: "video",
			wantMediaFile: "file_vid_btn",
			wantButton:    &DealDetailsButton{Text: "Watch trailer", URL: "http://cdn.example.com/video", Style: "default", Emoji: ""},
		},
		{
			name: "video only",
			input: map[string]any{
				"media_type":    "video",
				"media_file_id": "file_vid_only",
			},
			wantMediaType: "video",
			wantMediaFile: "file_vid_only",
		},
		{
			name: "animation only",
			input: map[string]any{
				"media_type":    "animation",
				"media_file_id": "file_anim_only",
			},
			wantMediaType: "animation",
			wantMediaFile: "file_anim_only",
		},
		{
			name: "text + animation",
			input: map[string]any{
				"message":       "Watch this GIF",
				"media_type":    "animation",
				"media_file_id": "file_anim_123",
			},
			wantMessage:   "Watch this GIF",
			wantMediaType: "animation",
			wantMediaFile: "file_anim_123",
		},
		{
			name: "animation + button",
			input: map[string]any{
				"media_type":    "animation",
				"media_file_id": "file_anim_btn",
				"button": map[string]any{
					"text":  "Click here",
					"url":   "https://example.com/anim",
					"style": "primary",
					"emoji": "",
				},
			},
			wantMediaType: "animation",
			wantMediaFile: "file_anim_btn",
			wantButton:    &DealDetailsButton{Text: "Click here", URL: "https://example.com/anim", Style: "primary", Emoji: ""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := buildDetails(tc.input)
			canonical, err := ValidateDealDetails(raw)
			if err != nil {
				t.Fatalf("ValidateDealDetails returned unexpected error: %v", err)
			}

			// Verify Get* helpers on canonical output.
			if got := GetMessageFromDetails(canonical); got != tc.wantMessage {
				t.Errorf("GetMessageFromDetails = %q, want %q", got, tc.wantMessage)
			}

			gotMediaType, gotMediaFile := GetMediaFromDetails(canonical)
			if gotMediaType != tc.wantMediaType {
				t.Errorf("GetMediaFromDetails mediaType = %q, want %q", gotMediaType, tc.wantMediaType)
			}
			if gotMediaFile != tc.wantMediaFile {
				t.Errorf("GetMediaFromDetails mediaFileID = %q, want %q", gotMediaFile, tc.wantMediaFile)
			}

			gotBtn := GetButtonFromDetails(canonical)
			if tc.wantButton == nil {
				if gotBtn != nil {
					t.Errorf("GetButtonFromDetails = %+v, want nil", gotBtn)
				}
			} else {
				if gotBtn == nil {
					t.Fatal("GetButtonFromDetails = nil, want non-nil button")
				}
				if gotBtn.Text != tc.wantButton.Text {
					t.Errorf("button.Text = %q, want %q", gotBtn.Text, tc.wantButton.Text)
				}
				if gotBtn.URL != tc.wantButton.URL {
					t.Errorf("button.URL = %q, want %q", gotBtn.URL, tc.wantButton.URL)
				}
				if gotBtn.Style != tc.wantButton.Style {
					t.Errorf("button.Style = %q, want %q", gotBtn.Style, tc.wantButton.Style)
				}
				if gotBtn.Emoji != tc.wantButton.Emoji {
					t.Errorf("button.Emoji = %q, want %q", gotBtn.Emoji, tc.wantButton.Emoji)
				}
			}
		})
	}
}

// --- ValidateDealDetails: invalid combinations ---

func TestValidateDealDetails_InvalidCombinations(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]any
	}{
		{
			name: "media_type without media_file_id",
			input: map[string]any{
				"message":    "text only but media_type set",
				"media_type": "photo",
				// media_file_id missing
			},
		},
		{
			name: "invalid media_type value",
			input: map[string]any{
				"media_type":    "gif",
				"media_file_id": "file_xyz",
			},
		},
		{
			name: "unknown top-level key",
			input: map[string]any{
				"message":     "hello",
				"unknown_key": "value",
			},
		},
		{
			name: "button with invalid style",
			input: map[string]any{
				"message": "text",
				"button": map[string]any{
					"text":  "Click",
					"url":   "https://example.com",
					"style": "purple", // not allowed
					"emoji": "",
				},
			},
		},
		{
			name: "button with non-http URL",
			input: map[string]any{
				"message": "text",
				"button": map[string]any{
					"text":  "Click",
					"url":   "ftp://example.com/file",
					"style": "default",
					"emoji": "",
				},
			},
		},
		{
			name: "button with invalid URL format",
			input: map[string]any{
				"message": "text",
				"button": map[string]any{
					"text":  "Click",
					"url":   "not-a-url",
					"style": "default",
					"emoji": "",
				},
			},
		},
		{
			name: "message field is not a string",
			input: map[string]any{
				"message": 12345,
			},
		},
		{
			name: "entities field is not an array",
			input: map[string]any{
				"message":  "hello",
				"entities": "not-an-array",
			},
		},
		{
			name: "text_link entity with empty url",
			input: map[string]any{
				"message": "hello",
				"entities": []any{
					map[string]any{
						"type": "text_link",
						"url":  "",
					},
				},
			},
		},
		{
			name: "text_link entity with non-http url",
			input: map[string]any{
				"message": "hello",
				"entities": []any{
					map[string]any{
						"type": "text_link",
						"url":  "ftp://bad-scheme.com",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := buildDetails(tc.input)
			_, err := ValidateDealDetails(raw)
			if err == nil {
				t.Errorf("ValidateDealDetails expected error for %q, got nil", tc.name)
			}
		})
	}
}

// --- ValidateDealDetails: edge cases ---

func TestValidateDealDetails_EdgeCases(t *testing.T) {
	t.Run("nil/empty input returns empty object", func(t *testing.T) {
		result, err := ValidateDealDetails(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result) != "{}" {
			t.Errorf("expected {}, got %s", result)
		}
	})

	t.Run("null JSON input returns empty object", func(t *testing.T) {
		result, err := ValidateDealDetails(json.RawMessage("null"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result) != "{}" {
			t.Errorf("expected {}, got %s", result)
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := ValidateDealDetails(json.RawMessage("{not valid json"))
		if err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})

	t.Run("button url can be empty string", func(t *testing.T) {
		raw := buildDetails(map[string]any{
			"message": "text",
			"button": map[string]any{
				"text":  "Click",
				"url":   "", // empty URL is allowed
				"style": "default",
				"emoji": "",
			},
		})
		_, err := ValidateDealDetails(raw)
		if err != nil {
			t.Errorf("unexpected error for button with empty url: %v", err)
		}
	})
}

// --- SetMessageInDetails ---

func TestSetMessageInDetails(t *testing.T) {
	t.Run("set message on empty details", func(t *testing.T) {
		result, err := SetMessageInDetails(json.RawMessage("{}"), "Hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := GetMessageFromDetails(result); got != "Hello" {
			t.Errorf("GetMessageFromDetails = %q, want %q", got, "Hello")
		}
	})

	t.Run("update existing message preserves other keys", func(t *testing.T) {
		initial := buildDetails(map[string]any{
			"message":       "old message",
			"media_type":    "photo",
			"media_file_id": "file_123",
		})
		result, err := SetMessageInDetails(initial, "new message")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := GetMessageFromDetails(result); got != "new message" {
			t.Errorf("GetMessageFromDetails = %q, want %q", got, "new message")
		}
		mt, mf := GetMediaFromDetails(result)
		if mt != "photo" || mf != "file_123" {
			t.Errorf("media lost after SetMessageInDetails: got (%q, %q)", mt, mf)
		}
	})

	t.Run("nil input treated as empty", func(t *testing.T) {
		result, err := SetMessageInDetails(nil, "Hello from nil")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := GetMessageFromDetails(result); got != "Hello from nil" {
			t.Errorf("GetMessageFromDetails = %q, want %q", got, "Hello from nil")
		}
	})
}

// --- SetMediaInDetails ---

func TestSetMediaInDetails(t *testing.T) {
	t.Run("set photo media on empty details", func(t *testing.T) {
		result, err := SetMediaInDetails(json.RawMessage("{}"), "photo", "file_abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mt, mf := GetMediaFromDetails(result)
		if mt != "photo" || mf != "file_abc" {
			t.Errorf("GetMediaFromDetails = (%q, %q), want (photo, file_abc)", mt, mf)
		}
	})

	t.Run("set video media preserves message", func(t *testing.T) {
		initial := buildDetails(map[string]any{"message": "keep me"})
		result, err := SetMediaInDetails(initial, "video", "file_vid")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mt, mf := GetMediaFromDetails(result)
		if mt != "video" || mf != "file_vid" {
			t.Errorf("unexpected media: (%q, %q)", mt, mf)
		}
		if got := GetMessageFromDetails(result); got != "keep me" {
			t.Errorf("message lost: got %q", got)
		}
	})

	t.Run("set animation media on empty details", func(t *testing.T) {
		result, err := SetMediaInDetails(json.RawMessage("{}"), "animation", "file_anim_xyz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mt, mf := GetMediaFromDetails(result)
		if mt != "animation" || mf != "file_anim_xyz" {
			t.Errorf("GetMediaFromDetails = (%q, %q), want (animation, file_anim_xyz)", mt, mf)
		}
	})

	t.Run("remove media with empty strings", func(t *testing.T) {
		initial := buildDetails(map[string]any{
			"message":       "text",
			"media_type":    "photo",
			"media_file_id": "file_abc",
		})
		result, err := SetMediaInDetails(initial, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mt, mf := GetMediaFromDetails(result)
		if mt != "" || mf != "" {
			t.Errorf("expected media removed, got (%q, %q)", mt, mf)
		}
	})
}

// TestSetAndGetVideoMedia verifies the full roundtrip of storing and retrieving video media
// via SetMediaInDetails and GetMediaFromDetails, matching the /edit command's code path.
func TestSetAndGetVideoMedia(t *testing.T) {
	t.Run("video file_id is stored and retrieved exactly", func(t *testing.T) {
		fileID := "BAACAgIAAxkBAAIBXWC3hKz9example_video_file_id"
		result, err := SetMediaInDetails(json.RawMessage("{}"), "video", fileID)
		if err != nil {
			t.Fatalf("SetMediaInDetails returned unexpected error: %v", err)
		}
		mt, mf := GetMediaFromDetails(result)
		if mt != "video" {
			t.Errorf("GetMediaFromDetails mediaType = %q, want %q", mt, "video")
		}
		if mf != fileID {
			t.Errorf("GetMediaFromDetails mediaFileID = %q, want %q", mf, fileID)
		}
	})

	t.Run("video media overwrites previous photo media", func(t *testing.T) {
		initial := buildDetails(map[string]any{
			"media_type":    "photo",
			"media_file_id": "photo_file_id",
		})
		videoFileID := "video_file_id_replacement"
		result, err := SetMediaInDetails(initial, "video", videoFileID)
		if err != nil {
			t.Fatalf("SetMediaInDetails returned unexpected error: %v", err)
		}
		mt, mf := GetMediaFromDetails(result)
		if mt != "video" {
			t.Errorf("GetMediaFromDetails mediaType = %q, want %q", mt, "video")
		}
		if mf != videoFileID {
			t.Errorf("GetMediaFromDetails mediaFileID = %q, want %q", mf, videoFileID)
		}
	})

	t.Run("video with empty file_id removes media keys (not stored)", func(t *testing.T) {
		// If detectMediaFromMessage returns mediaType="video" but mediaFileID="",
		// SetMediaInDetails should delete media keys rather than storing empty file_id.
		initial := buildDetails(map[string]any{"message": "text"})
		result, err := SetMediaInDetails(initial, "video", "")
		if err != nil {
			t.Fatalf("SetMediaInDetails returned unexpected error: %v", err)
		}
		mt, mf := GetMediaFromDetails(result)
		if mt != "" || mf != "" {
			t.Errorf("expected media removed when fileID empty, got (%q, %q)", mt, mf)
		}
		// message must still be preserved
		if got := GetMessageFromDetails(result); got != "text" {
			t.Errorf("message lost after SetMediaInDetails with empty fileID: got %q", got)
		}
	})

	t.Run("video media and message coexist after sequential Set calls", func(t *testing.T) {
		// Simulate the /edit flow: SetMessageInDetails then SetMediaInDetails
		details := json.RawMessage("{}")
		var err error

		details, err = SetMessageInDetails(details, "ad text")
		if err != nil {
			t.Fatalf("SetMessageInDetails error: %v", err)
		}

		videoFileID := "video_file_abc123"
		details, err = SetMediaInDetails(details, "video", videoFileID)
		if err != nil {
			t.Fatalf("SetMediaInDetails error: %v", err)
		}

		if got := GetMessageFromDetails(details); got != "ad text" {
			t.Errorf("GetMessageFromDetails = %q, want %q", got, "ad text")
		}
		mt, mf := GetMediaFromDetails(details)
		if mt != "video" {
			t.Errorf("GetMediaFromDetails mediaType = %q, want %q", mt, "video")
		}
		if mf != videoFileID {
			t.Errorf("GetMediaFromDetails mediaFileID = %q, want %q", mf, videoFileID)
		}
	})
}

// --- SetButtonInDetails ---

func TestSetButtonInDetails(t *testing.T) {
	t.Run("set button on empty details", func(t *testing.T) {
		btn := &DealDetailsButton{Text: "Go", URL: "https://go.dev", Style: "primary", Emoji: ""}
		result, err := SetButtonInDetails(json.RawMessage("{}"), btn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := GetButtonFromDetails(result)
		if got == nil {
			t.Fatal("expected button, got nil")
		}
		if got.Text != btn.Text || got.URL != btn.URL || got.Style != btn.Style {
			t.Errorf("button mismatch: got %+v, want %+v", got, btn)
		}
	})

	t.Run("remove button with nil", func(t *testing.T) {
		initial := buildDetails(map[string]any{
			"message": "text",
			"button": map[string]any{
				"text":  "Click",
				"url":   "https://example.com",
				"style": "danger",
				"emoji": "",
			},
		})
		result, err := SetButtonInDetails(initial, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := GetButtonFromDetails(result); got != nil {
			t.Errorf("expected nil button after removal, got %+v", got)
		}
	})

	t.Run("update button preserves media", func(t *testing.T) {
		initial := buildDetails(map[string]any{
			"media_type":    "video",
			"media_file_id": "file_v1",
		})
		btn := &DealDetailsButton{Text: "Buy", URL: "https://buy.example.com", Style: "success", Emoji: "💰"}
		result, err := SetButtonInDetails(initial, btn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mt, mf := GetMediaFromDetails(result)
		if mt != "video" || mf != "file_v1" {
			t.Errorf("media lost: got (%q, %q)", mt, mf)
		}
		got := GetButtonFromDetails(result)
		if got == nil || got.Style != "success" {
			t.Errorf("button mismatch after update: %+v", got)
		}
	})
}

// --- GetButtonFromDetails ---

func TestGetButtonFromDetails_NilAndEmpty(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		if got := GetButtonFromDetails(nil); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("null JSON returns nil", func(t *testing.T) {
		if got := GetButtonFromDetails(json.RawMessage("null")); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("empty object returns nil", func(t *testing.T) {
		if got := GetButtonFromDetails(json.RawMessage("{}")); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})
}

// --- GetMediaFromDetails ---

func TestGetMediaFromDetails_NilAndEmpty(t *testing.T) {
	t.Run("nil returns empty strings", func(t *testing.T) {
		mt, mf := GetMediaFromDetails(nil)
		if mt != "" || mf != "" {
			t.Errorf("expected empty, got (%q, %q)", mt, mf)
		}
	})

	t.Run("details without media returns empty strings", func(t *testing.T) {
		raw := buildDetails(map[string]any{"message": "hello"})
		mt, mf := GetMediaFromDetails(raw)
		if mt != "" || mf != "" {
			t.Errorf("expected empty, got (%q, %q)", mt, mf)
		}
	})
}

// --- GetMessageFromDetails ---

func TestGetMessageFromDetails_NilAndEmpty(t *testing.T) {
	t.Run("nil returns empty string", func(t *testing.T) {
		if got := GetMessageFromDetails(nil); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("null JSON returns empty string", func(t *testing.T) {
		if got := GetMessageFromDetails(json.RawMessage("null")); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("details without message returns empty", func(t *testing.T) {
		raw := buildDetails(map[string]any{
			"media_type":    "photo",
			"media_file_id": "file_x",
		})
		if got := GetMessageFromDetails(raw); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}

// --- Entities validation ---

func TestValidateDealDetails_EntitiesValid(t *testing.T) {
	t.Run("valid text_link entity", func(t *testing.T) {
		raw := buildDetails(map[string]any{
			"message": "click here",
			"entities": []any{
				map[string]any{
					"type":   "text_link",
					"url":    "https://example.com",
					"offset": 6,
					"length": 4,
				},
			},
		})
		canonical, err := ValidateDealDetails(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		entitiesRaw := GetRawEntitiesFromDetails(canonical)
		if entitiesRaw == nil {
			t.Error("expected entities in canonical output, got nil")
		}
	})

	t.Run("non-text_link entity type allowed without url", func(t *testing.T) {
		raw := buildDetails(map[string]any{
			"message": "bold text",
			"entities": []any{
				map[string]any{
					"type":   "bold",
					"offset": 0,
					"length": 4,
				},
			},
		})
		_, err := ValidateDealDetails(raw)
		if err != nil {
			t.Errorf("unexpected error for bold entity: %v", err)
		}
	})

	t.Run("caption_entities preserved", func(t *testing.T) {
		raw := buildDetails(map[string]any{
			"media_type":    "photo",
			"media_file_id": "file_photo",
			"caption_entities": []any{
				map[string]any{
					"type":   "bold",
					"offset": 0,
					"length": 5,
				},
			},
		})
		canonical, err := ValidateDealDetails(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ceRaw := GetRawCaptionEntitiesFromDetails(canonical)
		if ceRaw == nil {
			t.Error("expected caption_entities in canonical output, got nil")
		}
	})
}
