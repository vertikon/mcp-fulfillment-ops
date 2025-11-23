package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockLLMClient is a mock implementation of LLMClient for testing
type mockLLMClient struct {
	provider      LLMProvider
	available     bool
	models        []string
	generateFunc  func(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	streamFunc    func(ctx context.Context, req *LLMRequest) (<-chan *LLMResponse, error)
	availableFunc func(ctx context.Context) bool
}

func (m *mockLLMClient) Generate(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, req)
	}
	return &LLMResponse{
		Content:      "mock response",
		Model:        req.Model,
		Provider:     m.provider,
		TokensUsed:   10,
		FinishReason: "stop",
		Latency:      100 * time.Millisecond,
		Metadata:     make(map[string]interface{}),
	}, nil
}

func (m *mockLLMClient) GenerateStream(ctx context.Context, req *LLMRequest) (<-chan *LLMResponse, error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, req)
	}
	ch := make(chan *LLMResponse, 1)
	ch <- &LLMResponse{
		Content:      "stream response",
		Model:        req.Model,
		Provider:     m.provider,
		TokensUsed:   5,
		FinishReason: "stop",
		Latency:      50 * time.Millisecond,
		Metadata:     make(map[string]interface{}),
	}
	close(ch)
	return ch, nil
}

func (m *mockLLMClient) Provider() LLMProvider {
	return m.provider
}

func (m *mockLLMClient) IsAvailable(ctx context.Context) bool {
	if m.availableFunc != nil {
		return m.availableFunc(ctx)
	}
	return m.available
}

func (m *mockLLMClient) GetModels(ctx context.Context) ([]string, error) {
	return m.models, nil
}

func TestNewLLMInterface(t *testing.T) {
	clients := make(map[LLMProvider]LLMClient)
	router := NewRouter(make(map[LLMProvider]*ProviderConfig), StrategyBalanced, NewMetrics())
	metrics := NewMetrics()

	li := NewLLMInterface(clients, router, metrics)

	if li == nil {
		t.Fatal("NewLLMInterface returned nil")
	}
	if li.clients == nil {
		t.Error("clients map is nil")
	}
	if li.router == nil {
		t.Error("router is nil")
	}
	if li.metrics == nil {
		t.Error("metrics is nil")
	}
}

func TestLLMInterface_Generate_Success(t *testing.T) {
	clients := make(map[LLMProvider]LLMClient)
	mockClient := &mockLLMClient{
		provider:  ProviderOpenAI,
		available: true,
		models:    []string{"gpt-4", "gpt-3.5-turbo"},
	}
	clients[ProviderOpenAI] = mockClient

	configs := map[LLMProvider]*ProviderConfig{
		ProviderOpenAI: {
			Provider:     ProviderOpenAI,
			Models:       []string{"gpt-4", "gpt-3.5-turbo"},
			DefaultModel: "gpt-4",
			Priority:     10,
			Enabled:      true,
		},
	}
	router := NewRouter(configs, StrategyBalanced, NewMetrics())
	metrics := NewMetrics()

	li := NewLLMInterface(clients, router, metrics)
	router.SetAvailability(ProviderOpenAI, true)

	req := &LLMRequest{
		Prompt:      "Hello",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	resp, err := li.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}
	if resp.Provider != ProviderOpenAI {
		t.Errorf("Expected provider %v, got %v", ProviderOpenAI, resp.Provider)
	}
	if resp.Content != "mock response" {
		t.Errorf("Expected content 'mock response', got '%s'", resp.Content)
	}
}

