// Package entities provides domain entities
package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MemoryType represents the type of memory
type MemoryType string

const (
	MemoryTypeEpisodic MemoryType = "episodic"
	MemoryTypeSemantic MemoryType = "semantic"
	MemoryTypeWorking  MemoryType = "working"
)

// Memory represents a memory entity
type Memory struct {
	id          string
	memoryType  MemoryType
	content     string
	metadata    map[string]interface{}
	sessionID   string
	importance  float64
	accessCount int
	lastAccess  time.Time
	createdAt   time.Time
	updatedAt   time.Time
}

// NewMemory creates a new memory entity
func NewMemory(memoryType MemoryType, content string, sessionID string) (*Memory, error) {
	if memoryType == "" {
		return nil, fmt.Errorf("memory type cannot be empty")
	}
	if content == "" {
		return nil, fmt.Errorf("memory content cannot be empty")
	}

	now := time.Now()
	return &Memory{
		id:          uuid.New().String(),
		memoryType:  memoryType,
		content:     content,
		metadata:    make(map[string]interface{}),
		sessionID:   sessionID,
		importance:  0.5, // Default importance
		accessCount: 0,
		lastAccess:  now,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// ID returns the memory ID
func (m *Memory) ID() string {
	return m.id
}

// Type returns the memory type
func (m *Memory) Type() MemoryType {
	return m.memoryType
}

// Content returns the memory content
func (m *Memory) Content() string {
	return m.content
}

// SetContent updates the memory content
func (m *Memory) SetContent(content string) error {
	if content == "" {
		return fmt.Errorf("memory content cannot be empty")
	}
	m.content = content
	m.touch()
	return nil
}

// Metadata returns a copy of metadata
func (m *Memory) Metadata() map[string]interface{} {
	return copyMetadata(m.metadata)
}

// SetMetadata sets metadata
func (m *Memory) SetMetadata(metadata map[string]interface{}) {
	m.metadata = copyMetadata(metadata)
	m.touch()
}

// SessionID returns the session ID
func (m *Memory) SessionID() string {
	return m.sessionID
}

// Importance returns the importance score
func (m *Memory) Importance() float64 {
	return m.importance
}

// SetImportance sets the importance score
func (m *Memory) SetImportance(importance float64) error {
	if importance < 0 || importance > 1 {
		return fmt.Errorf("importance must be between 0 and 1")
	}
	m.importance = importance
	m.touch()
	return nil
}

// AccessCount returns the access count
func (m *Memory) AccessCount() int {
	return m.accessCount
}

// RecordAccess records an access to the memory
func (m *Memory) RecordAccess() {
	m.accessCount++
	m.lastAccess = time.Now()
	m.touch()
}

// LastAccess returns the last access time
func (m *Memory) LastAccess() time.Time {
	return m.lastAccess
}

// CreatedAt returns the creation timestamp
func (m *Memory) CreatedAt() time.Time {
	return m.createdAt
}

// UpdatedAt returns the last update timestamp
func (m *Memory) UpdatedAt() time.Time {
	return m.updatedAt
}

// touch updates the updatedAt timestamp
func (m *Memory) touch() {
	m.updatedAt = time.Now()
}

// EpisodicMemory represents episodic memory (short-term, session-based)
type EpisodicMemory struct {
	*Memory
	events []*MemoryEvent
}

// NewEpisodicMemory creates a new episodic memory
func NewEpisodicMemory(content string, sessionID string) (*EpisodicMemory, error) {
	mem, err := NewMemory(MemoryTypeEpisodic, content, sessionID)
	if err != nil {
		return nil, err
	}
	return &EpisodicMemory{
		Memory: mem,
		events: make([]*MemoryEvent, 0),
	}, nil
}

// AddEvent adds an event to episodic memory
func (em *EpisodicMemory) AddEvent(event *MemoryEvent) {
	em.events = append(em.events, event)
	em.touch()
}

// Events returns a copy of events
func (em *EpisodicMemory) Events() []*MemoryEvent {
	events := make([]*MemoryEvent, len(em.events))
	copy(events, em.events)
	return events
}

// SemanticMemory represents semantic memory (long-term, consolidated knowledge)
type SemanticMemory struct {
	*Memory
	concepts []string
	related  []string
}

// NewSemanticMemory creates a new semantic memory
func NewSemanticMemory(content string, sessionID string) (*SemanticMemory, error) {
	mem, err := NewMemory(MemoryTypeSemantic, content, sessionID)
	if err != nil {
		return nil, err
	}
	return &SemanticMemory{
		Memory:   mem,
		concepts: make([]string, 0),
		related:  make([]string, 0),
	}, nil
}

// AddConcept adds a concept to semantic memory
func (sm *SemanticMemory) AddConcept(concept string) {
	sm.concepts = append(sm.concepts, concept)
	sm.touch()
}

// Concepts returns concepts
func (sm *SemanticMemory) Concepts() []string {
	return sm.concepts
}

// AddRelated adds a related memory ID
func (sm *SemanticMemory) AddRelated(memoryID string) {
	sm.related = append(sm.related, memoryID)
	sm.touch()
}

// Related returns related memory IDs
func (sm *SemanticMemory) Related() []string {
	return sm.related
}

// WorkingMemory represents working memory (active context for tasks)
type WorkingMemory struct {
	*Memory
	taskID    string
	step      int
	maxSteps  int
	context   map[string]interface{}
	completed bool
}

// NewWorkingMemory creates a new working memory
func NewWorkingMemory(content string, sessionID string, taskID string, maxSteps int) (*WorkingMemory, error) {
	mem, err := NewMemory(MemoryTypeWorking, content, sessionID)
	if err != nil {
		return nil, err
	}
	return &WorkingMemory{
		Memory:    mem,
		taskID:    taskID,
		step:      0,
		maxSteps:  maxSteps,
		context:   make(map[string]interface{}),
		completed: false,
	}, nil
}

// TaskID returns the task ID
func (wm *WorkingMemory) TaskID() string {
	return wm.taskID
}

// Step returns the current step
func (wm *WorkingMemory) Step() int {
	return wm.step
}

// NextStep advances to the next step
func (wm *WorkingMemory) NextStep() error {
	if wm.completed {
		return fmt.Errorf("task already completed")
	}
	wm.step++
	if wm.step >= wm.maxSteps {
		wm.completed = true
	}
	wm.touch()
	return nil
}

// SetContext sets context for the current step
func (wm *WorkingMemory) SetContext(key string, value interface{}) {
	wm.context[key] = value
	wm.touch()
}

// GetContext gets context value
func (wm *WorkingMemory) GetContext(key string) (interface{}, bool) {
	value, exists := wm.context[key]
	return value, exists
}

// Context returns a copy of context
func (wm *WorkingMemory) Context() map[string]interface{} {
	return copyMetadata(wm.context)
}

// IsCompleted returns whether the task is completed
func (wm *WorkingMemory) IsCompleted() bool {
	return wm.completed
}

// Complete marks the task as completed
func (wm *WorkingMemory) Complete() {
	wm.completed = true
	wm.touch()
}

// MemoryEvent represents an event in episodic memory
type MemoryEvent struct {
	ID        string
	Type      string
	Content   string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// NewMemoryEvent creates a new memory event
func NewMemoryEvent(eventType string, content string) *MemoryEvent {
	return &MemoryEvent{
		ID:        uuid.New().String(),
		Type:      eventType,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}
