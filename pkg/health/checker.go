package health

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"ads-mrkt/pkg/health/config"
)

type CheckResult struct {
	Timestamp time.Time `json:"timestamp"`
	Result    bool      `json:"result"`
}

type HealthChecks []CheckResult

type HealthStatus struct {
	Healthy bool         `json:"healthy"`
	Checks  HealthChecks `json:"checks"`
}

type pinger interface {
	Ping(ctx context.Context) error
}

type Checker struct {
	config  config.Config
	log     *slog.Logger
	pingers []pinger
	mx      sync.Mutex
	checks  HealthChecks
}

func NewChecker(config config.Config, pingers ...pinger) *Checker {
	checker := Checker{
		config:  config,
		pingers: pingers,
		log:     slog.With("component", "health", "instance", config.InstanceID),
		mx:      sync.Mutex{},
		checks: HealthChecks{
			// if this code gets executed, we assume that there was an initial
			// check
			CheckResult{Timestamp: time.Now(), Result: true},
		},
	}

	go checker.run(context.Background())

	return &checker
}

func (c *Checker) run(ctx context.Context) {
	c.log.Debug("Starting the health checker...")

	ticker := time.NewTicker(c.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Debug("Stopping health checker ...")
			return
		case <-ticker.C:
			result := make([]CheckResult, 0, len(c.pingers))

			for _, pinger := range c.pingers {
				checkCtx, cancelCheck := context.WithTimeout(ctx, c.config.PingTimeout)

				err := pinger.Ping(checkCtx)
				if err != nil {
					c.log.Error("Health check failed", "error", err)
				}

				result = append(result, CheckResult{
					Timestamp: time.Now(),
					Result:    err == nil,
				})

				cancelCheck()
			}

			c.mx.Lock()
			c.checks = result
			c.mx.Unlock()
		}
	}
}

func (c *Checker) GetHealthStatus() HealthStatus {
	healthy := true

	c.mx.Lock()
	defer c.mx.Unlock()

	for component, check := range c.checks {
		if !check.Result {
			healthy = false
			c.log.Error("Component health check failed", "component", component)
		}
	}

	return HealthStatus{
		Healthy: healthy,
		Checks:  c.checks,
	}
}
