package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ProjectionType represents type of projection
type ProjectionType string

const (
	ProjectionTypeAggregation  ProjectionType = "aggregation"
	ProjectionTypeState        ProjectionType = "state"
	ProjectionTypeStatistics   ProjectionType = "statistics"
	ProjectionTypeMaterialized ProjectionType = "materialized"
	ProjectionTypeCustom       ProjectionType = "custom"
)

// Projection represents a projection of events
type Projection struct {
	ID            string                 `json:"id"`
	Type          ProjectionType         `json:"type"`
	Name          string                 `json:"name"`
	AggregateID   string                 `json:"aggregate_id,omitempty"`
	AggregateType string                 `json:"aggregate_type,omitempty"`
	EventType     []EventType            `json:"event_types"`
	Handler       ProjectionHandler      `json:"-"`
	Data          interface{}            `json:"data"`
	Metadata      map[string]interface{} `json:"metadata"`
	LastProcessed *time.Time             `json:"last_processed,omitempty"`
	Version       int64                  `json:"version"`
	IsActive      bool                   `json:"is_active"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// ProjectionHandler defines it handler function for projections
type ProjectionHandler interface {
	Project(ctx context.Context, event *Event, projection *Projection) (interface{}, error)
	CanHandle(event *Event) bool
	GetHandlerType() string
}

// EventProjection interface for managing projections
type EventProjection interface {
	// Projection operations
	CreateProjection(ctx context.Context, projection *Projection) error
	UpdateProjection(ctx context.Context, projection *Projection) error
	DeleteProjection(ctx context.Context, projectionID string) error
	GetProjection(ctx context.Context, projectionID string) (*Projection, error)
	ListProjections(ctx context.Context, filter *ProjectionFilter) ([]*Projection, error)

	// Projection execution
	ProcessEvent(ctx context.Context, event *Event) error
	ProcessEvents(ctx context.Context, events []*Event) error
	RebuildProjection(ctx context.Context, projectionID string) error
	RebuildAllProjections(ctx context.Context) error

	// Projection state
	GetProjectionState(ctx context.Context, projectionID string) (*ProjectionState, error)
	ResetProjection(ctx context.Context, projectionID string) error

	// Statistics and monitoring
	GetProjectionStats(ctx context.Context) (*ProjectionStats, error)
	GetProjectionMetrics(ctx context.Context, projectionID string) (*ProjectionMetrics, error)

	// Background processing
	StartBackgroundProcessor(ctx context.Context) error
	StopBackgroundProcessor() error
}

// ProjectionFilter represents a filter for projections
type ProjectionFilter struct {
	Type          ProjectionType `json:"type,omitempty"`
	AggregateID   string         `json:"aggregate_id,omitempty"`
	AggregateType string         `json:"aggregate_type,omitempty"`
	EventType     EventType      `json:"event_type,omitempty"`
	IsActive      *bool          `json:"is_active,omitempty"`
	Limit         int            `json:"limit,omitempty"`
	Offset        int            `json:"offset,omitempty"`
}

// ProjectionState represents a current state of a projection
type ProjectionState struct {
	ProjectionID    string                 `json:"projection_id"`
	LastEventID     string                 `json:"last_event_id"`
	LastVersion     int64                  `json:"last_version"`
	LastProcessed   time.Time              `json:"last_processed"`
	EventsProcessed int64                  `json:"events_processed"`
	ErrorsCount     int64                  `json:"errors_count"`
	LastError       *time.Time             `json:"last_error,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	StateData       map[string]interface{} `json:"state_data"`
}

// ProjectionStats represents projection statistics
type ProjectionStats struct {
	TotalProjections  int64            `json:"total_projections"`
	ActiveProjections int64            `json:"active_projections"`
	EventsProcessed   int64            `json:"events_processed"`
	ProjectionsByType map[string]int64 `json:"projections_by_type"`
	ProcessingTime    time.Duration    `json:"average_processing_time"`
	ErrorRate         float64          `json:"error_rate"`
	LastProcessed     *time.Time       `json:"last_processed"`
	BackgroundWorkers int              `json:"background_workers"`
}

