package telegram

import (
	"context"
	"log/slog"
	"time"

	"ads-mrkt/internal/helpers/ratelimiter"
)

// apiCall represents a single Telegram Bot API call to be executed through the pipeline.
type apiCall struct {
	name     string
	fn       func() ([]byte, error)
	resultCh chan callResult // nil for fire-and-forget calls
}

type callResult struct {
	body []byte
	err  error
}

// pipeline serializes all Telegram Bot API calls through a single rate-limited queue.
type pipeline struct {
	queue       chan apiCall
	rateLimiter *ratelimiter.RateLimiter
}

const (
	pipelineQueueSize     = 500
	pipelineRateLimitWait = 50 * time.Millisecond
)

func newPipeline(ctx context.Context, rl *ratelimiter.RateLimiter) *pipeline {
	p := &pipeline{
		queue:       make(chan apiCall, pipelineQueueSize),
		rateLimiter: rl,
	}
	go p.worker(ctx)
	return p
}

func (p *pipeline) worker(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("pipeline worker panicked", "panic", r)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			// Drain remaining calls with context-canceled errors
			for {
				select {
				case call := <-p.queue:
					if call.resultCh != nil {
						call.resultCh <- callResult{err: ctx.Err()}
					}
				default:
					return
				}
			}
		case call := <-p.queue:
			p.waitForRateLimit(ctx)
			body, err := call.fn()
			if call.resultCh != nil {
				call.resultCh <- callResult{body: body, err: err}
			} else if err != nil {
				slog.Error("pipeline fire-and-forget call failed", "method", call.name, "error", err)
			}
		}
	}
}

func (p *pipeline) waitForRateLimit(ctx context.Context) {
	for {
		allow, err := p.rateLimiter.CheckLimits(ctx)
		if err != nil {
			// Proceed anyway on rate limiter errors to avoid blocking indefinitely
			slog.Error("pipeline rate limit check failed", "error", err)
			return
		}
		if allow {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(pipelineRateLimitWait):
			continue
		}
	}
}

// submit enqueues a synchronous API call and waits for the result.
func (p *pipeline) submit(ctx context.Context, name string, fn func() ([]byte, error)) ([]byte, error) {
	resultCh := make(chan callResult, 1)
	call := apiCall{name: name, fn: fn, resultCh: resultCh}
	select {
	case p.queue <- call:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case result := <-resultCh:
		return result.body, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
