package http

import (
	"net/http"
	"strconv"

	apperrors "ads-mrkt/internal/errors"
	"ads-mrkt/pkg/auth"
)

func requireUserID(r *http.Request) (int64, error) {
	userID, ok := auth.GetTelegramID(r.Context())
	if !ok {
		return 0, apperrors.ServiceError{Err: nil, Message: "unauthorized", Code: apperrors.ErrorCodeUnauthorized}
	}
	return userID, nil
}

func parsePathID(r *http.Request, paramName string) (int64, error) {
	s := r.PathValue(paramName)
	if s == "" {
		return 0, apperrors.ServiceError{Err: nil, Message: paramName + " required", Code: apperrors.ErrorCodeBadRequest}
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, apperrors.ServiceError{Err: err, Message: "invalid " + paramName, Code: apperrors.ErrorCodeBadRequest}
	}
	return id, nil
}
