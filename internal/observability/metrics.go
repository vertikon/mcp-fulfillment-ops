// Package observability provides OpenTelemetry tracing and Prometheus metrics
package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics provides Prometheus metrics
type Metrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	activeWorkers   prometheus.Gauge
	queueSize       prometheus.Gauge
	cacheHits       *prometheus.CounterVec
	cacheMisses     *prometheus.CounterVec
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	m := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		activeWorkers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "worker_pool_active_workers",
				Help: "Number of active workers",
			},
		),
		queueSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "worker_pool_queue_size",
				Help: "Current queue size",
			},
		),
		cacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total cache hits",
			},
			[]string{"level"},
		),
		cacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total cache misses",
			},
			[]string{"level"},
		),
	}

	prometheus.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.activeWorkers,
		m.queueSize,
		m.cacheHits,
		m.cacheMisses,
	)

	return m
}

// HTTPHandler returns the Prometheus metrics HTTP handler
func (m *Metrics) HTTPHandler() http.Handler {
	return promhttp.Handler()
}

// RecordRequest records an HTTP request
func (m *Metrics) RecordRequest(method, endpoint, status string, duration float64) {
	m.requestsTotal.WithLabelValues(method, endpoint, status).Inc()
	m.requestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// SetActiveWorkers sets the number of active workers
func (m *Metrics) SetActiveWorkers(count float64) {
	m.activeWorkers.Set(count)
}

// SetQueueSize sets the queue size
func (m *Metrics) SetQueueSize(size float64) {
	m.queueSize.Set(size)
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(level string) {
	m.cacheHits.WithLabelValues(level).Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(level string) {
	m.cacheMisses.WithLabelValues(level).Inc()
}
