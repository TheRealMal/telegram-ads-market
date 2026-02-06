package postgres

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus data collector definitions

var promDatabaseQueriesTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "not_platform_db_queries_total",
		Help: "Total number of database queries per component",
	},
	[]string{"method"},
)

var promDatabaseErrorsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "not_platform_db_errors_total",
		Help: "Total number of database errors per component",
	},
	[]string{"method"},
)

var promDatabaseQueryLatency = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "not_platform_db_query_latency",
		Help: "Latency of database queries per component",
	},
	[]string{"method"},
)
