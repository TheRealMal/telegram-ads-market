package telegram

import "testing"

func TestToBotAPIChannelID(t *testing.T) {
	tests := []struct {
		mtproto int64
		botAPI  int64
	}{
		{1234567890, -1001234567890},
		{1, -1000000000001},
		{2345678901, -1002345678901},
	}
	for _, tt := range tests {
		got := ToBotAPIChannelID(tt.mtproto)
		if got != tt.botAPI {
			t.Errorf("ToBotAPIChannelID(%d) = %d, want %d", tt.mtproto, got, tt.botAPI)
		}
	}
}

func TestToMTProtoChannelID(t *testing.T) {
	tests := []struct {
		botAPI  int64
		mtproto int64
	}{
		{-1001234567890, 1234567890},
		{-1000000000001, 1},
		{-1002345678901, 2345678901},
	}
	for _, tt := range tests {
		got := ToMTProtoChannelID(tt.botAPI)
		if got != tt.mtproto {
			t.Errorf("ToMTProtoChannelID(%d) = %d, want %d", tt.botAPI, got, tt.mtproto)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	ids := []int64{1, 1234567890, 2345678901, 9999999999}
	for _, id := range ids {
		got := ToMTProtoChannelID(ToBotAPIChannelID(id))
		if got != id {
			t.Errorf("round-trip failed for %d: got %d", id, got)
		}
	}
}
