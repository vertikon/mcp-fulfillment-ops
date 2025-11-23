package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// MCPRegistry manages registration and discovery of MCPs, templates, and projects
type MCPRegistry struct {
	projects    map[string]*ProjectInfo
	templates   map[string]*TemplateInfo
	stacks      map[string]*StackInfo
	services    map[string]*ServiceInfo
	mu          sync.RWMutex
	logger      *zap.Logger
	config      *RegistryConfig
	lastUpdated time.Time
}

// RegistryConfig holds configuration for the registry
type RegistryConfig struct {
	StoragePath   string        `json:"storage_path"`
	AutoSave      bool          `json:"auto_save"`
	SaveInterval  time.Duration `json:"save_interval"`
	MaxProjects   int           `json:"max_projects"`
	MaxTemplates  int           `json:"max_templates"`
	EnableMetrics bool          `json:"enable_metrics"`
	CacheEnabled  bool          `json:"cache_enabled"`
	CacheTTL      time.Duration `json:"cache_ttl"`
}

// NewMCPRegistry creates a new MCP registry
func NewMCPRegistry(config *RegistryConfig) *MCPRegistry {
	if config == nil {
		config = &RegistryConfig{
			StoragePath:   "./registry",
			AutoSave:      true,
			SaveInterval:  5 * time.Minute,
			MaxProjects:   1000,
			MaxTemplates:  100,
			EnableMetrics: true,
			CacheEnabled:  true,
			CacheTTL:      1 * time.Hour,
		}
	}

	registry := &MCPRegistry{
		projects:    make(map[string]*ProjectInfo),
		templates:   make(map[string]*TemplateInfo),
		stacks:      make(map[string]*StackInfo),
		services:    make(map[string]*ServiceInfo),
		config:      config,
		logger:      logger.Get(),
		lastUpdated: time.Now(),
	}

	// Initialize registry
	if err := registry.initialize(); err != nil {
		registry.logger.Error("Failed to initialize registry", zap.Error(err))
	}

	// Start auto-save if enabled
	if config.AutoSave {
		go registry.startAutoSave()
	}

	return registry
}

// initialize initializes the registry with default data
func (r *MCPRegistry) initialize() error {
	r.logger.Info("Initializing MCP registry")

	// Initialize default stacks
	r.initializeDefaultStacks()

	// Initialize default templates
	r.initializeDefaultTemplates()

	// Load existing data from storage
	if err := r.loadFromStorage(); err != nil {
		r.logger.Warn("Failed to load registry data from storage", zap.Error(err))
	}

	return nil
}