// ProjectionMetrics represents metrics for a specific projection
type ProjectionMetrics struct {
	ProjectionID    string        `json:"projection_id"`
	EventsProcessed int64         `json:"events_processed"`
	EventsPerSecond float64       `json:"events_per_second"`
	AverageLatency  time.Duration `json:"average_latency"`
	ErrorRate       float64       `json:"error_rate"`
	LastProcessed   *time.Time    `json:"last_processed"`
	CPUUsage        float64       `json:"cpu_usage"`
	MemoryUsage     int64         `json:"memory_usage"`
	Health          string        `json:"health"`
}

// ProjectionConfig represents projection configuration
type ProjectionConfig struct {
	MaxProjections        int           `json:"max_projections"`
	BackgroundWorkers     int           `json:"background_workers"`
	BatchSize             int           `json:"batch_size"`
	BatchTimeout          time.Duration `json:"batch_timeout"`
	RetryAttempts         int           `json:"retry_attempts"`
	RetryDelay            time.Duration `json:"retry_delay"`
	MaxRetriesPerEvent    int           `json:"max_retries_per_event"`
	StateUpdateInterval   time.Duration `json:"state_update_interval"`
	MetricsUpdateInterval time.Duration `json:"metrics_update_interval"`
	EnableAutoRebuild     bool          `json:"enable_auto_rebuild"`
	RebuildInterval       time.Duration `json:"rebuild_interval"`
}

// DefaultProjectionConfig returns default projection configuration
func DefaultProjectionConfig() *ProjectionConfig {
	return &ProjectionConfig{
		MaxProjections:        10000, // Aumentado de 1000 para 10000
		BackgroundWorkers:     20,    // Aumentado de 5 para 20
		BatchSize:             1000,  // Aumentado de 100 para 1000
		BatchTimeout:          time.Second,
		RetryAttempts:         3,
		RetryDelay:            time.Millisecond * 100,
		MaxRetriesPerEvent:    5,
		StateUpdateInterval:   time.Second * 10,
		MetricsUpdateInterval: time.Second * 30,
		EnableAutoRebuild:     false,
		RebuildInterval:       time.Hour,
	}
}

// EventProjectionImpl implements EventProjection interface
type EventProjectionImpl struct {
	config      *ProjectionConfig
	eventStore  EventStore
	projections map[string]*Projection
	states      map[string]*ProjectionState
	metrics     map[string]*ProjectionMetrics
	handlers    map[string]ProjectionHandler
	mu          sync.RWMutex
	logger      *zap.Logger
	stats       *ProjectionStats
	ctx         context.Context
	cancel      context.CancelFunc
	eventCh     chan *Event
	batchCh     chan []*Event
	workerPool  []chan *Event
}

// NewEventProjection creates a new event projection implementation
func NewEventProjection(config *ProjectionConfig, eventStore EventStore) *EventProjectionImpl {
	if config == nil {
		config = DefaultProjectionConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	projection := &EventProjectionImpl{
		config:      config,
		eventStore:  eventStore,
		projections: make(map[string]*Projection),
		states:      make(map[string]*ProjectionState),
		metrics:     make(map[string]*ProjectionMetrics),
		handlers:    make(map[string]ProjectionHandler),
		eventCh:     make(chan *Event, config.BatchSize*config.BackgroundWorkers),
		batchCh:     make(chan []*Event, config.MaxProjections),
		workerPool:  make([]chan *Event, config.BackgroundWorkers),
		stats: &ProjectionStats{
			ProjectionsByType: make(map[string]int64),
		},
		ctx:    ctx,
		cancel: cancel,
		logger: logger.Get(),
	}

	// Initialize worker pool
	for i := 0; i < config.BackgroundWorkers; i++ {
		projection.workerPool[i] = make(chan *Event, config.BatchSize)
		go projection.worker(i, projection.workerPool[i])
	}

	// Start background processes
	go projection.batchProcessor()
	go projection.eventDistributor()

	projection.logger.Info("Event projection initialized",
		zap.Int("max_projections", config.MaxProjections),
		zap.Int("background_workers", config.BackgroundWorkers),
		zap.Int("batch_size", config.BatchSize))

	return projection
}

// CreateProjection creates a new projection
func (ep *EventProjectionImpl) CreateProjection(ctx context.Context, projection *Projection) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if len(ep.projections) >= ep.config.MaxProjections {
		return fmt.Errorf("maximum projections limit reached: %d", ep.config.MaxProjections)
	}

	// Validate projection
	if err := ep.validateProjection(projection); err != nil {
		return fmt.Errorf("projection validation failed: %w", err)
	}

	// Set defaults
	if projection.CreatedAt.IsZero() {
		projection.CreatedAt = time.Now()
	}
	projection.UpdatedAt = time.Now()

	// Initialize state and metrics
	ep.projections[projection.ID] = projection
	ep.states[projection.ID] = &ProjectionState{
		ProjectionID:    projection.ID,
		EventsProcessed: 0,
		ErrorsCount:     0,
		StateData:       make(map[string]interface{}),
	}

	ep.metrics[projection.ID] = &ProjectionMetrics{
		ProjectionID:    projection.ID,
		EventsProcessed: 0,
		Health:          "healthy",
	}

	// Update statistics
	ep.stats.TotalProjections++
	ep.stats.ActiveProjections++
	ep.stats.ProjectionsByType[string(projection.Type)]++

	ep.logger.Info("Projection created",
		zap.String("projection_id", projection.ID),
		zap.String("type", string(projection.Type)),
		zap.String("name", projection.Name))

	return nil
}

