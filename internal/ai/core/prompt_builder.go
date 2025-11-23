package core

import (
	"fmt"
	"strings"
)

// PromptPolicy defines policies for prompt construction
type PromptPolicy struct {
	MaxContextLength int
	IncludeSystem    bool
	IncludeHistory   bool
	IncludeKnowledge bool
	Temperature      float64
}

// DefaultPromptPolicy returns default prompt policy
func DefaultPromptPolicy() *PromptPolicy {
	return &PromptPolicy{
		MaxContextLength: 4000,
		IncludeSystem:    true,
		IncludeHistory:   true,
		IncludeKnowledge: true,
		Temperature:      0.7,
	}
}

// PromptContext contains all context for building a prompt
type PromptContext struct {
	SystemPrompt string
	UserPrompt   string
	Knowledge    []string
	History      []Message
	Metadata     map[string]interface{}
	Policy       *PromptPolicy
}

// Message represents a message in conversation history
type Message struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// PromptBuilder builds prompts with context and policies
type PromptBuilder struct {
	policy *PromptPolicy
}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder(policy *PromptPolicy) *PromptBuilder {
	if policy == nil {
		policy = DefaultPromptPolicy()
	}
	return &PromptBuilder{
		policy: policy,
	}
}

// Build builds a complete prompt from context
func (pb *PromptBuilder) Build(ctx *PromptContext) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("prompt context cannot be nil")
	}

	policy := ctx.Policy
	if policy == nil {
		policy = pb.policy
	}

	var parts []string

	// Add system prompt if enabled
	if policy.IncludeSystem && ctx.SystemPrompt != "" {
		parts = append(parts, fmt.Sprintf("System: %s", ctx.SystemPrompt))
	}

	// Add knowledge context if enabled
	if policy.IncludeKnowledge && len(ctx.Knowledge) > 0 {
		knowledgeText := pb.buildKnowledgeSection(ctx.Knowledge)
		parts = append(parts, fmt.Sprintf("Knowledge Context:\n%s", knowledgeText))
	}

	// Add conversation history if enabled
	if policy.IncludeHistory && len(ctx.History) > 0 {
		historyText := pb.buildHistorySection(ctx.History)
		parts = append(parts, fmt.Sprintf("Conversation History:\n%s", historyText))
	}

	// Add user prompt
	if ctx.UserPrompt == "" {
		return "", fmt.Errorf("user prompt cannot be empty")
	}
	parts = append(parts, fmt.Sprintf("User: %s", ctx.UserPrompt))

	// Join all parts
	fullPrompt := strings.Join(parts, "\n\n")

	// Truncate if exceeds max length
	if policy.MaxContextLength > 0 && len(fullPrompt) > policy.MaxContextLength {
		fullPrompt = pb.truncatePrompt(fullPrompt, policy.MaxContextLength, ctx.UserPrompt)
	}

	return fullPrompt, nil
}

// buildKnowledgeSection formats knowledge context
func (pb *PromptBuilder) buildKnowledgeSection(knowledge []string) string {
	var sections []string
	for i, k := range knowledge {
		sections = append(sections, fmt.Sprintf("[%d] %s", i+1, k))
	}
	return strings.Join(sections, "\n")
}

// buildHistorySection formats conversation history
func (pb *PromptBuilder) buildHistorySection(history []Message) string {
	var messages []string
	for _, msg := range history {
		messages = append(messages, fmt.Sprintf("%s: %s", strings.Title(msg.Role), msg.Content))
	}
	return strings.Join(messages, "\n")
}

// truncatePrompt truncates prompt while preserving user prompt
func (pb *PromptBuilder) truncatePrompt(prompt string, maxLength int, userPrompt string) string {
	if len(prompt) <= maxLength {
		return prompt
	}

	// Reserve space for user prompt
	reservedLength := len(userPrompt) + 100 // buffer
	availableLength := maxLength - reservedLength

	if availableLength <= 0 {
		// If user prompt itself is too long, truncate it
		if len(userPrompt) > maxLength {
			return userPrompt[:maxLength-3] + "..."
		}
		return userPrompt
	}

	// Truncate from the beginning, keeping the end (user prompt)
	truncated := prompt[len(prompt)-availableLength:]
	return "..." + truncated
}

// BuildSystemPrompt builds a system prompt from template
func (pb *PromptBuilder) BuildSystemPrompt(template string, vars map[string]interface{}) string {
	result := template
	for key, value := range vars {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// EstimateTokens estimates token count (rough approximation)
func (pb *PromptBuilder) EstimateTokens(text string) int {
	// Rough approximation: 1 token ≈ 4 characters
	return len(text) / 4
}
