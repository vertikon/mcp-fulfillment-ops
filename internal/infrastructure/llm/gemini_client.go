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

// geminiClient implements LLMClient using Google Gemini API
type geminiClient struct {
	apiKey  string
	baseURL string
	timeout time.Duration
	client  *http.Client
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(apiKey string, baseURL string, timeout time.Duration) LLMClient {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1"
	}
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &geminiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Complete generates a completion from a prompt
func (c *geminiClient) Complete(ctx context.Context, prompt string, options *CompletionOptions) (*Completion, error) {
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	if options == nil {
		options = &CompletionOptions{
			Model:       "gemini-pro",
			Temperature: 0.7,
			MaxTokens:   1000,
		}
	}

	logger.Info("Generating Gemini completion",
		zap.String("model", options.Model),
		zap.Int("prompt_length", len(prompt)),
	)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
	}

	if options.Temperature > 0 {
		payload["generationConfig"] = map[string]interface{}{
			"temperature":     options.Temperature,
			"maxOutputTokens": options.MaxTokens,
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, options.Model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	return &Completion{
		Text:         result.Candidates[0].Content.Parts[0].Text,
		Model:        options.Model,
		FinishReason: result.Candidates[0].FinishReason,
		Usage: &Usage{
			PromptTokens:     result.UsageMetadata.PromptTokenCount,
			CompletionTokens: result.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      result.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

// Chat generates a chat completion
func (c *geminiClient) Chat(ctx context.Context, messages []*Message, options *CompletionOptions) (*Completion, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}

	if options == nil {
		options = &CompletionOptions{
			Model:       "gemini-pro",
			Temperature: 0.7,
			MaxTokens:   1000,
		}
	}

	logger.Info("Generating Gemini chat completion",
		zap.String("model", options.Model),
		zap.Int("message_count", len(messages)),
	)

	// Convert messages to Gemini format
	contents := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		contents[i] = map[string]interface{}{
			"role": msg.Role,
			"parts": []map[string]interface{}{
				{"text": msg.Content},
			},
		}
	}

	payload := map[string]interface{}{
		"contents": contents,
	}

	if options.Temperature > 0 {
		payload["generationConfig"] = map[string]interface{}{
			"temperature":     options.Temperature,
			"maxOutputTokens": options.MaxTokens,
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, options.Model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	return &Completion{
		Text:         result.Candidates[0].Content.Parts[0].Text,
		Model:        options.Model,
		FinishReason: result.Candidates[0].FinishReason,
		Usage: &Usage{
			PromptTokens:     result.UsageMetadata.PromptTokenCount,
			CompletionTokens: result.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      result.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

// Embed generates embeddings for text
func (c *geminiClient) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	logger.Info("Generating Gemini embeddings",
		zap.Int("text_count", len(texts)),
	)

	// Gemini embedding API
	embeddings := make([][]float64, len(texts))
	for i, text := range texts {
		payload := map[string]interface{}{
			"model": "models/embedding-001",
			"content": map[string]interface{}{
				"parts": []map[string]interface{}{
					{"text": text},
				},
			},
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}

		url := fmt.Sprintf("%s/models/embedding-001:embedContent?key=%s", c.baseURL, c.apiKey)
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

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
			Embedding struct {
				Values []float64 `json:"values"`
			} `json:"embedding"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		embeddings[i] = result.Embedding.Values
	}

	return embeddings, nil
}
