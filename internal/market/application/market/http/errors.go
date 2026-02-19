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
	case errors.Is(err, marketerrors.ErrDealNotDraft), errors.Is(err, marketerrors.ErrWalletNotSet), errors.Is(err, marketerrors.ErrPayoutNotSet):
		return apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeBadRequest}
	case errors.Is(err, deal_chat.ErrForumNotConfigured):
		return apperrors.ServiceError{Err: err, Message: "deal chat forum not configured", Code: apperrors.ErrorCodeInternalServerError}
	default:
		return apperrors.ServiceError{Err: err, Message: err.Error(), Code: apperrors.ErrorCodeInternalServerError}
	}
}