// UpdateProjection updates an existing projection
func (ep *EventProjectionImpl) UpdateProjection(ctx context.Context, projection *Projection) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	existing, exists := ep.projections[projection.ID]
	if !exists {
		return fmt.Errorf("projection not found: %s", projection.ID)
	}

	// Update projection
	projection.CreatedAt = existing.CreatedAt
	projection.UpdatedAt = time.Now()
	ep.projections[projection.ID] = projection

	ep.logger.Info("Projection updated",
		zap.String("projection_id", projection.ID),
		zap.String("type", string(projection.Type)))

	return nil
}

// DeleteProjection deletes a projection
func (ep *EventProjectionImpl) DeleteProjection(ctx context.Context, projectionID string) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	projection, exists := ep.projections[projectionID]
	if !exists {
		return fmt.Errorf("projection not found: %s", projectionID)
	}

	// Clean up resources
	delete(ep.projections, projectionID)
	delete(ep.states, projectionID)
	delete(ep.metrics, projectionID)

	// Update statistics
	ep.stats.TotalProjections--
	if projection.IsActive {
		ep.stats.ActiveProjections--
	}

	ep.logger.Info("Projection deleted",
		zap.String("projection_id", projectionID))

	return nil
}

// GetProjection retrieves a projection by ID
func (ep *EventProjectionImpl) GetProjection(ctx context.Context, projectionID string) (*Projection, error) {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	projection, exists := ep.projections[projectionID]
	if !exists {
		return nil, fmt.Errorf("projection not found: %s", projectionID)
	}

	// Return a copy
	copyProjection := *projection
	return &copyProjection, nil
}

// ListProjections lists projections with optional filtering
func (ep *EventProjectionImpl) ListProjections(ctx context.Context, filter *ProjectionFilter) ([]*Projection, error) {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	var projections []*Projection

	for _, projection := range ep.projections {
		// Apply filter
		if filter != nil {
			if filter.Type != "" && projection.Type != filter.Type {
				continue
			}
			if filter.AggregateID != "" && projection.AggregateID != filter.AggregateID {
				continue
			}
			if filter.AggregateType != "" && projection.AggregateType != filter.AggregateType {
				continue
			}
			if filter.IsActive != nil && projection.IsActive != *filter.IsActive {
				continue
			}
		}

		// Return a copy
		copyProjection := *projection
		projections = append(projections, &copyProjection)
	}

	// Apply limit and offset
	if filter != nil && (filter.Limit > 0 || filter.Offset > 0) {
		start := filter.Offset
		end := start + filter.Limit
		if end > len(projections) {
			end = len(projections)
		}
		if start > len(projections) {
			return []*Projection{}, nil
		}
		projections = projections[start:end]
	}

	return projections, nil
}

