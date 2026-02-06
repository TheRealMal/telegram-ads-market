package builder

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type shutdownFn func(context.Context) error

type Shutdown struct {
	fn []shutdownFn

	once      sync.Once
	isStopped chan struct{}
}

func NewShutdown(fn ...shutdownFn) *Shutdown {
	return &Shutdown{
		isStopped: make(chan struct{}),
	}
}

func (s *Shutdown) Add(fn shutdownFn) {
	s.fn = append(s.fn, fn)
}

func (s *Shutdown) WaitShutdown(ctx context.Context) {
	stopSignals := []os.Signal{syscall.SIGTERM, syscall.SIGINT}
	signals := make(chan os.Signal, len(stopSignals))

	signal.Notify(signals, stopSignals...)
	select {
	case <-s.isStopped:
		return
	case sig := <-signals:
		slog.InfoContext(ctx, fmt.Sprintf("got %s os signal. application will be stopped", sig))

	}

	s.do(ctx)
}

func (s *Shutdown) MustShutdown(ctx context.Context, stopperName string, err error) {
	slog.ErrorContext(ctx, fmt.Sprintf("got error from %s. application will be stopped. %s", stopperName, err))

	s.do(ctx)
}

func (s *Shutdown) do(ctx context.Context) {
	s.once.Do(func() {
		for i := len(s.fn) - 1; i >= 0; i-- {
			if err := s.fn[i](ctx); err != nil {
				slog.ErrorContext(ctx, "got error while shutdown", "error", err)
			}
		}

		close(s.isStopped)
	})

	<-s.isStopped
}
