package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// GoGenerator generates Go projects with Clean Architecture
type GoGenerator struct {
	*BaseGenerator
}

// NewGoGenerator creates a new Go generator
func NewGoGenerator(templateDir string) *GoGenerator {
	base := NewBaseGenerator("Go Generator", "go", templateDir)
	return &GoGenerator{
		BaseGenerator: base,
	}
}

// Generate implements the Generator interface
func (g *GoGenerator) Generate(ctx interface{}, req GenerateRequest) (*GenerateResult, error) {
	// Call base generator with Go-specific modifications
	result, err := g.BaseGenerator.Generate(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add Go-specific post-processing
	if err := g.postProcessGoProject(result.Path, req); err != nil {
		return nil, fmt.Errorf("Go post-processing failed: %w", err)
	}

	return result, nil
}

// getTemplateFiles returns Go-specific template files
func (g *GoGenerator) getTemplateFiles(req GenerateRequest) ([]TemplateFile, error) {
	files := []TemplateFile{
		// Basic project files
		{
			SourcePath: filepath.Join(g.templateDir, "go.mod.tmpl"),
			TargetPath: "go.mod",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "go.sum.tmpl"),
			TargetPath: "go.sum",
			IsTemplate: false,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "README.md.tmpl"),
			TargetPath: "README.md",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, ".gitignore.tmpl"),
			TargetPath: ".gitignore",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "Makefile.tmpl"),
			TargetPath: "Makefile",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "Dockerfile.tmpl"),
			TargetPath: "Dockerfile",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "docker-compose.yml.tmpl"),
			TargetPath: "docker-compose.yml",
			IsTemplate: true,
		},

		// Command files
		{
			SourcePath: filepath.Join(g.templateDir, "cmd", "server", "main.go.tmpl"),
			TargetPath: "cmd/server/main.go",
			IsTemplate: true,
		},

		// Internal files - Domain
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "domain", "entities.go.tmpl"),
			TargetPath: "internal/domain/entities.go",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "domain", "repositories.go.tmpl"),
			TargetPath: "internal/domain/repositories.go",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "domain", "services.go.tmpl"),
			TargetPath: "internal/domain/services.go",
			IsTemplate: true,
		},

		// Internal files - Application
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "application", "use_cases.go.tmpl"),
			TargetPath: "internal/application/use_cases.go",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "application", "handlers.go.tmpl"),
			TargetPath: "internal/application/handlers.go",
			IsTemplate: true,
		},

		// Internal files - Infrastructure
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "infrastructure", "database.go.tmpl"),
			TargetPath: "internal/infrastructure/database.go",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "infrastructure", "repositories_impl.go.tmpl"),
			TargetPath: "internal/infrastructure/repositories_impl.go",
			IsTemplate: true,
		},

		// Config files
		{
			SourcePath: filepath.Join(g.templateDir, "configs", "config.yaml.tmpl"),
			TargetPath: "configs/config.yaml",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "configs", "config.dev.yaml.tmpl"),
			TargetPath: "configs/config.dev.yaml",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "configs", "config.prod.yaml.tmpl"),
			TargetPath: "configs/config.prod.yaml",
			IsTemplate: true,
		},

		// Test files
		{
			SourcePath: filepath.Join(g.templateDir, "tests", "main_test.go.tmpl"),
			TargetPath: "tests/main_test.go",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "tests", "integration_test.go.tmpl"),
			TargetPath: "tests/integration_test.go",
			IsTemplate: true,
		},
	}

	// Add feature-specific files
	if g.hasFeature(req.Features, "monitoring") {
		files = append(files, []TemplateFile{
			{
				SourcePath: filepath.Join(g.templateDir, "internal", "monitoring", "metrics.go.tmpl"),
				TargetPath: "internal/monitoring/metrics.go",
				IsTemplate: true,
			},
			{
				SourcePath: filepath.Join(g.templateDir, "internal", "monitoring", "health.go.tmpl"),
				TargetPath: "internal/monitoring/health.go",
				IsTemplate: true,
			},
		}...)
	}

	if g.hasFeature(req.Features, "security") {
		files = append(files, []TemplateFile{
			{
				SourcePath: filepath.Join(g.templateDir, "internal", "security", "auth.go.tmpl"),
				TargetPath: "internal/security/auth.go",
				IsTemplate: true,
			},
			{
				SourcePath: filepath.Join(g.templateDir, "internal", "security", "middleware.go.tmpl"),
				TargetPath: "internal/security/middleware.go",
				IsTemplate: true,
			},
		}...)
	}

	if g.hasFeature(req.Features, "grpc") {
		files = append(files, []TemplateFile{
			{
				SourcePath: filepath.Join(g.templateDir, "api", "proto", "service.proto.tmpl"),
				TargetPath: "api/proto/service.proto",
				IsTemplate: true,
			},
			{
				SourcePath: filepath.Join(g.templateDir, "internal", "grpc", "server.go.tmpl"),
				TargetPath: "internal/grpc/server.go",
				IsTemplate: true,
			},
		}...)
	}

	if g.hasFeature(req.Features, "migrations") {
		files = append(files, []TemplateFile{
			{
				SourcePath: filepath.Join(g.templateDir, "migrations", "001_initial.up.sql.tmpl"),
				TargetPath: "migrations/001_initial.up.sql",
				IsTemplate: true,
			},
			{
				SourcePath: filepath.Join(g.templateDir, "migrations", "001_initial.down.sql.tmpl"),
				TargetPath: "migrations/001_initial.down.sql",
				IsTemplate: true,
			},
		}...)
	}

	return files, nil
}