// ProcessEvent processes a single event through all applicable projections
func (ep *EventProjectionImpl) ProcessEvent(ctx context.Context, event *Event) error {
	// Send event to background processor
	select {
	case ep.eventCh <- event:
	default:
		return fmt.Errorf("event channel full, dropping event")
	}

	return nil
}

// ProcessEvents processes multiple events
func (ep *EventProjectionImpl) ProcessEvents(ctx context.Context, events []*Event) error {
	// Send events to background processor
	for _, event := range events {
		if err := ep.ProcessEvent(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

// RebuildProjection rebuilds a projection from scratch
func (ep *EventProjectionImpl) RebuildProjection(ctx context.Context, projectionID string) error {
	ep.mu.Lock()
	projection, exists := ep.projections[projectionID]
	ep.mu.Unlock()

	if !exists {
		return fmt.Errorf("projection not found: %s", projectionID)
	}

	ep.logger.Info("Rebuilding projection",
		zap.String("projection_id", projectionID))

	startTime := time.Now()

	// Reset projection state
	if err := ep.ResetProjection(ctx, projectionID); err != nil {
		return fmt.Errorf("failed to reset projection: %w", err)
	}

	// Get all events for aggregate
	var allEvents []*Event
	var err error

	if projection.AggregateID != "" {
		allEvents, err = ep.eventStore.GetAllEvents(ctx, projection.AggregateID)
	} else {
		// For projections without specific aggregate, get events by type
		for _, eventType := range projection.EventType {
			events, err := ep.eventStore.GetEventsByType(ctx, eventType, 10000)
			if err != nil {
				return fmt.Errorf("failed to get events by type: %w", err)
			}
			allEvents = append(allEvents, events...)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to get events for rebuild: %w", err)
	}

	// Process events
	for _, event := range allEvents {
		if err := ep.processEventForProjection(ctx, event, projection); err != nil {
			ep.logger.Error("Failed to process event during rebuild",
				zap.String("projection_id", projectionID),
				zap.String("event_id", event.ID),
				zap.Error(err))
			continue
		}
	}

	duration := time.Since(startTime)
	ep.logger.Info("Projection rebuilt",
		zap.String("projection_id", projectionID),
		zap.Int("events_processed", len(allEvents)),
		zap.Duration("duration", duration))

	return nil
}

// RebuildAllProjections rebuilds all projections
func (ep *EventProjectionImpl) RebuildAllProjections(ctx context.Context) error {
	ep.mu.RLock()
	projectionIDs := make([]string, 0, len(ep.projections))
	for id := range ep.projections {
		projectionIDs = append(projectionIDs, id)
	}
	ep.mu.RUnlock()

	for _, projectionID := range projectionIDs {
		if err := ep.RebuildProjection(ctx, projectionID); err != nil {
			ep.logger.Error("Failed to rebuild projection",
				zap.String("projection_id", projectionID),
				zap.Error(err))
		}
	}

	return nil
}

// GetProjectionState returns a current state of a projection
func (ep *EventProjectionImpl) GetProjectionState(ctx context.Context, projectionID string) (*ProjectionState, error) {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	state, exists := ep.states[projectionID]
	if !exists {
		return nil, fmt.Errorf("projection state not found: %s", projectionID)
	}

	// Return a copy
	copyState := *state
	copyState.StateData = make(map[string]interface{})
	for k, v := range state.StateData {
		copyState.StateData[k] = v
	}

	return &copyState, nil
}

// ResetProjection resets a projection to its initial state
func (ep *EventProjectionImpl) ResetProjection(ctx context.Context, projectionID string) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	state, exists := ep.states[projectionID]
	if !exists {
		return fmt.Errorf("projection state not found: %s", projectionID)
	}

	// Reset state
	state.LastEventID = ""
	state.LastVersion = 0
	state.LastProcessed = time.Time{}
	state.EventsProcessed = 0
	state.ErrorsCount = 0
	state.StateData = make(map[string]interface{})

	// Reset metrics
	if metrics, exists := ep.metrics[projectionID]; exists {
		metrics.EventsProcessed = 0
		metrics.AverageLatency = 0
		metrics.ErrorRate = 0
		metrics.Health = "healthy"
	}

	ep.logger.Info("Projection reset",
		zap.String("projection_id", projectionID))

	return nil
}

// GetProjectionStats returns projection statistics
func (ep *EventProjectionImpl) GetProjectionStats(ctx context.Context) (*ProjectionStats, error) {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	// Update worker count
	ep.stats.BackgroundWorkers = len(ep.workerPool)

	// Return a copy of stats
	stats := *ep.stats
	stats.ProjectionsByType = make(map[string]int64)
	for k, v := range ep.stats.ProjectionsByType {
		stats.ProjectionsByType[k] = v
	}

	return &stats, nil
}

// GetProjectionMetrics returns metrics for a specific projection
func (ep *EventProjectionImpl) GetProjectionMetrics(ctx context.Context, projectionID string) (*ProjectionMetrics, error) {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	metrics, exists := ep.metrics[projectionID]
	if !exists {
		return nil, fmt.Errorf("projection metrics not found: %s", projectionID)
	}

	// Return a copy
	copyMetrics := *metrics
	return &copyMetrics, nil
}

// StartBackgroundProcessor starts background processing
func (ep *EventProjectionImpl) StartBackgroundProcessor(ctx context.Context) error {
	ep.logger.Info("Background processor already started")
	return nil
}

// StopBackgroundProcessor stops background processing
func (ep *EventProjectionImpl) StopBackgroundProcessor() error {
	ep.cancel()
	close(ep.eventCh)
	close(ep.batchCh)

	for _, worker := range ep.workerPool {
		close(worker)
	}

	ep.logger.Info("Background processor stopped")
	return nil
}

// Private helper methods

func (ep *EventProjectionImpl) validateProjection(projection *Projection) error {
	if projection.ID == "" {
		return fmt.Errorf("projection ID is required")
	}
	if projection.Name == "" {
		return fmt.Errorf("projection name is required")
	}
	if projection.Type == "" {
		return fmt.Errorf("projection type is required")
	}
	if len(projection.EventType) == 0 {
		return fmt.Errorf("at least one event type is required")
	}
	if projection.Handler == nil {
		return fmt.Errorf("projection handler is required")
	}

	return nil
}

func (ep *EventProjectionImpl) eventDistributor() {
	for {
		select {
		case <-ep.ctx.Done():
			return
		case event := <-ep.eventCh:
			// Distribute event to workers (round-robin)
			workerIndex := int(time.Now().UnixNano()) % len(ep.workerPool)
			select {
			case ep.workerPool[workerIndex] <- event:
			default:
				ep.logger.Warn("Worker channel full, dropping event",
					zap.String("event_id", event.ID))
			}
		}
	}
}

func (ep *EventProjectionImpl) worker(workerID int, eventCh <-chan *Event) {
	for {
		select {
		case <-ep.ctx.Done():
			return
		case event := <-eventCh:
			startTime := time.Now()

			// Process event through all applicable projections
			ep.mu.RLock()
			var applicableProjections []*Projection
			for _, projection := range ep.projections {
				if projection.IsActive && ep.canProjectionHandleEvent(projection, event) {
					applicableProjections = append(applicableProjections, projection)
				}
			}
			ep.mu.RUnlock()

			for _, projection := range applicableProjections {
				if err := ep.processEventForProjection(ep.ctx, event, projection); err != nil {
					ep.logger.Error("Failed to process event for projection",
						zap.Int("worker_id", workerID),
						zap.String("projection_id", projection.ID),
						zap.String("event_id", event.ID),
						zap.Error(err))
				}
			}

			// Update processing time
			duration := time.Since(startTime)
			ep.updateProcessingTime(duration)
		}
	}
}

func (ep *EventProjectionImpl) batchProcessor() {
	batch := make([]*Event, 0, ep.config.BatchSize)
	ticker := time.NewTicker(ep.config.BatchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ep.ctx.Done():
			return
		case events := <-ep.batchCh:
			// Process batch
			batch = append(batch, events...)
			if len(batch) >= ep.config.BatchSize {
				ep.processBatch(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			// Process batch on timeout
			if len(batch) > 0 {
				ep.processBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

func (ep *EventProjectionImpl) processBatch(events []*Event) {
	// Process batch of events
	for _, event := range events {
		select {
		case ep.eventCh <- event:
		default:
			ep.logger.Warn("Event channel full, dropping event")
		}
	}
}

func (ep *EventProjectionImpl) canProjectionHandleEvent(projection *Projection, event *Event) bool {
	for _, eventType := range projection.EventType {
		if eventType == event.Type {
			// Check aggregate filter
			if projection.AggregateID != "" && projection.AggregateID != event.AggregateID {
				return false
			}
			if projection.AggregateType != "" && projection.AggregateType != event.AggregateType {
				return false
			}
			return true
		}
	}
	return false
}

func (ep *EventProjectionImpl) processEventForProjection(ctx context.Context, event *Event, projection *Projection) error {
	startTime := time.Now()

	// Check if handler can handle event
	if !projection.Handler.CanHandle(event) {
		return nil // Skip this projection
	}

	// Process event through handler
	newData, err := projection.Handler.Project(ctx, event, projection)
	if err != nil {
		ep.updateProjectionError(projection.ID, err)
		return fmt.Errorf("handler failed: %w", err)
	}

	// Update projection data
	if newData != nil {
		projection.Data = newData
	}

	// Update state
	ep.updateProjectionState(projection.ID, event)

	// Update metrics
	ep.updateProjectionMetrics(projection.ID, time.Since(startTime))

	return nil
}

func (ep *EventProjectionImpl) updateProjectionState(projectionID string, event *Event) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	state, exists := ep.states[projectionID]
	if !exists {
		return
	}

	state.LastEventID = event.ID
	state.LastVersion = event.Version
	state.LastProcessed = event.Timestamp
	state.EventsProcessed++

	ep.stats.EventsProcessed++
	if ep.stats.LastProcessed == nil {
		ep.stats.LastProcessed = &time.Time{}
	}
	*ep.stats.LastProcessed = event.Timestamp
}

func (ep *EventProjectionImpl) updateProjectionMetrics(projectionID string, latency time.Duration) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	metrics, exists := ep.metrics[projectionID]
	if !exists {
		return
	}

	metrics.EventsProcessed++

	// Update average latency
	if metrics.EventsProcessed == 1 {
		metrics.AverageLatency = latency
	} else {
		totalLatency := time.Duration(float64(metrics.AverageLatency.Nanoseconds()) * float64(metrics.EventsProcessed-1))
		totalLatency += latency
		metrics.AverageLatency = totalLatency / time.Duration(metrics.EventsProcessed)
	}

	// Update events per second (simplified)
	metrics.EventsPerSecond = float64(metrics.EventsProcessed) / time.Since(time.Now()).Seconds()

	if ep.stats.LastProcessed == nil {
		ep.stats.LastProcessed = &time.Time{}
	}
	*ep.stats.LastProcessed = time.Now()
}

func (ep *EventProjectionImpl) updateProjectionError(projectionID string, err error) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	state, exists := ep.states[projectionID]
	if !exists {
		return
	}

	state.ErrorsCount++
	state.LastError = &time.Time{}
	*state.LastError = time.Now()
	state.ErrorMessage = err.Error()
}

func (ep *EventProjectionImpl) updateProcessingTime(duration time.Duration) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.stats.EventsProcessed == 0 {
		ep.stats.ProcessingTime = duration
	} else {
		totalTime := ep.stats.ProcessingTime.Nanoseconds() * ep.stats.EventsProcessed
		totalTime += duration.Nanoseconds()
		ep.stats.ProcessingTime = time.Duration(totalTime / (ep.stats.EventsProcessed + 1))
	}
}

// RegisterHandler registers a projection handler
func (ep *EventProjectionImpl) RegisterHandler(handlerType string, handler ProjectionHandler) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	ep.handlers[handlerType] = handler

	ep.logger.Info("Projection handler registered",
		zap.String("handler_type", handlerType))
}

// GetHandler retrieves a projection handler
func (ep *EventProjectionImpl) GetHandler(handlerType string) (ProjectionHandler, bool) {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	handler, exists := ep.handlers[handlerType]
	return handler, exists
}
