// Package analytics provides performance analytics capabilities
package analytics

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/vertikon/mcp-hulk/pkg/logger"
	"go.uber.org/zap"
)

// PerformanceMetric represents a performance metric
type PerformanceMetric struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Unit      string                 `json:"unit"`
	Timestamp time.Time              `json:"timestamp"`
	Labels    map[string]string      `json:"labels"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceAnalytics provides performance analytics
type PerformanceAnalytics struct {
	mu           sync.RWMutex
	metrics      map[string][]PerformanceMetric
	aggregations map[string]*Aggregation
	windowSize   time.Duration
	retention    time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
}

// Aggregation represents aggregated metrics
type Aggregation struct {
	Name       string                 `json:"name"`
	Min        float64                `json:"min"`
	Max        float64                `json:"max"`
	Average    float64                `json:"average"`
	Median     float64                `json:"median"`
	P95        float64                `json:"p95"`
	P99        float64                `json:"p99"`
	Count      int64                  `json:"count"`
	LastUpdate time.Time              `json:"last_update"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceReport provides performance analysis report
type PerformanceReport struct {
	Period          time.Duration           `json:"period"`
	StartTime       time.Time               `json:"start_time"`
	EndTime         time.Time               `json:"end_time"`
	Metrics         map[string]*Aggregation `json:"metrics"`
	Recommendations []string                `json:"recommendations,omitempty"`
	Summary         map[string]interface{}  `json:"summary"`
}

// AnalyticsConfig represents analytics configuration
type AnalyticsConfig struct {
	WindowSize      time.Duration `json:"window_size"`
	Retention       time.Duration `json:"retention"`
	EnableAnalytics bool          `json:"enable_analytics"`
}

// DefaultAnalyticsConfig returns default analytics configuration
func DefaultAnalyticsConfig() *AnalyticsConfig {
	return &AnalyticsConfig{
		WindowSize:      5 * time.Minute,
		Retention:       24 * time.Hour,
		EnableAnalytics: true,
	}
}

// NewPerformanceAnalytics creates a new performance analytics instance
func NewPerformanceAnalytics(config *AnalyticsConfig) *PerformanceAnalytics {
	if config == nil {
		config = DefaultAnalyticsConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	analytics := &PerformanceAnalytics{
		metrics:      make(map[string][]PerformanceMetric),
		aggregations: make(map[string]*Aggregation),
		windowSize:   config.WindowSize,
		retention:    config.Retention,
		ctx:          ctx,
		cancel:       cancel,
	}

	logger.Info("Performance analytics initialized",
		zap.Duration("window_size", config.WindowSize),
		zap.Duration("retention", config.Retention))

	return analytics
}

// RecordMetric records a performance metric
func (pa *PerformanceAnalytics) RecordMetric(metric PerformanceMetric) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	metrics := pa.metrics[metric.Name]
	metrics = append(metrics, metric)

	// Cleanup old metrics
	cutoff := time.Now().Add(-pa.retention)
	filtered := make([]PerformanceMetric, 0)
	for _, m := range metrics {
		if m.Timestamp.After(cutoff) {
			filtered = append(filtered, m)
		}
	}

	pa.metrics[metric.Name] = filtered

	// Update aggregation
	pa.updateAggregation(metric.Name)
}

// GetAggregation returns aggregation for a metric
func (pa *PerformanceAnalytics) GetAggregation(metricName string) (*Aggregation, bool) {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	agg, exists := pa.aggregations[metricName]
	if !exists {
		return nil, false
	}

	copy := *agg
	return &copy, true
}

// GetAllAggregations returns all aggregations
func (pa *PerformanceAnalytics) GetAllAggregations() map[string]*Aggregation {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	aggs := make(map[string]*Aggregation)
	for k, v := range pa.aggregations {
		copy := *v
		aggs[k] = &copy
	}

	return aggs
}