// hasFeature checks if a feature is enabled
func (g *GoGenerator) hasFeature(features []string, feature string) bool {
	for _, f := range features {
		if strings.EqualFold(f, feature) {
			return true
		}
	}
	return false
}

// postProcessGoProject performs Go-specific post-processing
func (g *GoGenerator) postProcessGoProject(projectPath string, req GenerateRequest) error {
	g.logger.Info("Starting Go project post-processing",
		zap.String("project", req.Name),
		zap.String("path", projectPath),
		zap.Strings("features", req.Features))

	// Verify go.mod exists
	goModPath := filepath.Join(projectPath, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		g.logger.Warn("go.mod not found, skipping post-processing",
			zap.String("path", goModPath))
		return nil
	}

	// Create .git directory if it doesn't exist (for git init)
	gitDir := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Note: Actual git init would require executing shell commands
		// For now, we just ensure the directory structure is ready
		g.logger.Debug("Git directory not found, project ready for git init")
	}

	// Verify required directories exist
	requiredDirs := []string{"cmd", "internal", "pkg", "configs", "tests"}
	for _, dir := range requiredDirs {
		dirPath := filepath.Join(projectPath, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			g.logger.Warn("Required directory missing",
				zap.String("directory", dir))
		}
	}

	// Log completion
	g.logger.Info("Go project post-processing completed",
		zap.String("project", req.Name),
		zap.Strings("features", req.Features))

	return nil
}

// validateGoRequest validates Go-specific generation request
func (g *GoGenerator) validateGoRequest(req GenerateRequest) error {
	// Check Go version compatibility
	// Validate features are supported
	// Check naming conventions

	if req.Name == "" {
		return fmt.Errorf("project name is required")
	}

	// Check if project name is a valid Go module name
	if !isValidGoModuleName(req.Name) {
		return fmt.Errorf("invalid Go module name: %s", req.Name)
	}

	return nil
}

// isValidGoModuleName validates Go module name
func isValidGoModuleName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}

	// Check for valid characters and patterns
	// This is a simplified validation
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-./"

	for _, char := range name {
		if !strings.ContainsRune(validChars, char) {
			return false
		}
	}

	// Cannot start with dash or dot
	if name[0] == '-' || name[0] == '.' {
		return false
	}

	return true
}

// GetGoVersion returns the Go version for the project
func (g *GoGenerator) getGoVersion(req GenerateRequest) string {
	if goVersion, ok := req.Config["go_version"].(string); ok {
		return goVersion
	}
	return "1.21" // Default Go version
}

// GetDependencies returns the list of Go dependencies
func (g *GoGenerator) getDependencies(req GenerateRequest) []string {
	var deps []string

	// Core dependencies
	deps = append(deps, []string{
		"github.com/gin-gonic/gin",
		"github.com/spf13/viper",
		"go.uber.org/zap",
	}...)

	// Feature-specific dependencies
	if g.hasFeature(req.Features, "monitoring") {
		deps = append(deps, []string{
			"github.com/prometheus/client_golang",
			"github.com/prometheus/client_golang/prometheus",
		}...)
	}

	if g.hasFeature(req.Features, "security") {
		deps = append(deps, []string{
			"github.com/golang-jwt/jwt",
			"golang.org/x/crypto",
		}...)
	}

	if g.hasFeature(req.Features, "grpc") {
		deps = append(deps, []string{
			"google.golang.org/grpc",
			"google.golang.org/protobuf",
		}...)
	}

	if g.hasFeature(req.Features, "database") {
		deps = append(deps, []string{
			"gorm.io/gorm",
			"gorm.io/driver/postgres",
			"database/sql",
		}...)
	}

	return deps
}

// GetDevDependencies returns the list of development dependencies
func (g *GoGenerator) getDevDependencies(req GenerateRequest) []string {
	var deps []string

	// Core dev dependencies
	deps = append(deps, []string{
		"github.com/stretchr/testify/assert",
		"github.com/stretchr/testify/mock",
		"github.com/golang/mock/gomock",
	}...)

	return deps
}

// CreateDockerfile generates Dockerfile content
func (g *GoGenerator) CreateDockerfile(req GenerateRequest) string {
	goVersion := g.getGoVersion(req)

	return fmt.Sprintf(`# Build stage
FROM golang:%s-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/server/main.go

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
`, goVersion)
}

// CreateMakefile generates Makefile content
func (g *GoGenerator) CreateMakefile(req GenerateRequest) string {
	return `.PHONY: build test clean docker-build docker-run

build:
	go build -o bin/server cmd/server/main.go

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

docker-build:
	docker build -t $(PROJECT_NAME) .

docker-run:
	docker run -p 8080:8080 $(PROJECT_NAME)

lint:
	golangci-lint run

fmt:
	go fmt ./...

mod-tidy:
	go mod tidy

mod-download:
	go mod download

run:
	go run cmd/server/main.go
`
}

// Validate validates the Go generator
func (g *GoGenerator) Validate() error {
	if err := g.BaseGenerator.Validate(); err != nil {
		return err
	}

	// Go-specific validation
	if g.stack != "go" {
		return fmt.Errorf("generator stack mismatch: expected 'go', got '%s'", g.stack)
	}

	return nil
}
