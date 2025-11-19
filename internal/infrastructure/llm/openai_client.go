// Package llm provides LLM client implementations
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// LLMClient provides LLM operations
type LLMClient interface {
	// Complete generates a completion from a prompt
	Complete(ctx context.Context, prompt string, options *CompletionOptions) (*Completion, error)

	// Chat generates a chat completion
	Chat(ctx context.Context, messages []*Message, options *CompletionOptions) (*Completion, error)

	// Embed generates embeddings for text
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}

// Message represents a chat message
type Message struct {
	Role    string
	Content string
}

// CompletionOptions represents completion options
type CompletionOptions struct {
	Model       string
	Temperature float64
	MaxTokens   int
	TopP        float64
	Stream      bool
}

// Completion represents a completion result
type Completion struct {
	Text        string
	Model       string
	Usage       *Usage
	FinishReason string
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// openaiClient implements LLMClient using OpenAI API
type openaiClient struct {
	apiKey  string
	baseURL string
	timeout time.Duration
	client  *http.Client
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey string, baseURL string, timeout time.Duration) LLMClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &openaiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Complete generates a completion from a prompt
func (c *openaiClient) Complete(ctx context.Context, prompt string, options *CompletionOptions) (*Completion, error) {
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	if options == nil {
		options = &CompletionOptions{
			Model:       "gpt-3.5-turbo",
			Temperature: 0.7,
			MaxTokens:   1000,
		}
	}

	logger.Info("Generating OpenAI completion",
		zap.String("model", options.Model),
		zap.Int("prompt_length", len(prompt)),
	)

	payload := map[string]interface{}{
		"model":       options.Model,
		"prompt":      prompt,
		"temperature": options.Temperature,
		"max_tokens":  options.MaxTokens,
	}

	if options.TopP > 0 {
		payload["top_p"] = options.TopP
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/completions", c.baseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID      string `json:"id"`
		Choices []struct {
			Text         string `json:"text"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &Completion{
		Text:        result.Choices[0].Text,
		Model:       result.Model,
		FinishReason: result.Choices[0].FinishReason,
		Usage: &Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}, nil
}

// Chat generates a chat completion
func (c *openaiClient) Chat(ctx context.Context, messages []*Message, options *CompletionOptions) (*Completion, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}

	if options == nil {
		options = &CompletionOptions{
			Model:       "gpt-3.5-turbo",
			Temperature: 0.7,
			MaxTokens:   1000,
		}
	}

	logger.Info("Generating OpenAI chat completion",
		zap.String("model", options.Model),
		zap.Int("message_count", len(messages)),
	)

	// Convert messages to API format
	apiMessages := make([]map[string]string, len(messages))
	for i, msg := range messages {
		apiMessages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	payload := map[string]interface{}{
		"model":       options.Model,
		"messages":    apiMessages,
		"temperature": options.Temperature,
		"max_tokens":  options.MaxTokens,
	}

	if options.TopP > 0 {
		payload["top_p"] = options.TopP
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", c.baseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &Completion{
		Text:        result.Choices[0].Message.Content,
		Model:       result.Model,
		FinishReason: result.Choices[0].FinishReason,
		Usage: &Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}, nil
}

// Embed generates embeddings for text
func (c *openaiClient) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	logger.Info("Generating OpenAI embeddings",
		zap.Int("text_count", len(texts)),
	)

	payload := map[string]interface{}{
		"model": "text-embedding-ada-002",
		"input": texts,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/embeddings", c.baseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	embeddings := make([][]float64, len(result.Data))
	for i, item := range result.Data {
		embeddings[i] = item.Embedding
	}

	return embeddings, nil
}
