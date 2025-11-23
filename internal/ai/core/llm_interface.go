// Package core provides AI Core functionality including LLM interface, prompt building, routing, and metrics.
package core

import (
	"context"
	"fmt"
	"time"
)

// LLMProvider represents a supported LLM provider
type LLMProvider string

const (
	ProviderOpenAI LLMProvider = "openai"
	ProviderGemini LLMProvider = "gemini"
	ProviderGLM    LLMProvider = "glm"
)

// LLMRequest represents a request to an LLM
type LLMRequest struct {
	Prompt      string
	Model       string
	Temperature float64
	MaxTokens   int
	Stop        []string
	Stream      bool
	Metadata    map[string]interface{}
}

// LLMResponse represents a response from an LLM
type LLMResponse struct {
	Content      string
	Model        string
	Provider     LLMProvider
	TokensUsed   int
	FinishReason string
	Latency      time.Duration
	Metadata     map[string]interface{}
}

// LLMError represents an error from an LLM call
type LLMError struct {
	Provider  LLMProvider
	Model     string
	Message   string
	Retryable bool
	Err       error
}

func (e *LLMError) Error() string {
	return fmt.Sprintf("LLM error [%s/%s]: %s", e.Provider, e.Model, e.Message)
}

func (e *LLMError) Unwrap() error {
	return e.Err
}

// LLMClient defines the interface for LLM providers
type LLMClient interface {
	// Generate generates a completion from the LLM
	Generate(ctx context.Context, req *LLMRequest) (*LLMResponse, error)

	// GenerateStream generates a streaming completion
	GenerateStream(ctx context.Context, req *LLMRequest) (<-chan *LLMResponse, error)

	// Provider returns the provider name
	Provider() LLMProvider

	// IsAvailable checks if the provider is available
	IsAvailable(ctx context.Context) bool

	// GetModels returns available models for this provider
	GetModels(ctx context.Context) ([]string, error)
}

// LLMInterface provides a unified interface for LLM operations
type LLMInterface struct {
	clients map[LLMProvider]LLMClient
	router  *Router
	metrics *Metrics
}

// NewLLMInterface creates a new LLM interface
func NewLLMInterface(clients map[LLMProvider]LLMClient, router *Router, metrics *Metrics) *LLMInterface {
	return &LLMInterface{
		clients: clients,
		router:  router,
		metrics: metrics,
	}
}

// Generate generates a completion using the best available provider
func (li *LLMInterface) Generate(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	start := time.Now()

	// Use router to select best provider
	provider, model, err := li.router.SelectProvider(ctx, req)
	if err != nil {
		li.metrics.RecordError(provider, model, err)
		return nil, fmt.Errorf("failed to select provider: %w", err)
	}

	client, exists := li.clients[provider]
	if !exists {
		err := fmt.Errorf("provider %s not available", provider)
		li.metrics.RecordError(provider, model, err)
		return nil, err
	}

	// Check availability
	if !client.IsAvailable(ctx) {
		// Try fallback
		provider, model, err = li.router.SelectFallback(ctx, req, provider)
		if err != nil {
			li.metrics.RecordError(provider, model, err)
			return nil, fmt.Errorf("no available providers: %w", err)
		}
		client = li.clients[provider]
	}

	// Update request with selected model
	req.Model = model

	// Generate with retry logic
	var resp *LLMResponse
	var lastErr error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		resp, lastErr = client.Generate(ctx, req)
		if lastErr == nil {
			break
		}

		// Check if error is retryable
		if llmErr, ok := lastErr.(*LLMError); ok && !llmErr.Retryable {
			break
		}

		// Exponential backoff
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
		}
	}

	if lastErr != nil {
		li.metrics.RecordError(provider, model, lastErr)
		return nil, fmt.Errorf("generation failed after retries: %w", lastErr)
	}

	// Record metrics
	latency := time.Since(start)
	li.metrics.RecordGeneration(provider, model, resp.TokensUsed, latency, true)

	return resp, nil
}

// GenerateStream generates a streaming completion
func (li *LLMInterface) GenerateStream(ctx context.Context, req *LLMRequest) (<-chan *LLMResponse, error) {
	req.Stream = true

	provider, model, err := li.router.SelectProvider(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to select provider: %w", err)
	}

	client, exists := li.clients[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not available", provider)
	}

	req.Model = model
	return client.GenerateStream(ctx, req)
}

// GetAvailableProviders returns list of available providers
func (li *LLMInterface) GetAvailableProviders(ctx context.Context) []LLMProvider {
	var available []LLMProvider
	for provider, client := range li.clients {
		if client.IsAvailable(ctx) {
			available = append(available, provider)
		}
	}
	return available
}

// GetModels returns all available models across providers
func (li *LLMInterface) GetModels(ctx context.Context) (map[LLMProvider][]string, error) {
	models := make(map[LLMProvider][]string)
	for provider, client := range li.clients {
		if !client.IsAvailable(ctx) {
			continue
		}
		providerModels, err := client.GetModels(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get models for %s: %w", provider, err)
		}
		models[provider] = providerModels
	}
	return models, nil
}
