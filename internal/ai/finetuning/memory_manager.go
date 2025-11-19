package finetuning

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// MemorySource defines where to get training data from memory
type MemorySource struct {
	EpisodicMemory bool
	SemanticMemory bool
	MinImportance  float64
	TimeWindow     time.Duration
}

// MemoryManager generates training datasets from memory
type MemoryManager struct {
	memoryRetrieval interface {
		RetrieveRecent(ctx context.Context, sessionID string, window time.Duration, limit int) ([]*entities.MemoryEvent, error)
		RetrieveByImportance(ctx context.Context, sessionID string, minImportance float64) ([]*entities.EpisodicMemory, error)
		RetrieveSemanticByConcept(ctx context.Context, concept string, limit int) ([]*entities.SemanticMemory, error)
	}
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(memoryRetrieval interface {
	RetrieveRecent(ctx context.Context, sessionID string, window time.Duration, limit int) ([]*entities.MemoryEvent, error)
	RetrieveByImportance(ctx context.Context, sessionID string, minImportance float64) ([]*entities.EpisodicMemory, error)
	RetrieveSemanticByConcept(ctx context.Context, concept string, limit int) ([]*entities.SemanticMemory, error)
}) *MemoryManager {
	return &MemoryManager{
		memoryRetrieval: memoryRetrieval,
	}
}

// GenerateDataset generates a training dataset from memory
func (mm *MemoryManager) GenerateDataset(ctx context.Context, dataset *entities.Dataset) (string, error) {
	// This would generate a dataset file from memory
	// For now, return the dataset file path
	return dataset.FilePath(), nil
}

// GenerateDatasetFromMemory generates dataset from memory sources
func (mm *MemoryManager) GenerateDatasetFromMemory(ctx context.Context, sessionIDs []string, source *MemorySource) ([]TrainingExample, error) {
	examples := make([]TrainingExample, 0)

	for _, sessionID := range sessionIDs {
		// Get episodic memories if enabled
		if source.EpisodicMemory {
			episodic, err := mm.memoryRetrieval.RetrieveByImportance(ctx, sessionID, source.MinImportance)
			if err == nil {
				for _, mem := range episodic {
					example := TrainingExample{
						Input:  mem.Content(),
						Output: "", // Would need to determine output from context
					}
					examples = append(examples, example)
				}
			}
		}

		// Get semantic memories if enabled
		if source.SemanticMemory {
			// Get recent semantic memories
			// This is simplified - in production would query by concepts
			semantic, err := mm.memoryRetrieval.RetrieveSemanticByConcept(ctx, "", 100)
			if err == nil {
				for _, mem := range semantic {
					example := TrainingExample{
						Input:  mem.Content(),
						Output: "", // Would need to determine output
					}
					examples = append(examples, example)
				}
			}
		}
	}

	return examples, nil
}

// SaveDatasetToFile saves dataset to file format (JSONL)
func (mm *MemoryManager) SaveDatasetToFile(examples []TrainingExample, filePath string) error {
	if len(examples) == 0 {
		return fmt.Errorf("examples cannot be empty")
	}
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Open file for writing
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write each example as a JSONL line
	for _, example := range examples {
		jsonl, err := example.ToJSONL()
		if err != nil {
			return fmt.Errorf("failed to convert example to JSONL: %w", err)
		}

		if _, err := file.WriteString(jsonl + "\n"); err != nil {
			return fmt.Errorf("failed to write example: %w", err)
		}
	}

	return nil
}

// TrainingExample represents a training example
type TrainingExample struct {
	Input  string
	Output string
}

// ToJSONL converts training example to JSONL format
func (te *TrainingExample) ToJSONL() (string, error) {
	data, err := json.Marshal(te)
	if err != nil {
		return "", fmt.Errorf("failed to marshal example: %w", err)
	}
	return string(data), nil
}

// ParseDatasetFile parses a dataset file
func (mm *MemoryManager) ParseDatasetFile(filePath string) ([]TrainingExample, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Open file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	examples := make([]TrainingExample, 0)
	scanner := bufio.NewScanner(file)

	// Read each line (JSONL format)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue // Skip empty lines
		}

		var example TrainingExample
		if err := json.Unmarshal([]byte(line), &example); err != nil {
			return nil, fmt.Errorf("failed to parse JSONL line: %w", err)
		}

		examples = append(examples, example)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return examples, nil
}