// GenerateReport generates a performance report
func (pa *PerformanceAnalytics) GenerateReport(startTime, endTime time.Time) (*PerformanceReport, error) {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	report := &PerformanceReport{
		Period:          endTime.Sub(startTime),
		StartTime:       startTime,
		EndTime:         endTime,
		Metrics:         make(map[string]*Aggregation),
		Recommendations: make([]string, 0),
		Summary:         make(map[string]interface{}),
	}

	// Filter metrics within time range
	for name, metrics := range pa.metrics {
		filtered := make([]PerformanceMetric, 0)
		for _, m := range metrics {
			if m.Timestamp.After(startTime) && m.Timestamp.Before(endTime) {
				filtered = append(filtered, m)
			}
		}

		if len(filtered) > 0 {
			agg := pa.calculateAggregation(name, filtered)
			report.Metrics[name] = agg
		}
	}

	// Generate recommendations
	report.Recommendations = pa.generateRecommendations(report)

	// Generate summary
	report.Summary = pa.generateSummary(report)

	return report, nil
}

// updateAggregation updates aggregation for a metric
func (pa *PerformanceAnalytics) updateAggregation(metricName string) {
	metrics, exists := pa.metrics[metricName]
	if !exists || len(metrics) == 0 {
		return
	}

	agg := pa.calculateAggregation(metricName, metrics)
	pa.aggregations[metricName] = agg
}

// calculateAggregation calculates aggregation statistics
func (pa *PerformanceAnalytics) calculateAggregation(name string, metrics []PerformanceMetric) *Aggregation {
	if len(metrics) == 0 {
		return &Aggregation{
			Name:       name,
			LastUpdate: time.Now(),
		}
	}

	values := make([]float64, len(metrics))
	var sum float64
	min := metrics[0].Value
	max := metrics[0].Value

	for i, m := range metrics {
		values[i] = m.Value
		sum += m.Value
		if m.Value < min {
			min = m.Value
		}
		if m.Value > max {
			max = m.Value
		}
	}

	// Sort for percentile calculation
	sort.Float64s(values)

	agg := &Aggregation{
		Name:       name,
		Min:        min,
		Max:        max,
		Average:    sum / float64(len(metrics)),
		Count:      int64(len(metrics)),
		LastUpdate: time.Now(),
	}

	// Calculate median
	if len(values) > 0 {
		mid := len(values) / 2
		if len(values)%2 == 0 {
			agg.Median = (values[mid-1] + values[mid]) / 2
		} else {
			agg.Median = values[mid]
		}

		// Calculate P95
		p95Index := int(float64(len(values)) * 0.95)
		if p95Index < len(values) {
			agg.P95 = values[p95Index]
		}

		// Calculate P99
		p99Index := int(float64(len(values)) * 0.99)
		if p99Index < len(values) {
			agg.P99 = values[p99Index]
		}
	}

	return agg
}

// generateRecommendations generates performance recommendations
func (pa *PerformanceAnalytics) generateRecommendations(report *PerformanceReport) []string {
	recommendations := make([]string, 0)

	for name, agg := range report.Metrics {
		// Check for high latency
		if agg.Average > 1000 && (name == "latency" || name == "duration") {
			recommendations = append(recommendations,
				fmt.Sprintf("High average %s detected: %.2fms. Consider optimization.", name, agg.Average))
		}

		// Check for high P95
		if agg.P95 > 2000 && (name == "latency" || name == "duration") {
			recommendations = append(recommendations,
				fmt.Sprintf("High P95 %s detected: %.2fms. Review slow operations.", name, agg.P95))
		}

		// Check for high variance
		if agg.Max-agg.Min > agg.Average*2 {
			recommendations = append(recommendations,
				fmt.Sprintf("High variance in %s detected. Investigate outliers.", name))
		}
	}

	return recommendations
}

// generateSummary generates report summary
func (pa *PerformanceAnalytics) generateSummary(report *PerformanceReport) map[string]interface{} {
	summary := make(map[string]interface{})

	totalMetrics := len(report.Metrics)
	summary["total_metrics"] = totalMetrics
	summary["period"] = report.Period.String()

	if totalMetrics > 0 {
		var totalAvg float64
		for _, agg := range report.Metrics {
			totalAvg += agg.Average
		}
		summary["average_metric_value"] = totalAvg / float64(totalMetrics)
	}

	return summary
}
