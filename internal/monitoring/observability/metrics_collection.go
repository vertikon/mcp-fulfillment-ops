// Package observability provides metrics collection capabilities
package observability

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// MetricType represents type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric represents a collected metric
type Metric struct {
	Name        string                 `json:"name"`
	Type        MetricType             `json:"type"`
	Value       float64                `json:"value"`
	Labels      map[string]string      `json:"labels"`
	Timestamp   time.Time              `json:"timestamp"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MetricsCollector collects and manages metrics
type MetricsCollector struct {
	mu            sync.RWMutex
	metrics       map[string]*Metric
	registry      *prometheus.Registry
	counters      map[string]*prometheus.CounterVec
	gauges        map[string]*prometheus.GaugeVec
	histograms    map[string]*prometheus.HistogramVec
	summaries     map[string]*prometheus.SummaryVec
	collectInterval time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
}

// MetricsConfig represents metrics collection configuration
type MetricsConfig struct {
	CollectInterval time.Duration `json:"collect_interval"`
	EnablePrometheus bool         `json:"enable_prometheus"`
	EnableExport     bool         `json:"enable_export"`
	ExportInterval   time.Duration `json:"export_interval"`
}

// DefaultMetricsConfig returns default metrics configuration
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		CollectInterval: 15 * time.Second,
		EnablePrometheus: true,
		EnableExport:     false,
		ExportInterval:   60 * time.Second,
	}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config *MetricsConfig) *MetricsCollector {
	if config == nil {
		config = DefaultMetricsConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	collector := &MetricsCollector{
		metrics:        make(map[string]*Metric),
		registry:       prometheus.NewRegistry(),
		counters:       make(map[string]*prometheus.CounterVec),
		gauges:         make(map[string]*prometheus.GaugeVec),
		histograms:     make(map[string]*prometheus.HistogramVec),
		summaries:      make(map[string]*prometheus.SummaryVec),
		collectInterval: config.CollectInterval,
		ctx:            ctx,
		cancel:         cancel,
	}

	if config.EnablePrometheus {
		collector.initPrometheusMetrics()
	}

	return collector
}

// initPrometheusMetrics initializes default Prometheus metrics
func (mc *MetricsCollector) initPrometheusMetrics() {
	// HTTP metrics
	mc.counters["http_requests_total"] = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	mc.histograms["http_request_duration_seconds"] = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// System metrics
	mc.gauges["system_memory_usage_bytes"] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_memory_usage_bytes",
			Help: "System memory usage in bytes",
		},
		[]string{"type"},
	)

	mc.gauges["system_cpu_usage_percent"] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_cpu_usage_percent",
			Help: "System CPU usage percentage",
		},
		[]string{"core"},
	)

	// Register all metrics
	for _, counter := range mc.counters {
		mc.registry.MustRegister(counter)
	}
	for _, gauge := range mc.gauges {
		mc.registry.MustRegister(gauge)
	}
	for _, histogram := range mc.histograms {
		mc.registry.MustRegister(histogram)
	}
	for _, summary := range mc.summaries {
		mc.registry.MustRegister(summary)
	}
}

// Start begins metrics collection
func (mc *MetricsCollector) Start() {
	logger.Info("Starting metrics collector",
		zap.Duration("interval", mc.collectInterval))

	go func() {
		ticker := time.NewTicker(mc.collectInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				mc.collectSystemMetrics()
			case <-mc.ctx.Done():
				return
			}
		}
	}()
}

// Stop stops metrics collection
func (mc *MetricsCollector) Stop() {
	logger.Info("Stopping metrics collector")
	mc.cancel()
}

// RecordCounter increments a counter metric
func (mc *MetricsCollector) RecordCounter(name string, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if counter, exists := mc.counters[name]; exists {
		counter.With(prometheus.Labels(labels)).Inc()
	}

	mc.recordMetric(name, MetricTypeCounter, 1, labels)
}

// RecordGauge sets a gauge metric
func (mc *MetricsCollector) RecordGauge(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if gauge, exists := mc.gauges[name]; exists {
		gauge.With(prometheus.Labels(labels)).Set(value)
	}

	mc.recordMetric(name, MetricTypeGauge, value, labels)
}

// RecordHistogram records a histogram observation
func (mc *MetricsCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if histogram, exists := mc.histograms[name]; exists {
		histogram.With(prometheus.Labels(labels)).Observe(value)
	}

	mc.recordMetric(name, MetricTypeHistogram, value, labels)
}

// recordMetric records a metric internally
func (mc *MetricsCollector) recordMetric(name string, metricType MetricType, value float64, labels map[string]string) {
	metric := &Metric{
		Name:      name,
		Type:      metricType,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}

	mc.metrics[name] = metric
}

// GetMetric returns a metric by name
func (mc *MetricsCollector) GetMetric(name string) (*Metric, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metric, exists := mc.metrics[name]
	if !exists {
		return nil, false
	}

	// Return a copy
	copy := *metric
	return &copy, true
}

// GetAllMetrics returns all collected metrics
func (mc *MetricsCollector) GetAllMetrics() map[string]*Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := make(map[string]*Metric)
	for k, v := range mc.metrics {
		copy := *v
		metrics[k] = &copy
	}

	return metrics
}

// GetRegistry returns the Prometheus registry
func (mc *MetricsCollector) GetRegistry() *prometheus.Registry {
	return mc.registry
}

// collectSystemMetrics collects system-level metrics
func (mc *MetricsCollector) collectSystemMetrics() {
	// Collect memory metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	mc.RecordGauge("system_memory_usage_bytes", float64(memStats.Alloc), map[string]string{"type": "alloc"})
	mc.RecordGauge("system_memory_usage_bytes", float64(memStats.Sys), map[string]string{"type": "sys"})
	mc.RecordGauge("system_memory_usage_bytes", float64(memStats.HeapAlloc), map[string]string{"type": "heap_alloc"})

	// Collect goroutine count
	mc.RecordGauge("system_goroutines", float64(runtime.NumGoroutine()), map[string]string{})

	logger.Debug("System metrics collected",
		zap.Uint64("memory_alloc", memStats.Alloc),
		zap.Int("goroutines", runtime.NumGoroutine()))
}
