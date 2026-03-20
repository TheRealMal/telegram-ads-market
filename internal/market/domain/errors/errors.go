package errors

import (
	"errors"
	"time"
)

var (
	ErrNotFound                   = errors.New("market: not found")
	ErrNotChannelAdmin            = errors.New("market: user is not admin of the channel")
	ErrChannelStatsDenied         = errors.New("market: channel stats only for admins or users who listed this channel")
	ErrDealNotDraft               = errors.New("market: deal is not in draft status")
	ErrUnauthorizedSide           = errors.New("market: user is not lessor or lessee of this deal")
	ErrWalletNotSet               = errors.New("market: connect wallet before signing")
	ErrPayoutNotSet               = errors.New("market: both parties must set payout address before signing")
	ErrDealDetailsMessageRequired = errors.New("market: deal details message must be set before signing")
	ErrDealDetailsMediaRequired   = errors.New("market: deal details media must be set before signing story deal")
	ErrWalletRequired             = errors.New("market: connect wallet before proceeding")
	ErrPriceMismatch              = errors.New("market: type, duration and price must match one of the listing's price options")
	ErrOwnListing                 = errors.New("market: cannot create deal on your own listing")
	ErrInstantPostWalletRequired  = errors.New("market: all parties must have a wallet connected for instant_post deals")
	ErrInvalidListingType         = errors.New("market: invalid listing type")
	ErrInvalidInstantPost         = errors.New("market: prepared_post with non-empty message is required for instant_post listings")
	ErrInstantPostLesseeOnly      = errors.New("market: instant_post ad type is only allowed for lessee listings")
	ErrInvalidDealDetails         = errors.New("market: invalid deal details")
	ErrInvalidInput               = errors.New("market: invalid input")
)

// ErrStatsRefreshTooSoon is returned when channel stats refresh is requested within the cooldown period.
type ErrStatsRefreshTooSoon struct {
	NextAvailableAt time.Time
}

func (e *ErrStatsRefreshTooSoon) Error() string {
	return "market: channel stats update requested too recently"
}
