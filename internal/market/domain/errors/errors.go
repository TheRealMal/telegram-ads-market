package errors

import "errors"

var (
	ErrNotFound            = errors.New("market: not found")
	ErrNotChannelAdmin     = errors.New("market: user is not admin of the channel")
	ErrChannelStatsDenied  = errors.New("market: channel stats only for admins or users who listed this channel")
	ErrDealNotDraft        = errors.New("market: deal is not in draft status")
	ErrUnauthorizedSide    = errors.New("market: user is not lessor or lessee of this deal")
)
