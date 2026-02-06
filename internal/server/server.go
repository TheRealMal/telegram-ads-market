package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"ads-mrkt/internal/server/config"
	"ads-mrkt/pkg/health"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

// APIHandler is a custom handler type that returns data or an error
type APIHandler func(w http.ResponseWriter, r *http.Request) (interface{}, error)

type Server struct {
	Config        config.Config
	httpServer    *http.Server
	healthChecker *health.Checker
}

func NewServer(config config.Config, healthChecker *health.Checker) *Server {
	return &Server{
		Config: config,

		httpServer: &http.Server{
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			IdleTimeout:  config.IdleTimeout,
		},
		healthChecker: healthChecker,
	}
}

func (s *Server) StartProbesAndMetrics() {
	// Expose Prometheus metrics
	go func() {
		handler := http.NewServeMux()

		handler.Handle("/metrics", promhttp.Handler())
		slog.Info("Serving metrics", "port", s.Config.MetricsPort)

		addr := fmt.Sprintf(":%d", s.Config.MetricsPort)
		slog.Error("Prometheus HTTP listener failed", "error",
			http.ListenAndServe(addr, handler)) //nolint:gosec
	}()

	// Expose swagger
	go func() {
		handler := http.NewServeMux()

		handler.Handle("/swagger", httpSwagger.WrapHandler)
		slog.Info("Serving swagger", "port", s.Config.SwaggerPort)

		addr := fmt.Sprintf(":%d", s.Config.SwaggerPort)
		slog.Error("Swagger HTTP listener failed", "error",
			http.ListenAndServe(addr, handler)) //nolint:gosec
	}()

	// Expose health probes
	go func() {
		handler := http.NewServeMux()

		handler.Handle("/health", WithMetrics(
			WithMethod(
				WithJSONResponse(s.HealthHandler),
				http.MethodGet,
			),
			"",
		))
		slog.Info("Serving health probes", "port", s.Config.ProbesPort)

		addr := fmt.Sprintf(":%d", s.Config.ProbesPort)
		slog.Error("Health checks HTTP listener failed", "error",
			http.ListenAndServe(addr, handler)) //nolint:gosec
	}()
}

func (s *Server) Start(ctx context.Context, mux http.Handler) {
	s.StartProbesAndMetrics()

	s.httpServer.Handler = http.TimeoutHandler(mux, s.Config.WriteTimeout, "Timeout")

	s.run(ctx)

	slog.Info("Shutting down server...")
}

func (s *Server) Stop(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exiting")
}

func (s *Server) run(ctx context.Context) {
	slog.Info("Starting server", "port", s.Config.ListenPort)

	// Use ListenConfig to create a listener with context support
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("%s:%d", s.Config.ListenAddr, s.Config.ListenPort))
	if err != nil {
		slog.Error("Error creating listener", "error", err)
	}
	defer listener.Close()

	if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		slog.Error("Could not start server", "error", err.Error())
	}
}
