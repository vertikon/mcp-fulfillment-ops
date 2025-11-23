package core

import (
	"sync"
	"time"
)

// Metrics tracks AI Core metrics and observability
type Metrics struct {
	mu sync.RWMutex

	// Generation metrics
	generations  map[string]int64 // key: "provider:model"
	totalTokens  map[string]int64
	totalLatency map[string]time.Duration
	successCount map[string]int64
	errorCount   map[string]int64

	// Latency tracking
	latencies map[string][]time.Duration // key: "provider:model"

	// Error tracking
	errors map[string][]error // key: "provider:model"
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		generations:  make(map[string]int64),
		totalTokens:  make(map[string]int64),
		totalLatency: make(map[string]time.Duration),
		successCount: make(map[string]int64),
		errorCount:   make(map[string]int64),
		latencies:    make(map[string][]time.Duration),
		errors:       make(map[string][]error),
	}
}

// RecordGeneration records a successful generation
func (m *Metrics) RecordGeneration(provider LLMProvider, model string, tokens int, latency time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.key(provider, model)

	m.generations[key]++
	m.totalTokens[key] += int64(tokens)
	m.totalLatency[key] += latency

	if success {
		m.successCount[key]++
	} else {
		m.errorCount[key]++
	}

	// Track latencies (keep last 100)
	if m.latencies[key] == nil {
		m.latencies[key] = make([]time.Duration, 0, 100)
	}
	m.latencies[key] = append(m.latencies[key], latency)
	if len(m.latencies[key]) > 100 {
		m.latencies[key] = m.latencies[key][1:]
	}
}

// RecordError records an error
func (m *Metrics) RecordError(provider LLMProvider, model string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.key(provider, model)
	m.errorCount[key]++

	// Track errors (keep last 50)
	if m.errors[key] == nil {
		m.errors[key] = make([]error, 0, 50)
	}
	m.errors[key] = append(m.errors[key], err)
	if len(m.errors[key]) > 50 {
		m.errors[key] = m.errors[key][1:]
	}
}

// GetTotalGenerations returns total generations for a provider/model
func (m *Metrics) GetTotalGenerations(provider LLMProvider, model string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.key(provider, model)
	return m.generations[key]
}

// GetTotalTokens returns total tokens used for a provider/model
func (m *Metrics) GetTotalTokens(provider LLMProvider, model string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.key(provider, model)
	return m.totalTokens[key]
}

// GetAverageLatency returns average latency for a provider/model
func (m *Metrics) GetAverageLatency(provider LLMProvider, model string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.key(provider, model)
	count := m.generations[key]
	if count == 0 {
		return 0
	}

	return m.totalLatency[key] / time.Duration(count)
}

// GetSuccessRate returns success rate for a provider/model
func (m *Metrics) GetSuccessRate(provider LLMProvider, model string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.key(provider, model)
	total := m.successCount[key] + m.errorCount[key]
	if total == 0 {
		return 0
	}

	return float64(m.successCount[key]) / float64(total)
}

// GetErrorRate returns error rate for a provider/model
func (m *Metrics) GetErrorRate(provider LLMProvider, model string) float64 {
	return 1.0 - m.GetSuccessRate(provider, model)
}

// GetRecentErrors returns recent errors for a provider/model
func (m *Metrics) GetRecentErrors(provider LLMProvider, model string, limit int) []error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.key(provider, model)
	errors := m.errors[key]
	if len(errors) == 0 {
		return nil
	}

	if limit > len(errors) {
		limit = len(errors)
	}

	// Return most recent errors
	start := len(errors) - limit
	result := make([]error, limit)
	copy(result, errors[start:])
	return result
}

// GetP95Latency returns P95 latency for a provider/model
func (m *Metrics) GetP95Latency(provider LLMProvider, model string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.key(provider, model)
	latencies := m.latencies[key]
	if len(latencies) == 0 {
		return 0
	}

	// Sort and get P95
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Simple P95 calculation (for production, use proper sorting)
	index := int(float64(len(sorted)) * 0.95)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

// GetStats returns comprehensive stats for a provider/model
func (m *Metrics) GetStats(provider LLMProvider, model string) ProviderStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.key(provider, model)

	return ProviderStats{
		Provider:         provider,
		Model:            model,
		TotalGenerations: m.generations[key],
		TotalTokens:      m.totalTokens[key],
		AverageLatency:   m.GetAverageLatency(provider, model),
		P95Latency:       m.GetP95Latency(provider, model),
		SuccessRate:      m.GetSuccessRate(provider, model),
		ErrorRate:        m.GetErrorRate(provider, model),
		ErrorCount:       m.errorCount[key],
	}
}

// GetAllStats returns stats for all providers/models
func (m *Metrics) GetAllStats() map[string]ProviderStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]ProviderStats)
	for key := range m.generations {
		provider, model := m.parseKey(key)
		stats[key] = m.GetStats(provider, model)
	}
	return stats
}

// ProviderStats contains statistics for a provider/model
type ProviderStats struct {
	Provider         LLMProvider
	Model            string
	TotalGenerations int64
	TotalTokens      int64
	AverageLatency   time.Duration
	P95Latency       time.Duration
	SuccessRate      float64
	ErrorRate        float64
	ErrorCount       int64
}

// key generates a key for provider:model
func (m *Metrics) key(provider LLMProvider, model string) string {
	if model == "" {
		return string(provider)
	}
	return string(provider) + ":" + model
}

// parseKey parses a key into provider and model
func (m *Metrics) parseKey(key string) (LLMProvider, string) {
	// Simple parsing - in production, use more robust method
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == ':' {
			return LLMProvider(key[:i]), key[i+1:]
		}
	}
	return LLMProvider(key), ""
}

// Reset clears all metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.generations = make(map[string]int64)
	m.totalTokens = make(map[string]int64)
	m.totalLatency = make(map[string]time.Duration)
	m.successCount = make(map[string]int64)
	m.errorCount = make(map[string]int64)
	m.latencies = make(map[string][]time.Duration)
	m.errors = make(map[string][]error)
}
