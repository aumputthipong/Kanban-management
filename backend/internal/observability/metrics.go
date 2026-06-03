// Prometheus metrics. Exposed at /metrics; the recommended scrape interval
// is 15s. Tracks HTTP latency, in-flight requests, and pgx pool gauges so
// p95 latency and "is the pool saturated?" can be answered without a tracing
// vendor.
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

// httpDuration is labelled by method + route pattern (e.g. /api/boards/{boardID})
// + status code class. Using the route pattern instead of the raw path keeps
// cardinality bounded — otherwise every board UUID becomes its own series.
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

// Registry is the per-process Prometheus registry. Kept separate from the
// default registry so tests can build a clean one and so accidental global
// registration in third-party libs doesn't pollute /metrics.
var Registry = prometheus.NewRegistry()

func init() {
	Registry.MustRegister(
		httpDuration,
		httpInFlight,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
}

// RegisterDBPool exposes pgxpool stats as gauges. Call once after the pool
// is created. Re-registration with the same labels would panic, so the caller
// must call this exactly once per pool.
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

// MetricsHandler returns the http.Handler that serves /metrics. Uses the
// package-local Registry so go/process/pgx collectors all land in one scrape.
func MetricsHandler() http.Handler {
	return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{Registry: Registry})
}

// HTTPMetrics is a chi middleware that observes request latency. It must be
// mounted AFTER chi's RouteContext is populated (i.e. after r.Use(...) but
// before route-specific groups) so that chi.RouteContext(r.Context())
// returns the matched pattern.
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

// Unwrap lets http.ResponseController (and other wrappers) reach the
// underlying writer — needed so optional interfaces like http.Hijacker and
// http.Flusher aren't lost behind this recorder.
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

// Hijack delegates to the underlying writer so WebSocket upgrades work through
// this middleware. Without it, gorilla's upgrade fails (the recorder hides the
// connection's Hijacker), 500-ing the /ws handshake.
func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(s.ResponseWriter).Hijack()
}
