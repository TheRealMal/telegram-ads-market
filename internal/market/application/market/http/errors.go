package http

import (
	"errors"

	apperrors "ads-mrkt/internal/errors"
	marketerrors "ads-mrkt/internal/market/domain/errors"
	"ads-mrkt/internal/market/service/deal_chat"
)

func toServiceError(err error) apperrors.ServiceError {
	if err == nil {
		return apperrors.ServiceError{}
	}
	switch {
	case errors.Is(err, marketerrors.ErrNotFound):
		return apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeNotFound}
	case errors.Is(err, marketerrors.ErrNotChannelAdmin), errors.Is(err, marketerrors.ErrUnauthorizedSide), errors.Is(err, marketerrors.ErrChannelStatsDenied):
		return apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeForbidden}
	case errors.Is(err, marketerrors.ErrDealNotDraft):
		return apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeBadRequest}
	case errors.Is(err, deal_chat.ErrTelegramSenderNil):
		return apperrors.ServiceError{Err: err, Message: "telegram not configured", Code: apperrors.ErrorCodeInternalServerError}
	case errors.Is(err, deal_chat.ErrActiveDealChatExists):
		return apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeBadRequest}
	default:
		return apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeInternalServerError}
	}
}
