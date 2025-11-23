package core

import (
	"strings"
	"testing"
)

func TestNewPromptBuilder(t *testing.T) {
	pb := NewPromptBuilder(nil)
	if pb == nil {
		t.Fatal("NewPromptBuilder returned nil")
	}
	if pb.policy == nil {
		t.Error("Default policy is nil")
	}
}

func TestNewPromptBuilder_WithPolicy(t *testing.T) {
	policy := &PromptPolicy{
		MaxContextLength: 2000,
		IncludeSystem:    false,
		IncludeHistory:   false,
		IncludeKnowledge: false,
		Temperature:      0.5,
	}

	pb := NewPromptBuilder(policy)
	if pb.policy != policy {
		t.Error("Policy not set correctly")
	}
}

func TestPromptBuilder_Build_Basic(t *testing.T) {
	pb := NewPromptBuilder(nil)

	ctx := &PromptContext{
		UserPrompt: "Hello, world!",
		Policy:     DefaultPromptPolicy(),
	}

	prompt, err := pb.Build(ctx)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "Hello, world!") {
		t.Errorf("Prompt should contain user prompt, got: %s", prompt)
	}
}

func TestPromptBuilder_Build_WithSystemPrompt(t *testing.T) {
	pb := NewPromptBuilder(nil)

	ctx := &PromptContext{
		SystemPrompt: "You are a helpful assistant.",
		UserPrompt:   "Hello",
		Policy: &PromptPolicy{
			MaxContextLength: 4000,
			IncludeSystem:    true,
			IncludeHistory:   false,
			IncludeKnowledge: false,
		},
	}

	prompt, err := pb.Build(ctx)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "You are a helpful assistant") {
		t.Errorf("Prompt should contain system prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Hello") {
		t.Errorf("Prompt should contain user prompt, got: %s", prompt)
	}
}

func TestPromptBuilder_Build_WithKnowledge(t *testing.T) {
	pb := NewPromptBuilder(nil)

	ctx := &PromptContext{
		UserPrompt: "What is AI?",
		Knowledge:  []string{"AI is artificial intelligence", "AI uses machine learning"},
		Policy: &PromptPolicy{
			MaxContextLength: 4000,
			IncludeSystem:    false,
			IncludeHistory:   false,
			IncludeKnowledge: true,
		},
	}

	prompt, err := pb.Build(ctx)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "AI is artificial intelligence") {
		t.Errorf("Prompt should contain knowledge, got: %s", prompt)
	}
}

func TestPromptBuilder_Build_WithHistory(t *testing.T) {
	pb := NewPromptBuilder(nil)

	ctx := &PromptContext{
		UserPrompt: "Continue",
		History: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
		Policy: &PromptPolicy{
			MaxContextLength: 4000,
			IncludeSystem:    false,
			IncludeHistory:   true,
			IncludeKnowledge: false,
		},
	}

	prompt, err := pb.Build(ctx)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "Hello") {
		t.Errorf("Prompt should contain history, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Hi there!") {
		t.Errorf("Prompt should contain history, got: %s", prompt)
	}
}

func TestPromptBuilder_Build_Truncation(t *testing.T) {
	pb := NewPromptBuilder(nil)

	longPrompt := strings.Repeat("a", 5000)
	ctx := &PromptContext{
		UserPrompt: longPrompt,
		Policy: &PromptPolicy{
			MaxContextLength: 1000,
			IncludeSystem:    false,
			IncludeHistory:   false,
			IncludeKnowledge: false,
		},
	}

	prompt, err := pb.Build(ctx)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(prompt) > 1000 {
		t.Errorf("Prompt should be truncated to 1000 chars, got %d", len(prompt))
	}
}

func TestPromptBuilder_Build_EmptyUserPrompt(t *testing.T) {
	pb := NewPromptBuilder(nil)

	ctx := &PromptContext{
		UserPrompt: "",
		Policy:     DefaultPromptPolicy(),
	}

	_, err := pb.Build(ctx)
	if err == nil {
		t.Error("Expected error for empty user prompt")
	}
}

func TestPromptBuilder_Build_NilContext(t *testing.T) {
	pb := NewPromptBuilder(nil)

	_, err := pb.Build(nil)
	if err == nil {
		t.Error("Expected error for nil context")
	}
}

func TestPromptBuilder_BuildSystemPrompt(t *testing.T) {
	pb := NewPromptBuilder(nil)

	template := "Hello {{name}}, you are {{role}}"
	vars := map[string]interface{}{
		"name": "Alice",
		"role": "assistant",
	}

	result := pb.BuildSystemPrompt(template, vars)

	if !strings.Contains(result, "Alice") {
		t.Errorf("Expected 'Alice' in result, got: %s", result)
	}
	if !strings.Contains(result, "assistant") {
		t.Errorf("Expected 'assistant' in result, got: %s", result)
	}
	if strings.Contains(result, "{{") {
		t.Errorf("Template variables not replaced, got: %s", result)
	}
}

func TestPromptBuilder_EstimateTokens(t *testing.T) {
	pb := NewPromptBuilder(nil)

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty",
			text:     "",
			expected: 0,
		},
		{
			name:     "short",
			text:     "Hello",
			expected: 1, // 5 chars / 4 = 1
		},
		{
			name:     "medium",
			text:     strings.Repeat("a", 100),
			expected: 25, // 100 / 4 = 25
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pb.EstimateTokens(tt.text)
			if result != tt.expected {
				t.Errorf("Expected %d tokens, got %d", tt.expected, result)
			}
		})
	}
}

func TestDefaultPromptPolicy(t *testing.T) {
	policy := DefaultPromptPolicy()

	if policy == nil {
		t.Fatal("DefaultPromptPolicy returned nil")
	}
	if policy.MaxContextLength != 4000 {
		t.Errorf("Expected MaxContextLength 4000, got %d", policy.MaxContextLength)
	}
	if !policy.IncludeSystem {
		t.Error("IncludeSystem should be true")
	}
	if !policy.IncludeHistory {
		t.Error("IncludeHistory should be true")
	}
	if !policy.IncludeKnowledge {
		t.Error("IncludeKnowledge should be true")
	}
	if policy.Temperature != 0.7 {
		t.Errorf("Expected Temperature 0.7, got %f", policy.Temperature)
	}
}
