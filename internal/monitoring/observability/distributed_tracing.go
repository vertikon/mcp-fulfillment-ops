// Package observability provides distributed tracing capabilities
package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TraceConfig represents tracing configuration
type TraceConfig struct {
	ServiceName      string        `json:"service_name"`
	ServiceVersion   string        `json:"service_version"`
	Environment      string        `json:"environment"`
	SamplingRate     float64       `json:"sampling_rate"`
	EnableTracing    bool          `json:"enable_tracing"`
	ExporterEndpoint string        `json:"exporter_endpoint"`
	BatchTimeout     time.Duration `json:"batch_timeout"`
	MaxExportBatch   int           `json:"max_export_batch"`
}

// DefaultTraceConfig returns default tracing configuration
func DefaultTraceConfig() *TraceConfig {
	return &TraceConfig{
		ServiceName:      "mcp-fulfillment-ops",
		ServiceVersion:   "1.0.0",
		Environment:      "production",
		SamplingRate:     1.0,
		EnableTracing:    true,
		ExporterEndpoint: "http://localhost:4318/v1/traces",
		BatchTimeout:     5 * time.Second,
		MaxExportBatch:   512,
	}
}

// DistributedTracer provides distributed tracing functionality
type DistributedTracer struct {
	config     *TraceConfig
	tracer     trace.Tracer
	mu         sync.RWMutex
	spans      map[string]*SpanInfo
	propagator propagation.TextMapPropagator
}

// SpanInfo represents information about a span
type SpanInfo struct {
	TraceID      string                 `json:"trace_id"`
	SpanID       string                 `json:"span_id"`
	ParentSpanID string                 `json:"parent_span_id,omitempty"`
	Name         string                 `json:"name"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Duration     *time.Duration         `json:"duration,omitempty"`
	Status       string                 `json:"status"`
	Attributes   map[string]interface{} `json:"attributes"`
	Events       []SpanEvent            `json:"events,omitempty"`
	Links        []SpanLink             `json:"links,omitempty"`
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string                 `json:"name"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes"`
}

// SpanLink represents a link to another span
type SpanLink struct {
	TraceID    string                 `json:"trace_id"`
	SpanID     string                 `json:"span_id"`
	Attributes map[string]interface{} `json:"attributes"`
}

// TraceStats provides tracing statistics
type TraceStats struct {
	TotalSpans      int64         `json:"total_spans"`
	ActiveSpans     int           `json:"active_spans"`
	CompletedSpans  int64         `json:"completed_spans"`
	FailedSpans     int64         `json:"failed_spans"`
	AverageDuration time.Duration `json:"average_duration"`
	LastTrace       *time.Time    `json:"last_trace,omitempty"`
}

// NewDistributedTracer creates a new distributed tracer
func NewDistributedTracer(config *TraceConfig) (*DistributedTracer, error) {
	if config == nil {
		config = DefaultTraceConfig()
	}

	tracer := otel.Tracer(config.ServiceName)

	dt := &DistributedTracer{
		config:     config,
		tracer:     tracer,
		spans:      make(map[string]*SpanInfo),
		propagator: otel.GetTextMapPropagator(),
	}

	logger.Info("Distributed tracer initialized",
		zap.String("service", config.ServiceName),
		zap.Bool("enabled", config.EnableTracing))

	return dt, nil
}

// StartSpan starts a new span
func (dt *DistributedTracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !dt.config.EnableTracing {
		return ctx, trace.SpanFromContext(ctx)
	}

	ctx, span := dt.tracer.Start(ctx, name, opts...)

	spanCtx := span.SpanContext()
	if spanCtx.IsValid() {
		spanInfo := &SpanInfo{
			TraceID:    spanCtx.TraceID().String(),
			SpanID:     spanCtx.SpanID().String(),
			Name:       name,
			StartTime:  time.Now(),
			Status:     "active",
			Attributes: make(map[string]interface{}),
			Events:     make([]SpanEvent, 0),
			Links:      make([]SpanLink, 0),
		}

		// Get parent span ID if exists
		if parentSpan := trace.SpanFromContext(ctx); parentSpan.SpanContext().IsValid() {
			spanInfo.ParentSpanID = parentSpan.SpanContext().SpanID().String()
		}

		dt.mu.Lock()
		dt.spans[spanInfo.SpanID] = spanInfo
		dt.mu.Unlock()
	}

	return ctx, span
}

