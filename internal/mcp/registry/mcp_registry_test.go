package registry

import (
	"testing"
	"time"
)

func TestNewMCPRegistry(t *testing.T) {
	tests := []struct {
		name   string
		config *RegistryConfig
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
		},
		{
			name: "custom config",
			config: &RegistryConfig{
				StoragePath:  "./custom-registry",
				AutoSave:     false,
				MaxProjects:  500,
				MaxTemplates: 50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewMCPRegistry(tt.config)
			if registry == nil {
				t.Fatal("NewMCPRegistry returned nil")
			}
			if registry.projects == nil {
				t.Error("projects map should not be nil")
			}
			if registry.templates == nil {
				t.Error("templates map should not be nil")
			}
			if registry.stacks == nil {
				t.Error("stacks map should not be nil")
			}
		})
	}
}

func TestMCPRegistry_RegisterProject(t *testing.T) {
	registry := NewMCPRegistry(nil)

	project := ProjectInfo{
		Name:      "test-project",
		Stack:     "go",
		Path:      "/tmp/test-project",
		Status:    "active",
		CreatedAt: time.Now(),
	}

	if err := registry.RegisterProject(project); err != nil {
		t.Fatalf("RegisterProject() error = %v", err)
	}

	// Try to register duplicate (should fail)
	if err := registry.RegisterProject(project); err == nil {
		t.Error("RegisterProject() should fail for duplicate project")
	}
}

func TestMCPRegistry_GetProjectByName(t *testing.T) {
	registry := NewMCPRegistry(nil)

	project := ProjectInfo{
		Name:      "test-get",
		Stack:     "go",
		Path:      "/tmp/test-get",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	_ = registry.RegisterProject(project)

	got, err := registry.GetProjectByName("test-get")
	if err != nil {
		t.Fatalf("GetProjectByName() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetProjectByName() returned nil")
	}
	if got.Name != "test-get" {
		t.Errorf("Expected name 'test-get', got '%s'", got.Name)
	}

	// Get nonexistent project
	_, err = registry.GetProjectByName("nonexistent")
	if err == nil {
		t.Error("GetProjectByName() should fail for nonexistent project")
	}
}

func TestMCPRegistry_GetProjectByPath(t *testing.T) {
	registry := NewMCPRegistry(nil)

	project := ProjectInfo{
		Name:      "test-path",
		Stack:     "go",
		Path:      "/tmp/test-path",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	_ = registry.RegisterProject(project)

	got, err := registry.GetProjectByPath("/tmp/test-path")
	if err != nil {
		t.Fatalf("GetProjectByPath() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetProjectByPath() returned nil")
	}
	if got.Path != "/tmp/test-path" {
		t.Errorf("Expected path '/tmp/test-path', got '%s'", got.Path)
	}
}

func TestMCPRegistry_ListProjects(t *testing.T) {
	registry := NewMCPRegistry(nil)

	// Register multiple projects
	_ = registry.RegisterProject(ProjectInfo{
		Name:   "project1",
		Stack:  "go",
		Path:   "/tmp/project1",
		Status: "active",
	})
	_ = registry.RegisterProject(ProjectInfo{
		Name:   "project2",
		Stack:  "web",
		Path:   "/tmp/project2",
		Status: "active",
	})

	// List all projects
	projects, err := registry.ListProjects(ProjectFilter{})
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) < 2 {
		t.Errorf("Expected at least 2 projects, got %d", len(projects))
	}

	// Filter by stack
	projects, err = registry.ListProjects(ProjectFilter{Stack: "go"})
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) == 0 {
		t.Error("Expected at least 1 project with stack 'go'")
	}
}

func TestMCPRegistry_ListTemplates(t *testing.T) {
	registry := NewMCPRegistry(nil)

	templates, err := registry.ListTemplates(TemplateFilter{})
	if err != nil {
		t.Fatalf("ListTemplates() error = %v", err)
	}
	if len(templates) == 0 {
		t.Error("ListTemplates() should return default templates")
	}
}

func TestMCPRegistry_GetStackInfo(t *testing.T) {
	registry := NewMCPRegistry(nil)

	info, err := registry.GetStackInfo("go")
	if err != nil {
		t.Fatalf("GetStackInfo() error = %v", err)
	}
	if info == nil {
		t.Fatal("GetStackInfo() returned nil")
	}
	if info.Name != "go" {
		t.Errorf("Expected stack name 'go', got '%s'", info.Name)
	}

	// Get nonexistent stack
	_, err = registry.GetStackInfo("nonexistent")
	if err == nil {
		t.Error("GetStackInfo() should fail for nonexistent stack")
	}
}

func TestMCPRegistry_UnregisterProject(t *testing.T) {
	registry := NewMCPRegistry(nil)

	project := ProjectInfo{
		Name:      "test-delete",
		Stack:     "go",
		Path:      "/tmp/test-delete",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	_ = registry.RegisterProject(project)

	if err := registry.UnregisterProject("test-delete"); err != nil {
		t.Fatalf("UnregisterProject() error = %v", err)
	}

	// Try to unregister again (should fail)
	if err := registry.UnregisterProject("test-delete"); err == nil {
		t.Error("UnregisterProject() should fail for nonexistent project")
	}
}

func TestMCPRegistry_UnregisterProjectByPath(t *testing.T) {
	registry := NewMCPRegistry(nil)

	project := ProjectInfo{
		Name:      "test-delete-path",
		Stack:     "go",
		Path:      "/tmp/test-delete-path",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	_ = registry.RegisterProject(project)

	if err := registry.UnregisterProjectByPath("/tmp/test-delete-path"); err != nil {
		t.Fatalf("UnregisterProjectByPath() error = %v", err)
	}

	// Verify project is deleted
	_, err := registry.GetProjectByName("test-delete-path")
	if err == nil {
		t.Error("Project should be deleted")
	}
}

func TestMCPRegistry_UpdateProject(t *testing.T) {
	registry := NewMCPRegistry(nil)

	project := ProjectInfo{
		Name:      "test-update",
		Stack:     "go",
		Path:      "/tmp/test-update",
		Status:    "active",
		Features:  []string{"feature1"},
		CreatedAt: time.Now(),
	}
	_ = registry.RegisterProject(project)

	updates := ProjectInfo{
		Features: []string{"feature1", "feature2"},
		Config:   map[string]interface{}{"key": "value"},
	}

	if err := registry.UpdateProject("test-update", updates); err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}

	// Verify update
	updated, err := registry.GetProjectByName("test-update")
	if err != nil {
		t.Fatalf("GetProjectByName() error = %v", err)
	}
	if len(updated.Features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(updated.Features))
	}
}

func TestMCPRegistry_GetRegistryStats(t *testing.T) {
	registry := NewMCPRegistry(nil)

	stats := registry.GetRegistryStats()
	if stats.TotalProjects < 0 {
		t.Error("TotalProjects should be >= 0")
	}
	if stats.TotalTemplates == 0 {
		t.Error("TotalTemplates should be > 0 (default templates)")
	}
	if stats.TotalStacks == 0 {
		t.Error("TotalStacks should be > 0 (default stacks)")
	}
}
