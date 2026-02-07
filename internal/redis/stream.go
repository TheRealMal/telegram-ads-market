package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func (c *Client) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	return c.client.XAdd(ctx, args)
}

func (c *Client) XReadGroup(ctx context.Context, args *redis.XReadGroupArgs) *redis.XStreamSliceCmd {
	return c.client.XReadGroup(ctx, args)
}

func (c *Client) XGroupCreateMkStream(ctx context.Context, stream, group, id string) *redis.StatusCmd {
	return c.client.XGroupCreateMkStream(ctx, stream, group, id)
}

func (c *Client) XAck(ctx context.Context, stream, group string, ids []string) *redis.IntCmd {
	return c.client.XAck(ctx, stream, group, ids...)
}

func (c *Client) XClaim(ctx context.Context, args *redis.XClaimArgs) *redis.XMessageSliceCmd {
	return c.client.XClaim(ctx, args)
}

func (c *Client) XPendingExt(ctx context.Context, args *redis.XPendingExtArgs) *redis.XPendingExtCmd {
	return c.client.XPendingExt(ctx, args)
}

func (c *Client) XPendingAutoClaim(ctx context.Context, args *redis.XAutoClaimArgs) *redis.XAutoClaimCmd {
	return c.client.XAutoClaim(ctx, args)
}

func (c *Client) XGroupDelConsumer(ctx context.Context, stream, group, consumer string) (int64, error) {
	return c.client.XGroupDelConsumer(ctx, stream, group, consumer).Result()
}

func (c *Client) XTrim(ctx context.Context, stream, minId string, limit int64) *redis.IntCmd {
	return c.client.XTrimMinIDApprox(ctx, stream, minId, limit)
}
