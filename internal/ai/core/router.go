package core

import (
	"context"
	"fmt"
	"time"
)

// RoutingStrategy defines how to select providers
type RoutingStrategy string

const (
	StrategyCost     RoutingStrategy = "cost"
	StrategyLatency  RoutingStrategy = "latency"
	StrategyQuality  RoutingStrategy = "quality"
	StrategyBalanced RoutingStrategy = "balanced"
	StrategyFallback RoutingStrategy = "fallback"
)

// ProviderConfig contains configuration for a provider
type ProviderConfig struct {
	Provider      LLMProvider
	Models        []string
	DefaultModel  string
	CostPerToken  float64
	MaxTokens     int
	Priority      int // Higher priority = preferred
	Enabled       bool
	FallbackOrder int // Order for fallback (lower = first fallback)
}

// Router intelligently routes requests to the best LLM provider
type Router struct {
	configs       map[LLMProvider]*ProviderConfig
	strategy      RoutingStrategy
	metrics       *Metrics
	availability  map[LLMProvider]bool
	lastCheck     map[LLMProvider]time.Time
	checkInterval time.Duration
}

// NewRouter creates a new router
func NewRouter(configs map[LLMProvider]*ProviderConfig, strategy RoutingStrategy, metrics *Metrics) *Router {
	return &Router{
		configs:       configs,
		strategy:      strategy,
		metrics:       metrics,
		availability:  make(map[LLMProvider]bool),
		lastCheck:     make(map[LLMProvider]time.Time),
		checkInterval: 30 * time.Second,
	}
}

// SelectProvider selects the best provider for a request
func (r *Router) SelectProvider(ctx context.Context, req *LLMRequest) (LLMProvider, string, error) {
	// Update availability cache
	r.updateAvailability(ctx)

	// Filter enabled and available providers
	available := r.getAvailableProviders()

	if len(available) == 0 {
		return "", "", fmt.Errorf("no available providers")
	}

	// Select based on strategy
	var selected LLMProvider
	var model string

	switch r.strategy {
	case StrategyCost:
		selected, model = r.selectByCost(available, req)
	case StrategyLatency:
		selected, model = r.selectByLatency(available, req)
	case StrategyQuality:
		selected, model = r.selectByQuality(available, req)
	case StrategyBalanced:
		selected, model = r.selectBalanced(available, req)
	case StrategyFallback:
		selected, model = r.selectFallback(available, req)
	default:
		selected, model = r.selectByPriority(available, req)
	}

	if selected == "" {
		return "", "", fmt.Errorf("failed to select provider")
	}

	return selected, model, nil
}

// SelectFallback selects a fallback provider
func (r *Router) SelectFallback(ctx context.Context, req *LLMRequest, failedProvider LLMProvider) (LLMProvider, string, error) {
	r.updateAvailability(ctx)
	available := r.getAvailableProviders()

	// Remove failed provider from available
	filtered := make([]LLMProvider, 0)
	for _, p := range available {
		if p != failedProvider {
			filtered = append(filtered, p)
		}
	}

	if len(filtered) == 0 {
		return "", "", fmt.Errorf("no fallback providers available")
	}

	// Select by fallback order
	var selected LLMProvider
	var selectedConfig *ProviderConfig
	lowestOrder := 999

	for _, p := range filtered {
		config, exists := r.configs[p]
		if !exists || !config.Enabled {
			continue
		}
		if config.FallbackOrder < lowestOrder {
			lowestOrder = config.FallbackOrder
			selected = p
			selectedConfig = config
		}
	}

	if selected == "" {
		return "", "", fmt.Errorf("no fallback provider found")
	}

	model := r.selectModel(selected, req, selectedConfig)
	return selected, model, nil
}

// selectByCost selects provider with lowest cost
func (r *Router) selectByCost(available []LLMProvider, req *LLMRequest) (LLMProvider, string) {
	var selected LLMProvider
	var selectedConfig *ProviderConfig
	lowestCost := 999999.0

	for _, p := range available {
		config, exists := r.configs[p]
		if !exists || !config.Enabled {
			continue
		}
		if config.CostPerToken < lowestCost {
			lowestCost = config.CostPerToken
			selected = p
			selectedConfig = config
		}
	}

	model := r.selectModel(selected, req, selectedConfig)
	return selected, model
}

