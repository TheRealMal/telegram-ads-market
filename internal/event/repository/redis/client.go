package redis

import (
	"context"

	redisclient "github.com/redis/go-redis/v9"

	"ads-mrkt/internal/redis"
)

type db interface {
	XAdd(ctx context.Context, args *redisclient.XAddArgs) *redisclient.StringCmd
	XReadGroup(ctx context.Context, args *redisclient.XReadGroupArgs) *redisclient.XStreamSliceCmd
	XGroupCreateMkStream(ctx context.Context, stream, group, id string) *redisclient.StatusCmd
	XAck(ctx context.Context, stream, group string, ids []string) *redisclient.IntCmd
	XClaim(ctx context.Context, args *redisclient.XClaimArgs) *redisclient.XMessageSliceCmd
	XPendingExt(ctx context.Context, args *redisclient.XPendingExtArgs) *redisclient.XPendingExtCmd
	XPendingAutoClaim(ctx context.Context, args *redisclient.XAutoClaimArgs) *redisclient.XAutoClaimCmd
	XGroupDelConsumer(ctx context.Context, stream, group, consumer string) (int64, error)
	XTrim(ctx context.Context, stream, minId string, limit int64) *redisclient.IntCmd
}

type repository struct {
	db db
}

// New returns an event repository that uses the given redis client.
func New(client *redis.Client) *repository {
	return &repository{
		db: client,
	}
}
