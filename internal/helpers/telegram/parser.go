package telegram

import (
	"encoding/json"
	"fmt"
)

func ParseUpdateData(rawData []byte) (*Update, error) {
	var data Update

	rawData = removeNewLines(rawData)

	err := json.Unmarshal(rawData, &data)
	if err != nil {
		return nil, fmt.Errorf("update unmarshalling error: %w", err)
	}

	return &data, nil
}

// removeNewLines removes new line characters from the slice of bytes
func removeNewLines(input []byte) []byte {
	// Pre-allocate a slice with the same capacity as input to avoid allocations
	output := make([]byte, 0, len(input))

	i := 0
	for i < len(input) {
		// Check if the current byte and the next byte form the substring "\n"
		if i < len(input)-1 && input[i] == '\\' && input[i+1] == 'n' {
			// Skip the "\n" substring
			i += 2
		} else {
			// Otherwise, append the current byte to the output slice
			output = append(output, input[i])
			i++
		}
	}

	return output
}
