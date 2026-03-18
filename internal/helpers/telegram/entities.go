package telegram

// AdjustEntitiesOffset adjusts entity offsets after stripping a prefix of the
// given UTF-16 code unit length from the message text.
// Entities entirely within the prefix are dropped.
// Entities partially overlapping the prefix are truncated.
func AdjustEntitiesOffset(entities []MessageEntity, prefixUTF16Len int) []MessageEntity {
	if len(entities) == 0 {
		return nil
	}
	result := make([]MessageEntity, 0, len(entities))
	for _, e := range entities {
		end := e.Offset + e.Length
		if end <= prefixUTF16Len {
			continue // entirely within the prefix -- drop
		}
		adjusted := e // copy
		if adjusted.Offset < prefixUTF16Len {
			// partially within prefix -- truncate
			overlap := prefixUTF16Len - adjusted.Offset
			adjusted.Offset = 0
			adjusted.Length -= overlap
		} else {
			adjusted.Offset -= prefixUTF16Len
		}
		result = append(result, adjusted)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
