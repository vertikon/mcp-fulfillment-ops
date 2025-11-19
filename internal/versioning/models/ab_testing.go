package models

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ABTest represents an A/B test configuration
type ABTest struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	ModelID         string                 `json:"model_id"`
	VersionA        string                 `json:"version_a"` // version ID
	VersionB        string                 `json:"version_b"` // version ID
	TrafficSplit    TrafficSplit          `json:"traffic_split"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	Status          ABTestStatus           `json:"status"`
	Metrics         ABTestMetrics          `json:"metrics"`
	Criteria        PromotionCriteria      `json:"criteria"`
	CreatedAt       time.Time              `json:"created_at"`
	CreatedBy       string                 `json:"created_by"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// TrafficSplit represents traffic distribution
type TrafficSplit struct {
	VersionAPercent float64 `json:"version_a_percent"` // 0.0 to 1.0
	VersionBPercent  float64 `json:"version_b_percent"` // 0.0 to 1.0
}

// ABTestStatus represents test status
type ABTestStatus string

const (
	ABTestStatusDraft     ABTestStatus = "draft"
	ABTestStatusRunning   ABTestStatus = "running"
	ABTestStatusCompleted ABTestStatus = "completed"
	ABTestStatusStopped   ABTestStatus = "stopped"
)

// ABTestMetrics represents test metrics
type ABTestMetrics struct {
	VersionARequests int64   `json:"version_a_requests"`
	VersionBRequests int64   `json:"version_b_requests"`
	VersionAErrors   int64   `json:"version_a_errors"`
	VersionBErrors   int64   `json:"version_b_errors"`
	VersionALatency  float64 `json:"version_a_latency_ms"`
	VersionBLatency  float64 `json:"version_b_latency_ms"`
	VersionAScore    float64 `json:"version_a_score"`
	VersionBScore    float64 `json:"version_b_score"`
}

// PromotionCriteria represents criteria for promotion
type PromotionCriteria struct {
	MinRequests      int64   `json:"min_requests"`
	MinScore         float64 `json:"min_score"`
	MaxErrorRate     float64 `json:"max_error_rate"`
	MaxLatencyMs     float64 `json:"max_latency_ms"`
	MinImprovement   float64 `json:"min_improvement"` // percentage improvement required
}

// ABTesting interface for A/B testing operations
type ABTesting interface {
	// CreateTest creates a new A/B test
	CreateTest(ctx context.Context, test *ABTest) (*ABTest, error)
	
	// GetTest retrieves an A/B test
	GetTest(ctx context.Context, testID string) (*ABTest, error)
	
	// StartTest starts an A/B test
	StartTest(ctx context.Context, testID string) error
	
	// StopTest stops an A/B test
	StopTest(ctx context.Context, testID string) error
	
	// RecordRequest records a request for a version
	RecordRequest(ctx context.Context, testID string, versionID string, latency time.Duration, error bool) error
	
	// GetMetrics gets current metrics for a test
	GetMetrics(ctx context.Context, testID string) (*ABTestMetrics, error)
	
	// EvaluateTest evaluates if test criteria are met
	EvaluateTest(ctx context.Context, testID string) (*TestEvaluation, error)
	
	// SelectVersion selects which version to use based on traffic split
	SelectVersion(ctx context.Context, testID string) (string, error)
	
	// ListTests lists all tests for a model
	ListTests(ctx context.Context, modelID string) ([]*ABTest, error)
}

// TestEvaluation represents evaluation result
type TestEvaluation struct {
	TestID          string   `json:"test_id"`
	CanPromote      bool     `json:"can_promote"`
	PromoteVersion  string   `json:"promote_version,omitempty"`
	Reason          string   `json:"reason"`
	Metrics         ABTestMetrics `json:"metrics"`
	CriteriaMet     bool     `json:"criteria_met"`
}

// InMemoryABTesting implements ABTesting
type InMemoryABTesting struct {
	tests  map[string]*ABTest
	mu     sync.RWMutex
	logger *zap.Logger
	rand   *rand.Rand
}

