package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus data collector definitions

var promRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "ads_mrkt_server_http_requests_total",
		Help: "Total number of requests received per endpoint",
	},
	[]string{"resource"},
)

var promErrorsTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "ads_mrkt_server_http_errors_total",
		Help: "Total number of requests received per endpoint",
	},
)

var promNumActiveRequests = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "ads_mrkt_server_http_num_active_requests",
		Help: "Number of active requests that have yet to receive a response",
	},
)

var promRequestLatency = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "ads_mrkt_server_http_request_latency",
		Help: "Latency of processed requests for a given endpoint",
	},
	[]string{"resource"},
)

// Prometheus HTTP requests middleware
func WithMetrics(next http.HandlerFunc, prefix string) http.HandlerFunc {
	return func(writer http.ResponseWriter, reader *http.Request) {
		// Increment the active requests counter
		promNumActiveRequests.Inc()

		// Record start time
		start := time.Now()

		promWriter := NewPromWriter(writer)

		// Call the target handler
		next(promWriter, reader)

		// Calculate request duration and observe the result
		duration := time.Since(start).Seconds()

		if promWriter.StatusCode != http.StatusOK {
			promErrorsTotal.Inc()
		}

		// Having only the first-level resource is enough
		resource := GetFirstURLChunk(reader.URL.Path, prefix)

		// Observe request duration
		promRequestLatency.WithLabelValues(resource).Observe(duration)

		// Count the request
		promRequestsTotal.WithLabelValues(resource).Inc()

		// Decrement the active requests counter
		promNumActiveRequests.Dec()
	}
}

// Wrapper around a `http.ResponseWriter` that allows checking the status code
// of a response
type PromWriter struct {
	http.ResponseWriter
	StatusCode int
}

// Wrap http.ResponseWriter and set initial `StatusCode` value to 200 as per
// documentation:
//
// "If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes or 1xx informational responses."
func NewPromWriter(writer http.ResponseWriter) *PromWriter {
	return &PromWriter{writer, http.StatusOK}
}

// Intercept the `statusCode` value and forward the call to the parent
func (w *PromWriter) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func GetFirstURLChunk(url string, prefix string) string {
	url = strings.TrimPrefix(url, prefix)
	index := strings.Index(url[1:], "/")

	if index < 0 {
		return url
	}

	return url[:index+1]
}
