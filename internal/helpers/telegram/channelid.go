package telegram

// ToBotAPIChannelID converts a bare MTProto channel ID to Bot API format.
// e.g., 1234567890 → -1001234567890
// See https://core.telegram.org/api/bots/ids
func ToBotAPIChannelID(mtprotoID int64) int64 {
	return -1000000000000 - mtprotoID
}

// ToMTProtoChannelID converts a Bot API channel ID to bare MTProto format.
// e.g., -1001234567890 → 1234567890
// See https://core.telegram.org/api/bots/ids
func ToMTProtoChannelID(botAPIID int64) int64 {
	return -botAPIID - 1000000000000
}
