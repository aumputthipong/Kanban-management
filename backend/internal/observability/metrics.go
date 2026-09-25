// Package observability: Prometheus metrics at /metrics and Sentry wiring.
package observability

import (
	"bufio"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Labelled by route pattern, not raw path, to keep cardinality bounded.
var (
	httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency, partitioned by method/route/status.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
		[]string{"method", "route", "status"},
	)
	httpInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Number of HTTP requests currently being served.",
	})
)

var Registry = prometheus.NewRegistry()

func init() {
	Registry.MustRegister(
		httpDuration,
		httpInFlight,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
}

// Call exactly once per pool — re-registration panics.
func RegisterDBPool(pool *pgxpool.Pool) {
	Registry.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{Name: "pgx_pool_total_conns", Help: "Total connections in the pgx pool."},
		func() float64 { return float64(pool.Stat().TotalConns()) },
	))
	Registry.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{Name: "pgx_pool_idle_conns", Help: "Idle connections in the pgx pool."},
		func() float64 { return float64(pool.Stat().IdleConns()) },
	))
	Registry.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{Name: "pgx_pool_acquired_conns", Help: "Acquired connections in the pgx pool."},
		func() float64 { return float64(pool.Stat().AcquiredConns()) },
	))
}

func MetricsHandler() http.Handler {
	return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{Registry: Registry})
}

// Mount after chi's RouteContext is populated so the route pattern resolves.
func HTTPMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpInFlight.Inc()
		defer httpInFlight.Dec()

		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rw, r)

		route := "unmatched"
		if ctx := chi.RouteContext(r.Context()); ctx != nil && ctx.RoutePattern() != "" {
			route = ctx.RoutePattern()
		}
		httpDuration.WithLabelValues(r.Method, route, strconv.Itoa(rw.status)).
			Observe(time.Since(start).Seconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status = code
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(code)
}

// Keeps optional interfaces (Hijacker, Flusher) reachable.
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

// Without this the WS upgrade fails with a 500.
func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(s.ResponseWriter).Hijack()
}