// initializeDefaultStacks sets up default technology stacks
func (r *MCPRegistry) initializeDefaultStacks() {
	defaultStacks := []*StackInfo{
		{
			Name:        "go",
			DisplayName: "Go",
			Description: "Go microservice with Clean Architecture",
			Version:     "1.0.0",
			Category:    "backend",
			Features: []string{
				"clean-architecture",
				"docker",
				"testing",
				"monitoring",
				"graceful-shutdown",
			},
			Dependencies: []string{"go >= 1.21", "docker"},
			SupportedOS:  []string{"linux", "darwin", "windows"},
			Requirements: []string{},
			Templates:    []string{"go", "go-grpc", "go-web"},
			Status:       "active",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			Name:        "web",
			DisplayName: "Web",
			Description: "React/Vue.js web application with modern tooling",
			Version:     "1.0.0",
			Category:    "frontend",
			Features: []string{
				"typescript",
				"vite",
				"testing",
				"eslint",
				"prettier",
				"docker",
			},
			Dependencies: []string{"node >= 18", "npm"},
			SupportedOS:  []string{"linux", "darwin", "windows"},
			Requirements: []string{},
			Templates:    []string{"react", "vue", "angular"},
			Status:       "active",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			Name:        "tinygo",
			DisplayName: "TinyGo",
			Description: "TinyGo project for WebAssembly and embedded systems",
			Version:     "1.0.0",
			Category:    "embedded",
			Features: []string{
				"wasm",
				"embedded",
				"testing",
				"docker",
			},
			Dependencies: []string{"tinygo >= 0.30", "go >= 1.21"},
			SupportedOS:  []string{"linux", "darwin", "windows"},
			Requirements: []string{},
			Templates:    []string{"tinygo-wasm", "tinygo-embedded"},
			Status:       "active",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			Name:        "wasm",
			DisplayName: "WebAssembly",
			Description: "WebAssembly project using Rust/C++",
			Version:     "1.0.0",
			Category:    "wasm",
			Features: []string{
				"rust",
				"wasm-bindgen",
				"testing",
				"docker",
			},
			Dependencies: []string{"rust >= 1.70", "cargo"},
			SupportedOS:  []string{"linux", "darwin", "windows"},
			Requirements: []string{},
			Templates:    []string{"rust-wasm", "cpp-wasm"},
			Status:       "active",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			Name:        "mcp-go-premium",
			DisplayName: "MCP Go Premium",
			Description: "Premium Go MCP server with AI capabilities",
			Version:     "1.0.0",
			Category:    "mcp",
			Features: []string{
				"mcp-protocol",
				"ai-integration",
				"clean-architecture",
				"monitoring",
				"security",
				"docker",
				"kubernetes",
			},
			Dependencies: []string{"go >= 1.21", "docker", "kubectl"},
			SupportedOS:  []string{"linux", "darwin", "windows"},
			Requirements: []string{},
			Templates:    []string{"mcp-go-premium", "mcp-go-enterprise"},
			Status:       "active",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	for _, stack := range defaultStacks {
		r.stacks[stack.Name] = stack
	}

	r.logger.Info("Initialized default stacks", zap.Int("count", len(defaultStacks)))
}

// initializeDefaultTemplates sets up default templates
func (r *MCPRegistry) initializeDefaultTemplates() {
	defaultTemplates := []*TemplateInfo{
		{
			ID:          "go-basic",
			Name:        "Go Basic Microservice",
			Description: "Basic Go microservice with Clean Architecture",
			Stack:       "go",
			Version:     "1.0.0",
			Category:    "backend",
			Features:    []string{"clean-architecture", "docker", "testing"},
			Author:      "mcp-fulfillment-ops Team",
			License:     "MIT",
			Repository:  "https://github.com/vertikon/mcp-fulfillment-ops",
			TemplateDir: "go",
			Status:      "active",
			Downloads:   0,
			Rating:      5.0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "web-react",
			Name:        "React Web Application",
			Description: "React web application with TypeScript and Vite",
			Stack:       "web",
			Version:     "1.0.0",
			Category:    "frontend",
			Features:    []string{"typescript", "vite", "testing"},
			Author:      "mcp-fulfillment-ops Team",
			License:     "MIT",
			Repository:  "https://github.com/vertikon/mcp-fulfillment-ops",
			TemplateDir: "web",
			Status:      "active",
			Downloads:   0,
			Rating:      5.0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "tinygo-wasm",
			Name:        "TinyGo WebAssembly",
			Description: "TinyGo project targeting WebAssembly",
			Stack:       "tinygo",
			Version:     "1.0.0",
			Category:    "wasm",
			Features:    []string{"wasm", "embedded", "testing"},
			Author:      "mcp-fulfillment-ops Team",
			License:     "MIT",
			Repository:  "https://github.com/vertikon/mcp-fulfillment-ops",
			TemplateDir: "tinygo",
			Status:      "active",
			Downloads:   0,
			Rating:      5.0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, template := range defaultTemplates {
		r.templates[template.ID] = template
	}

	r.logger.Info("Initialized default templates", zap.Int("count", len(defaultTemplates)))
}

// ProjectInfo represents information about a generated project
type ProjectInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Stack        string                 `json:"stack"`
	Path         string                 `json:"path"`
	Features     []string               `json:"features"`
	Config       map[string]interface{} `json:"config"`
	Status       string                 `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	LastBuilt    *time.Time             `json:"last_built,omitempty"`
	Size         int64                  `json:"size"`
	Files        int                    `json:"files"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Author       string                 `json:"author,omitempty"`
	Description  string                 `json:"description,omitempty"`
}

// TemplateInfo represents information about a template
type TemplateInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Stack       string                 `json:"stack"`
	Version     string                 `json:"version"`
	Category    string                 `json:"category"`
	Features    []string               `json:"features"`
	Author      string                 `json:"author"`
	License     string                 `json:"license"`
	Repository  string                 `json:"repository"`
	TemplateDir string                 `json:"template_dir"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Status      string                 `json:"status"`
	Downloads   int64                  `json:"downloads"`
	Rating      float64                `json:"rating"`
	Reviews     int                    `json:"reviews"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// StackInfo represents information about a technology stack
type StackInfo struct {
	Name         string    `json:"name"`
	DisplayName  string    `json:"display_name"`
	Description  string    `json:"description"`
	Version      string    `json:"version"`
	Category     string    `json:"category"`
	Features     []string  `json:"features"`
	Dependencies []string  `json:"dependencies"`
	SupportedOS  []string  `json:"supported_os"`
	Requirements []string  `json:"requirements"`
	Templates    []string  `json:"templates"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ServiceInfo represents information about a registered service
type ServiceInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Version      string                 `json:"version"`
	Endpoint     string                 `json:"endpoint"`
	Status       string                 `json:"status"`
	Health       string                 `json:"health"`
	Config       map[string]interface{} `json:"config"`
	Metadata     map[string]string      `json:"metadata"`
	RegisteredAt time.Time              `json:"registered_at"`
	LastSeen     *time.Time             `json:"last_seen,omitempty"`
}

// RegisterProject registers a new project
func (r *MCPRegistry) RegisterProject(project ProjectInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if project already exists
	if _, exists := r.projects[project.Name]; exists {
		return fmt.Errorf("project %s already exists", project.Name)
	}

	// Set default values
	if project.ID == "" {
		project.ID = generateID("project", project.Name)
	}

	if project.Status == "" {
		project.Status = "active"
	}

	if project.CreatedAt.IsZero() {
		project.CreatedAt = time.Now()
	}

	project.UpdatedAt = time.Now()

	// Add project
	r.projects[project.Name] = &project
	r.lastUpdated = time.Now()

	r.logger.Info("Project registered",
		zap.String("name", project.Name),
		zap.String("stack", project.Stack),
		zap.String("path", project.Path))

	// Save to storage if auto-save enabled
	if r.config.AutoSave {
		go r.saveToStorage()
	}

	return nil
}

// GetProjectByName retrieves a project by name
func (r *MCPRegistry) GetProjectByName(name string) (*ProjectInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, exists := r.projects[name]
	if !exists {
		return nil, fmt.Errorf("project not found: %s", name)
	}

	return project, nil
}

// GetProjectByPath retrieves a project by path
func (r *MCPRegistry) GetProjectByPath(path string) (*ProjectInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, project := range r.projects {
		if project.Path == path {
			return project, nil
		}
	}

	return nil, fmt.Errorf("project not found at path: %s", path)
}

// ListProjects returns a list of all projects
func (r *MCPRegistry) ListProjects(filter ProjectFilter) ([]*ProjectInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var projects []*ProjectInfo

	for _, project := range r.projects {
		if r.matchesProjectFilter(project, filter) {
			projects = append(projects, project)
		}
	}

	return projects, nil
}

// ProjectFilter represents filters for listing projects
type ProjectFilter struct {
	Stack  string `json:"stack,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// matchesProjectFilter checks if a project matches the filter
func (r *MCPRegistry) matchesProjectFilter(project *ProjectInfo, filter ProjectFilter) bool {
	if filter.Stack != "" && project.Stack != filter.Stack {
		return false
	}

	if filter.Status != "" && project.Status != filter.Status {
		return false
	}

	return true
}

// ListTemplates returns a list of all templates
func (r *MCPRegistry) ListTemplates(filter TemplateFilter) ([]*TemplateInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var templates []*TemplateInfo

	for _, template := range r.templates {
		if r.matchesTemplateFilter(template, filter) {
			templates = append(templates, template)
		}
	}

	return templates, nil
}

// TemplateFilter represents filters for listing templates
type TemplateFilter struct {
	Stack    string `json:"stack,omitempty"`
	Category string `json:"category,omitempty"`
	Status   string `json:"status,omitempty"`
}

// matchesTemplateFilter checks if a template matches the filter
func (r *MCPRegistry) matchesTemplateFilter(template *TemplateInfo, filter TemplateFilter) bool {
	if filter.Stack != "" && template.Stack != filter.Stack {
		return false
	}

	if filter.Category != "" && template.Category != filter.Category {
		return false
	}

	if filter.Status != "" && template.Status != filter.Status {
		return false
	}

	return true
}

// GetStackInfo returns information about a specific stack
func (r *MCPRegistry) GetStackInfo(stack string) (*StackInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stackInfo, exists := r.stacks[stack]
	if !exists {
		return nil, fmt.Errorf("stack not found: %s", stack)
	}

	return stackInfo, nil
}

// ListStacks returns a list of all available stacks
func (r *MCPRegistry) ListStacks() ([]*StackInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stacks := make([]*StackInfo, 0, len(r.stacks))
	for _, stack := range r.stacks {
		stacks = append(stacks, stack)
	}

	return stacks, nil
}

// RegisterService registers a new service
func (r *MCPRegistry) RegisterService(service ServiceInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if service.ID == "" {
		service.ID = generateID("service", service.Name)
	}

	if service.RegisteredAt.IsZero() {
		service.RegisteredAt = time.Now()
	}

	r.services[service.ID] = &service
	r.lastUpdated = time.Now()

	r.logger.Info("Service registered",
		zap.String("id", service.ID),
		zap.String("name", service.Name),
		zap.String("type", service.Type))

	return nil
}

// GetService retrieves a service by ID
func (r *MCPRegistry) GetService(id string) (*ServiceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	service, exists := r.services[id]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", id)
	}

	return service, nil
}

// ListServices returns a list of all services
func (r *MCPRegistry) ListServices() ([]*ServiceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]*ServiceInfo, 0, len(r.services))
	for _, service := range r.services {
		services = append(services, service)
	}

	return services, nil
}

// GetRegistryStats returns statistics about the registry
type RegistryStats struct {
	TotalProjects  int       `json:"total_projects"`
	TotalTemplates int       `json:"total_templates"`
	TotalStacks    int       `json:"total_stacks"`
	TotalServices  int       `json:"total_services"`
	LastUpdated    time.Time `json:"last_updated"`
}

func (r *MCPRegistry) GetRegistryStats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return RegistryStats{
		TotalProjects:  len(r.projects),
		TotalTemplates: len(r.templates),
		TotalStacks:    len(r.stacks),
		TotalServices:  len(r.services),
		LastUpdated:    r.lastUpdated,
	}
}

