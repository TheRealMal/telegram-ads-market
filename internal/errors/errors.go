package errors

type ErrorCode string

const (
	ErrorCodeTooManyRequests     ErrorCode = "too_many_requests"
	ErrorCodeInitDataRequired    ErrorCode = "init_data_required"
	ErrorCodeUnauthorized        ErrorCode = "unauthorized"
	ErrorCodeNotFound            ErrorCode = "not_found"
	ErrorCodeBadRequest          ErrorCode = "bad_request"
	ErrorCodeInternalServerError ErrorCode = "internal_server_error"
	ErrorCodeForbidden           ErrorCode = "forbidden"
)

type ServiceError struct {
	Err     error
	Message string
	Code    ErrorCode
	// Data is optional payload for error responses (e.g. next_available_at for rate limit).
	Data interface{}
}

func (se ServiceError) Error() string {
	return se.Message
}

func (se ServiceError) Unwrap() error {
	return se.Err
}
