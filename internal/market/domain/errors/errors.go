package errors

import (
	"errors"
	"time"
)

var (
	ErrNotFound           = errors.New("market: not found")
	ErrNotChannelAdmin    = errors.New("market: user is not admin of the channel")
	ErrChannelStatsDenied = errors.New("market: channel stats only for admins or users who listed this channel")
	ErrDealNotDraft       = errors.New("market: deal is not in draft status")
	ErrUnauthorizedSide   = errors.New("market: user is not lessor or lessee of this deal")
	ErrWalletNotSet       = errors.New("market: connect wallet before signing")
	ErrPayoutNotSet       = errors.New("market: both parties must set payout address before signing")
)

// ErrStatsRefreshTooSoon is returned when channel stats refresh is requested within the cooldown period.
type ErrStatsRefreshTooSoon struct {
	NextAvailableAt time.Time
}

func (e *ErrStatsRefreshTooSoon) Error() string {
	return "market: channel stats update requested too recently"
}