func TestLLMInterface_Generate_Fallback(t *testing.T) {
	clients := make(map[LLMProvider]LLMClient)

	// Primary provider unavailable
	primaryClient := &mockLLMClient{
		provider:  ProviderOpenAI,
		available: false,
		models:    []string{"gpt-4"},
	}

	// Fallback provider available
	fallbackClient := &mockLLMClient{
		provider:  ProviderGemini,
		available: true,
		models:    []string{"gemini-pro"},
	}

	clients[ProviderOpenAI] = primaryClient
	clients[ProviderGemini] = fallbackClient

	configs := map[LLMProvider]*ProviderConfig{
		ProviderOpenAI: {
			Provider:      ProviderOpenAI,
			Models:        []string{"gpt-4"},
			DefaultModel:  "gpt-4",
			Priority:      10,
			Enabled:       true,
			FallbackOrder: 1,
		},
		ProviderGemini: {
			Provider:      ProviderGemini,
			Models:        []string{"gemini-pro"},
			DefaultModel:  "gemini-pro",
			Priority:      8,
			Enabled:       true,
			FallbackOrder: 0,
		},
	}
	router := NewRouter(configs, StrategyBalanced, NewMetrics())
	metrics := NewMetrics()

	li := NewLLMInterface(clients, router, metrics)
	router.SetAvailability(ProviderOpenAI, false)
	router.SetAvailability(ProviderGemini, true)

	req := &LLMRequest{
		Prompt:      "Hello",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	resp, err := li.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resp.Provider != ProviderGemini {
		t.Errorf("Expected fallback to %v, got %v", ProviderGemini, resp.Provider)
	}
}

func TestLLMInterface_Generate_Retry(t *testing.T) {
	attempts := 0
	clients := make(map[LLMProvider]LLMClient)
	mockClient := &mockLLMClient{
		provider:  ProviderOpenAI,
		available: true,
		models:    []string{"gpt-4"},
		generateFunc: func(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
			attempts++
			if attempts < 2 {
				return nil, &LLMError{
					Provider:  ProviderOpenAI,
					Model:     req.Model,
					Message:   "temporary error",
					Retryable: true,
					Err:       errors.New("network error"),
				}
			}
			return &LLMResponse{
				Content:      "success after retry",
				Model:        req.Model,
				Provider:     ProviderOpenAI,
				TokensUsed:   10,
				FinishReason: "stop",
				Latency:      100 * time.Millisecond,
				Metadata:     make(map[string]interface{}),
			}, nil
		},
	}
	clients[ProviderOpenAI] = mockClient

	configs := map[LLMProvider]*ProviderConfig{
		ProviderOpenAI: {
			Provider:     ProviderOpenAI,
			Models:       []string{"gpt-4"},
			DefaultModel: "gpt-4",
			Priority:     10,
			Enabled:      true,
		},
	}
	router := NewRouter(configs, StrategyBalanced, NewMetrics())
	metrics := NewMetrics()

	li := NewLLMInterface(clients, router, metrics)
	router.SetAvailability(ProviderOpenAI, true)

	req := &LLMRequest{
		Prompt:      "Hello",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	resp, err := li.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if attempts < 2 {
		t.Errorf("Expected at least 2 attempts, got %d", attempts)
	}
	if resp.Content != "success after retry" {
		t.Errorf("Expected 'success after retry', got '%s'", resp.Content)
	}
}

func TestLLMInterface_GenerateStream(t *testing.T) {
	clients := make(map[LLMProvider]LLMClient)
	mockClient := &mockLLMClient{
		provider:  ProviderOpenAI,
		available: true,
		models:    []string{"gpt-4"},
	}
	clients[ProviderOpenAI] = mockClient

	configs := map[LLMProvider]*ProviderConfig{
		ProviderOpenAI: {
			Provider:     ProviderOpenAI,
			Models:       []string{"gpt-4"},
			DefaultModel: "gpt-4",
			Priority:     10,
			Enabled:      true,
		},
	}
	router := NewRouter(configs, StrategyBalanced, NewMetrics())
	metrics := NewMetrics()

	li := NewLLMInterface(clients, router, metrics)
	router.SetAvailability(ProviderOpenAI, true)

	req := &LLMRequest{
		Prompt:      "Hello",
		Temperature: 0.7,
		MaxTokens:   100,
		Stream:      true,
	}

	ch, err := li.GenerateStream(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateStream failed: %v", err)
	}

	resp := <-ch
	if resp == nil {
		t.Fatal("Stream response is nil")
	}
	if resp.Content != "stream response" {
		t.Errorf("Expected 'stream response', got '%s'", resp.Content)
	}
}

func TestLLMInterface_GetAvailableProviders(t *testing.T) {
	clients := make(map[LLMProvider]LLMClient)

	clients[ProviderOpenAI] = &mockLLMClient{
		provider:  ProviderOpenAI,
		available: true,
	}
	clients[ProviderGemini] = &mockLLMClient{
		provider:  ProviderGemini,
		available: false,
	}

	router := NewRouter(make(map[LLMProvider]*ProviderConfig), StrategyBalanced, NewMetrics())
	metrics := NewMetrics()

	li := NewLLMInterface(clients, router, metrics)

	providers := li.GetAvailableProviders(context.Background())
	if len(providers) != 1 {
		t.Errorf("Expected 1 available provider, got %d", len(providers))
	}
	if providers[0] != ProviderOpenAI {
		t.Errorf("Expected %v, got %v", ProviderOpenAI, providers[0])
	}
}

func TestLLMInterface_GetModels(t *testing.T) {
	clients := make(map[LLMProvider]LLMClient)
	clients[ProviderOpenAI] = &mockLLMClient{
		provider:  ProviderOpenAI,
		available: true,
		models:    []string{"gpt-4", "gpt-3.5-turbo"},
	}
	clients[ProviderGemini] = &mockLLMClient{
		provider:  ProviderGemini,
		available: true,
		models:    []string{"gemini-pro"},
	}

	router := NewRouter(make(map[LLMProvider]*ProviderConfig), StrategyBalanced, NewMetrics())
	metrics := NewMetrics()

	li := NewLLMInterface(clients, router, metrics)

	models, err := li.GetModels(context.Background())
	if err != nil {
		t.Fatalf("GetModels failed: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(models))
	}
	if len(models[ProviderOpenAI]) != 2 {
		t.Errorf("Expected 2 OpenAI models, got %d", len(models[ProviderOpenAI]))
	}
	if len(models[ProviderGemini]) != 1 {
		t.Errorf("Expected 1 Gemini model, got %d", len(models[ProviderGemini]))
	}
}

func TestLLMError_Error(t *testing.T) {
	err := &LLMError{
		Provider: ProviderOpenAI,
		Model:    "gpt-4",
		Message:  "test error",
		Err:      errors.New("underlying error"),
	}

	msg := err.Error()
	if msg == "" {
		t.Error("Error message is empty")
	}
	if !contains(msg, "openai") {
		t.Errorf("Error message should contain 'openai', got '%s'", msg)
	}
}

func TestLLMError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &LLMError{
		Provider: ProviderOpenAI,
		Model:    "gpt-4",
		Message:  "test error",
		Err:      underlying,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlying {
		t.Errorf("Expected underlying error, got %v", unwrapped)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