// EndSpan ends a span
func (dt *DistributedTracer) EndSpan(ctx context.Context, span trace.Span, err error) {
	if !dt.config.EnableTracing {
		return
	}

	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return
	}

	dt.mu.Lock()
	spanInfo, exists := dt.spans[spanCtx.SpanID().String()]
	dt.mu.Unlock()

	if !exists {
		return
	}

	endTime := time.Now()
	duration := endTime.Sub(spanInfo.StartTime)

	spanInfo.EndTime = &endTime
	spanInfo.Duration = &duration

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		spanInfo.Status = "error"
	} else {
		span.SetStatus(codes.Ok, "")
		spanInfo.Status = "ok"
	}

	span.End()
}

// AddSpanAttribute adds an attribute to a span
func (dt *DistributedTracer) AddSpanAttribute(ctx context.Context, key string, value interface{}) {
	if !dt.config.EnableTracing {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}

	spanCtx := span.SpanContext()
	dt.mu.Lock()
	spanInfo, exists := dt.spans[spanCtx.SpanID().String()]
	dt.mu.Unlock()

	if exists {
		spanInfo.Attributes[key] = value
	}

	// Add to OpenTelemetry span
	switch v := value.(type) {
	case string:
		span.SetAttributes(attribute.String(key, v))
	case int:
		span.SetAttributes(attribute.Int(key, v))
	case int64:
		span.SetAttributes(attribute.Int64(key, v))
	case float64:
		span.SetAttributes(attribute.Float64(key, v))
	case bool:
		span.SetAttributes(attribute.Bool(key, v))
	default:
		span.SetAttributes(attribute.String(key, toString(v)))
	}
}

// AddSpanEvent adds an event to a span
func (dt *DistributedTracer) AddSpanEvent(ctx context.Context, name string, attributes map[string]interface{}) {
	if !dt.config.EnableTracing {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}

	spanCtx := span.SpanContext()
	dt.mu.Lock()
	spanInfo, exists := dt.spans[spanCtx.SpanID().String()]
	dt.mu.Unlock()

	if exists {
		event := SpanEvent{
			Name:       name,
			Timestamp:  time.Now(),
			Attributes: attributes,
		}
		spanInfo.Events = append(spanInfo.Events, event)
	}

	// Add to OpenTelemetry span
	otelAttrs := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		otelAttrs = append(otelAttrs, attribute.String(k, toString(v)))
	}
	span.AddEvent(name, trace.WithAttributes(otelAttrs...))
}

// GetSpanInfo returns information about a span
func (dt *DistributedTracer) GetSpanInfo(spanID string) (*SpanInfo, bool) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	spanInfo, exists := dt.spans[spanID]
	if !exists {
		return nil, false
	}

	// Return a copy
	copy := *spanInfo
	return &copy, true
}

// GetAllSpans returns all spans
func (dt *DistributedTracer) GetAllSpans() map[string]*SpanInfo {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	spans := make(map[string]*SpanInfo)
	for k, v := range dt.spans {
		copy := *v
		spans[k] = &copy
	}

	return spans
}

// GetTraceStats returns tracing statistics
func (dt *DistributedTracer) GetTraceStats() TraceStats {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	stats := TraceStats{
		TotalSpans:     int64(len(dt.spans)),
		ActiveSpans:    0,
		CompletedSpans: 0,
		FailedSpans:    0,
	}

	var totalDuration time.Duration
	var completedCount int64

	for _, spanInfo := range dt.spans {
		if spanInfo.EndTime == nil {
			stats.ActiveSpans++
		} else {
			stats.CompletedSpans++
			completedCount++
			if spanInfo.Duration != nil {
				totalDuration += *spanInfo.Duration
			}
			if spanInfo.Status == "error" {
				stats.FailedSpans++
			}
		}
	}

	if completedCount > 0 {
		stats.AverageDuration = totalDuration / time.Duration(completedCount)
	}

	return stats
}

// InjectTraceContext injects trace context into a carrier
func (dt *DistributedTracer) InjectTraceContext(ctx context.Context, carrier propagation.TextMapCarrier) {
	if !dt.config.EnableTracing {
		return
	}
	dt.propagator.Inject(ctx, carrier)
}

// ExtractTraceContext extracts trace context from a carrier
func (dt *DistributedTracer) ExtractTraceContext(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	if !dt.config.EnableTracing {
		return ctx
	}
	return dt.propagator.Extract(ctx, carrier)
}

// Helper function to convert value to string
func toString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}