// selectByLatency selects provider with best latency
func (r *Router) selectByLatency(available []LLMProvider, req *LLMRequest) (LLMProvider, string) {
	var selected LLMProvider
	var selectedConfig *ProviderConfig
	bestLatency := time.Duration(999999) * time.Second

	for _, p := range available {
		config, exists := r.configs[p]
		if !exists || !config.Enabled {
			continue
		}
		avgLatency := r.metrics.GetAverageLatency(p, "")
		if avgLatency > 0 && avgLatency < bestLatency {
			bestLatency = avgLatency
			selected = p
			selectedConfig = config
		}
	}

	// Fallback to priority if no latency data
	if selected == "" {
		return r.selectByPriority(available, req)
	}

	model := r.selectModel(selected, req, selectedConfig)
	return selected, model
}

// selectByQuality selects provider with best quality (highest priority)
func (r *Router) selectByQuality(available []LLMProvider, req *LLMRequest) (LLMProvider, string) {
	return r.selectByPriority(available, req)
}

// selectBalanced selects provider balancing cost, latency, and quality
func (r *Router) selectBalanced(available []LLMProvider, req *LLMRequest) (LLMProvider, string) {
	var selected LLMProvider
	var selectedConfig *ProviderConfig
	bestScore := -1.0

	for _, p := range available {
		config, exists := r.configs[p]
		if !exists || !config.Enabled {
			continue
		}

		// Calculate balanced score
		latencyScore := 1.0
		if avgLatency := r.metrics.GetAverageLatency(p, ""); avgLatency > 0 {
			latencyScore = 1.0 / (1.0 + float64(avgLatency.Milliseconds())/1000.0)
		}

		costScore := 1.0 / (1.0 + config.CostPerToken*1000)
		priorityScore := float64(config.Priority) / 10.0

		score := (latencyScore * 0.4) + (costScore * 0.3) + (priorityScore * 0.3)

		if score > bestScore {
			bestScore = score
			selected = p
			selectedConfig = config
		}
	}

	if selected == "" {
		return r.selectByPriority(available, req)
	}

	model := r.selectModel(selected, req, selectedConfig)
	return selected, model
}

// selectFallback selects provider based on fallback order
func (r *Router) selectFallback(available []LLMProvider, req *LLMRequest) (LLMProvider, string) {
	return r.selectByPriority(available, req)
}

// selectByPriority selects provider with highest priority
func (r *Router) selectByPriority(available []LLMProvider, req *LLMRequest) (LLMProvider, string) {
	var selected LLMProvider
	var selectedConfig *ProviderConfig
	highestPriority := -1

	for _, p := range available {
		config, exists := r.configs[p]
		if !exists || !config.Enabled {
			continue
		}
		if config.Priority > highestPriority {
			highestPriority = config.Priority
			selected = p
			selectedConfig = config
		}
	}

	model := r.selectModel(selected, req, selectedConfig)
	return selected, model
}

// selectModel selects the best model for a provider
func (r *Router) selectModel(provider LLMProvider, req *LLMRequest, config *ProviderConfig) string {
	// Use requested model if specified and available
	if req.Model != "" {
		for _, m := range config.Models {
			if m == req.Model {
				return m
			}
		}
	}

	// Use default model
	if config.DefaultModel != "" {
		return config.DefaultModel
	}

	// Use first available model
	if len(config.Models) > 0 {
		return config.Models[0]
	}

	return "default"
}

// getAvailableProviders returns list of available providers
func (r *Router) getAvailableProviders() []LLMProvider {
	var available []LLMProvider
	for p, config := range r.configs {
		if !config.Enabled {
			continue
		}
		if isAvailable, ok := r.availability[p]; ok && isAvailable {
			available = append(available, p)
		} else if !ok {
			// Assume available if not checked yet
			available = append(available, p)
		}
	}
	return available
}

// updateAvailability updates availability cache
func (r *Router) updateAvailability(ctx context.Context) {
	now := time.Now()
	for p := range r.configs {
		lastCheck, ok := r.lastCheck[p]
		if !ok || now.Sub(lastCheck) > r.checkInterval {
			// In real implementation, this would check actual provider availability
			// For now, assume all enabled providers are available
			r.availability[p] = true
			r.lastCheck[p] = now
		}
	}
}

// SetAvailability manually sets provider availability
func (r *Router) SetAvailability(provider LLMProvider, available bool) {
	r.availability[provider] = available
	r.lastCheck[provider] = time.Now()
}