// NewInMemoryABTesting creates a new A/B testing instance
func NewInMemoryABTesting() *InMemoryABTesting {
	return &InMemoryABTesting{
		tests:  make(map[string]*ABTest),
		logger: logger.WithContext(context.Background()),
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// CreateTest creates a new A/B test
func (ab *InMemoryABTesting) CreateTest(ctx context.Context, test *ABTest) (*ABTest, error) {
	logger := logger.WithContext(ctx)
	logger.Info("Creating A/B test", zap.String("name", test.Name))

	if test.ID == "" {
		test.ID = uuid.New().String()
	}

	if test.CreatedAt.IsZero() {
		test.CreatedAt = time.Now()
	}

	if test.CreatedBy == "" {
		test.CreatedBy = getCurrentUser(ctx)
	}

	if test.Status == "" {
		test.Status = ABTestStatusDraft
	}

	if test.Metadata == nil {
		test.Metadata = make(map[string]interface{})
	}

	// Validate traffic split
	total := test.TrafficSplit.VersionAPercent + test.TrafficSplit.VersionBPercent
	if total > 1.0 || total < 0.0 {
		return nil, fmt.Errorf("invalid traffic split: total must be between 0.0 and 1.0")
	}

	ab.mu.Lock()
	ab.tests[test.ID] = test
	ab.mu.Unlock()

	logger.Info("A/B test created", zap.String("test_id", test.ID))
	return test, nil
}

// GetTest retrieves an A/B test
func (ab *InMemoryABTesting) GetTest(ctx context.Context, testID string) (*ABTest, error) {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	test, exists := ab.tests[testID]
	if !exists {
		return nil, fmt.Errorf("test %s not found", testID)
	}

	return test, nil
}

// StartTest starts an A/B test
func (ab *InMemoryABTesting) StartTest(ctx context.Context, testID string) error {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	logger := logger.WithContext(ctx)

	test, exists := ab.tests[testID]
	if !exists {
		return fmt.Errorf("test %s not found", testID)
	}

	if test.Status != ABTestStatusDraft {
		return fmt.Errorf("test must be in draft status to start")
	}

	test.Status = ABTestStatusRunning
	test.StartTime = time.Now()

	logger.Info("A/B test started", zap.String("test_id", testID))
	return nil
}

// StopTest stops an A/B test
func (ab *InMemoryABTesting) StopTest(ctx context.Context, testID string) error {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	logger := logger.WithContext(ctx)

	test, exists := ab.tests[testID]
	if !exists {
		return fmt.Errorf("test %s not found", testID)
	}

	if test.Status != ABTestStatusRunning {
		return fmt.Errorf("test must be running to stop")
	}

	test.Status = ABTestStatusStopped
	now := time.Now()
	test.EndTime = &now

	logger.Info("A/B test stopped", zap.String("test_id", testID))
	return nil
}

// RecordRequest records a request for a version
func (ab *InMemoryABTesting) RecordRequest(ctx context.Context, testID string, versionID string, latency time.Duration, hasError bool) error {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	test, exists := ab.tests[testID]
	if !exists {
		return fmt.Errorf("test %s not found", testID)
	}

	if test.Status != ABTestStatusRunning {
		return fmt.Errorf("test is not running")
	}

	if versionID == test.VersionA {
		test.Metrics.VersionARequests++
		if hasError {
			test.Metrics.VersionAErrors++
		}
		test.Metrics.VersionALatency = (test.Metrics.VersionALatency*float64(test.Metrics.VersionARequests-1) + latency.Seconds()*1000) / float64(test.Metrics.VersionARequests)
	} else if versionID == test.VersionB {
		test.Metrics.VersionBRequests++
		if hasError {
			test.Metrics.VersionBErrors++
		}
		test.Metrics.VersionBLatency = (test.Metrics.VersionBLatency*float64(test.Metrics.VersionBRequests-1) + latency.Seconds()*1000) / float64(test.Metrics.VersionBRequests)
	} else {
		return fmt.Errorf("version %s not part of test", versionID)
	}

	return nil
}

// GetMetrics gets current metrics for a test
func (ab *InMemoryABTesting) GetMetrics(ctx context.Context, testID string) (*ABTestMetrics, error) {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	test, exists := ab.tests[testID]
	if !exists {
		return nil, fmt.Errorf("test %s not found", testID)
	}

	return &test.Metrics, nil
}

// EvaluateTest evaluates if test criteria are met
func (ab *InMemoryABTesting) EvaluateTest(ctx context.Context, testID string) (*TestEvaluation, error) {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	test, exists := ab.tests[testID]
	if !exists {
		return nil, fmt.Errorf("test %s not found", testID)
	}

	evaluation := &TestEvaluation{
		TestID:      testID,
		CanPromote:  false,
		Metrics:     test.Metrics,
		CriteriaMet: false,
	}

	// Check minimum requests
	totalRequests := test.Metrics.VersionARequests + test.Metrics.VersionBRequests
	if totalRequests < test.Criteria.MinRequests {
		evaluation.Reason = fmt.Sprintf("insufficient requests: %d < %d", totalRequests, test.Criteria.MinRequests)
		return evaluation, nil
	}

	// Calculate error rates
	errorRateA := float64(test.Metrics.VersionAErrors) / float64(test.Metrics.VersionARequests)
	errorRateB := float64(test.Metrics.VersionBErrors) / float64(test.Metrics.VersionBRequests)

	// Check error rates
	if errorRateA > test.Criteria.MaxErrorRate {
		evaluation.Reason = fmt.Sprintf("version A error rate too high: %.2f > %.2f", errorRateA, test.Criteria.MaxErrorRate)
		return evaluation, nil
	}

	if errorRateB > test.Criteria.MaxErrorRate {
		evaluation.Reason = fmt.Sprintf("version B error rate too high: %.2f > %.2f", errorRateB, test.Criteria.MaxErrorRate)
		return evaluation, nil
	}

	// Check latency
	if test.Metrics.VersionALatency > test.Criteria.MaxLatencyMs {
		evaluation.Reason = fmt.Sprintf("version A latency too high: %.2f > %.2f", test.Metrics.VersionALatency, test.Criteria.MaxLatencyMs)
		return evaluation, nil
	}

	if test.Metrics.VersionBLatency > test.Criteria.MaxLatencyMs {
		evaluation.Reason = fmt.Sprintf("version B latency too high: %.2f > %.2f", test.Metrics.VersionBLatency, test.Criteria.MaxLatencyMs)
		return evaluation, nil
	}

	// Compare scores
	scoreA := test.Metrics.VersionAScore
	scoreB := test.Metrics.VersionBScore

	if scoreA >= test.Criteria.MinScore && scoreB >= test.Criteria.MinScore {
		// Check improvement
		improvement := (scoreB - scoreA) / scoreA * 100
		if improvement >= test.Criteria.MinImprovement {
			evaluation.CanPromote = true
			evaluation.PromoteVersion = test.VersionB
			evaluation.Reason = fmt.Sprintf("version B shows %.2f%% improvement", improvement)
			evaluation.CriteriaMet = true
		} else if scoreA > scoreB {
			evaluation.CanPromote = true
			evaluation.PromoteVersion = test.VersionA
			evaluation.Reason = "version A performs better"
			evaluation.CriteriaMet = true
		} else {
			evaluation.Reason = fmt.Sprintf("improvement %.2f%% < required %.2f%%", improvement, test.Criteria.MinImprovement)
		}
	} else {
		evaluation.Reason = "scores below minimum threshold"
	}

	return evaluation, nil
}

// SelectVersion selects which version to use based on traffic split
func (ab *InMemoryABTesting) SelectVersion(ctx context.Context, testID string) (string, error) {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	test, exists := ab.tests[testID]
	if !exists {
		return "", fmt.Errorf("test %s not found", testID)
	}

	if test.Status != ABTestStatusRunning {
		return "", fmt.Errorf("test is not running")
	}

	// Random selection based on traffic split
	random := ab.rand.Float64()
	if random < test.TrafficSplit.VersionAPercent {
		return test.VersionA, nil
	}
	return test.VersionB, nil
}

// ListTests lists all tests for a model
func (ab *InMemoryABTesting) ListTests(ctx context.Context, modelID string) ([]*ABTest, error) {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	var tests []*ABTest
	for _, test := range ab.tests {
		if test.ModelID == modelID {
			tests = append(tests, test)
		}
	}

	return tests, nil
}

// getCurrentUser extracts user from context (placeholder)
func getCurrentUser(ctx context.Context) string {
	if userID := ctx.Value("user_id"); userID != nil {
		return userID.(string)
	}
	return "system"
}
