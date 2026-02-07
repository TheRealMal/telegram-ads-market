package telegram

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrNotFound      = errors.New("NOT_FOUND")
	ErrUserForbidden = errors.New("USER_FORBIDDEN")
)

type RetryAfterError struct {
	RetryAfter int
	Msg        string
}

func (e *RetryAfterError) Error() string {
	return fmt.Sprintf("%s (retry after: %d)", e.Msg, e.RetryAfter)
}

type TelegramErrorResponse struct {
	OK          bool           `json:"ok"`
	ErrorCode   int            `json:"error_code"`
	Description string         `json:"description"`
	Parameters  map[string]int `json:"parameters,omitempty"`
}

// Map the error response text to a distinct `error` value that can be acted
// upon by the calling code
func (r *TelegramErrorResponse) GetMappedError() error {
	if r.ErrorCode == http.StatusTooManyRequests {
		retryAfter, ok := r.Parameters["retry_after"]
		if !ok {
			retryAfter = 30 //nolint:revive
		}
		return &RetryAfterError{RetryAfter: retryAfter, Msg: "rate limited"}
	}
	if r.ErrorCode == http.StatusForbidden {
		return fmt.Errorf("%w: %s", ErrUserForbidden, r.Description)
	}
	switch r.Description {
	case "Not Found",
		"[Error]: Bad Request: user not found",
		"Bad Request: chat not found":
		return ErrNotFound
	default:
		return nil
	}
}