// Helper functions
func generateID(prefix, name string) string {
	return fmt.Sprintf("%s-%s-%d", prefix, name, time.Now().Unix())
}

// startAutoSave starts the auto-save routine
func (r *MCPRegistry) startAutoSave() {
	ticker := time.NewTicker(r.config.SaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.saveToStorage(); err != nil {
				r.logger.Error("Auto-save failed", zap.Error(err))
			}
		}
	}
}

// saveToStorage saves registry data to persistent storage
func (r *MCPRegistry) saveToStorage() error {
	r.logger.Info("Saving registry data to storage",
		zap.String("path", r.config.StoragePath))

	// Ensure storage directory exists
	if err := os.MkdirAll(r.config.StoragePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Save projects
	projectsFile := filepath.Join(r.config.StoragePath, "projects.json")
	if err := r.saveProjects(projectsFile); err != nil {
		r.logger.Warn("Failed to save projects", zap.Error(err))
	}

	// Save templates
	templatesFile := filepath.Join(r.config.StoragePath, "templates.json")
	if err := r.saveTemplates(templatesFile); err != nil {
		r.logger.Warn("Failed to save templates", zap.Error(err))
	}

	// Save stacks
	stacksFile := filepath.Join(r.config.StoragePath, "stacks.json")
	if err := r.saveStacks(stacksFile); err != nil {
		r.logger.Warn("Failed to save stacks", zap.Error(err))
	}

	// Save services
	servicesFile := filepath.Join(r.config.StoragePath, "services.json")
	if err := r.saveServices(servicesFile); err != nil {
		r.logger.Warn("Failed to save services", zap.Error(err))
	}

	r.logger.Info("Registry data saved successfully")
	return nil
}

// loadFromStorage loads registry data from persistent storage
func (r *MCPRegistry) loadFromStorage() error {
	r.logger.Info("Loading registry data from storage",
		zap.String("path", r.config.StoragePath))

	// Check if storage directory exists
	if _, err := os.Stat(r.config.StoragePath); os.IsNotExist(err) {
		r.logger.Info("Storage directory does not exist, skipping load")
		return nil
	}

	// Load projects
	projectsFile := filepath.Join(r.config.StoragePath, "projects.json")
	if err := r.loadProjects(projectsFile); err != nil {
		r.logger.Warn("Failed to load projects", zap.Error(err))
	}

	// Load templates
	templatesFile := filepath.Join(r.config.StoragePath, "templates.json")
	if err := r.loadTemplates(templatesFile); err != nil {
		r.logger.Warn("Failed to load templates", zap.Error(err))
	}

	// Load stacks
	stacksFile := filepath.Join(r.config.StoragePath, "stacks.json")
	if err := r.loadStacks(stacksFile); err != nil {
		r.logger.Warn("Failed to load stacks", zap.Error(err))
	}

	// Load services
	servicesFile := filepath.Join(r.config.StoragePath, "services.json")
	if err := r.loadServices(servicesFile); err != nil {
		r.logger.Warn("Failed to load services", zap.Error(err))
	}

	r.logger.Info("Registry data loaded successfully")
	return nil
}

// Helper methods for saving/loading individual components
func (r *MCPRegistry) saveProjects(filePath string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := json.MarshalIndent(r.projects, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func (r *MCPRegistry) loadProjects(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var projects map[string]*ProjectInfo
	if err := json.Unmarshal(data, &projects); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for name, project := range projects {
		r.projects[name] = project
	}

	return nil
}

func (r *MCPRegistry) saveTemplates(filePath string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := json.MarshalIndent(r.templates, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func (r *MCPRegistry) loadTemplates(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var templates map[string]*TemplateInfo
	if err := json.Unmarshal(data, &templates); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for id, template := range templates {
		r.templates[id] = template
	}

	return nil
}

func (r *MCPRegistry) saveStacks(filePath string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := json.MarshalIndent(r.stacks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func (r *MCPRegistry) loadStacks(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var stacks map[string]*StackInfo
	if err := json.Unmarshal(data, &stacks); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for name, stack := range stacks {
		r.stacks[name] = stack
	}

	return nil
}

func (r *MCPRegistry) saveServices(filePath string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := json.MarshalIndent(r.services, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func (r *MCPRegistry) loadServices(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var services map[string]*ServiceInfo
	if err := json.Unmarshal(data, &services); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for id, service := range services {
		r.services[id] = service
	}

	return nil
}

// UnregisterProject removes a project from the registry
func (r *MCPRegistry) UnregisterProject(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.projects[name]; !exists {
		return fmt.Errorf("project not found: %s", name)
	}

	delete(r.projects, name)
	r.lastUpdated = time.Now()

	r.logger.Info("Project unregistered", zap.String("name", name))

	// Save to storage if auto-save enabled
	if r.config.AutoSave {
		go r.saveToStorage()
	}

	return nil
}

// UnregisterProjectByPath removes a project from the registry by path
func (r *MCPRegistry) UnregisterProjectByPath(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var projectName string
	for name, project := range r.projects {
		if project.Path == path {
			projectName = name
			break
		}
	}

	if projectName == "" {
		return fmt.Errorf("project not found at path: %s", path)
	}

	delete(r.projects, projectName)
	r.lastUpdated = time.Now()

	r.logger.Info("Project unregistered by path", zap.String("path", path))

	// Save to storage if auto-save enabled
	if r.config.AutoSave {
		go r.saveToStorage()
	}

	return nil
}

// UpdateProject updates an existing project in the registry
func (r *MCPRegistry) UpdateProject(name string, updates ProjectInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	project, exists := r.projects[name]
	if !exists {
		return fmt.Errorf("project not found: %s", name)
	}

	// Update fields
	if updates.Features != nil {
		project.Features = updates.Features
	}
	if updates.Config != nil {
		if project.Config == nil {
			project.Config = make(map[string]interface{})
		}
		for k, v := range updates.Config {
			project.Config[k] = v
		}
	}
	if updates.Status != "" {
		project.Status = updates.Status
	}
	if updates.Description != "" {
		project.Description = updates.Description
	}
	if updates.Tags != nil {
		project.Tags = updates.Tags
	}

	project.UpdatedAt = time.Now()
	r.lastUpdated = time.Now()

	r.logger.Info("Project updated",
		zap.String("name", name),
		zap.Strings("features", project.Features))

	// Save to storage if auto-save enabled
	if r.config.AutoSave {
		go r.saveToStorage()
	}

	return nil
}

// Shutdown gracefully shuts down the registry
func (r *MCPRegistry) Shutdown() error {
	r.logger.Info("Shutting down MCP registry")

	// Final save
	if err := r.saveToStorage(); err != nil {
		r.logger.Error("Final save failed", zap.Error(err))
	}

	return nil
}
