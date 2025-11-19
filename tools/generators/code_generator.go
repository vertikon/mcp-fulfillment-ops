package generators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// CodeGenerator generates code from blueprints/DTOS
type CodeGenerator struct {
	logger *zap.Logger
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator() *CodeGenerator {
	return &CodeGenerator{
		logger: logger.Get(),
	}
}

// GenerateCode generates code files from a blueprint or DTO definition
func (g *CodeGenerator) GenerateCode(ctx context.Context, req CodeGenerateRequest) (*CodeGenerateResult, error) {
	g.logger.Info("Generating code",
		zap.String("type", req.Type),
		zap.String("name", req.Name),
		zap.String("output", req.OutputPath))

	// Validate request
	if err := g.Validate(req); err != nil {
		return nil, err
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(req.OutputPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate code based on type
	var content []byte
	var err error

	switch req.Type {
	case "handler":
		content, err = g.generateHandler(req)
	case "entity":
		content, err = g.generateEntity(req)
	case "repository":
		content, err = g.generateRepository(req)
	case "service":
		content, err = g.generateService(req)
	case "usecase":
		content, err = g.generateUseCase(req)
	case "dto":
		content, err = g.generateDTO(req)
	default:
		return nil, fmt.Errorf("unsupported code type: %s", req.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("code generation failed: %w", err)
	}

	// Write file
	if err := os.WriteFile(req.OutputPath, content, 0644); err != nil {
		return nil, fmt.Errorf("failed to write code file: %w", err)
	}

	return &CodeGenerateResult{
		Path:    req.OutputPath,
		Size:    int64(len(content)),
		Type:    req.Type,
		Content: string(content),
	}, nil
}

// CodeGenerateRequest represents a request to generate code
type CodeGenerateRequest struct {
	Type       string                 `json:"type"` // handler, entity, repository, service, usecase, dto
	Name       string                 `json:"name"`
	Package    string                 `json:"package"`
	OutputPath string                 `json:"output_path"`
	Fields     []FieldDefinition     `json:"fields,omitempty"`
	Methods    []MethodDefinition    `json:"methods,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
}

// CodeGenerateResult represents the result of code generation
type CodeGenerateResult struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

// FieldDefinition represents a field in a struct
type FieldDefinition struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Tag      string `json:"tag,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

// MethodDefinition represents a method signature
type MethodDefinition struct {
	Name       string   `json:"name"`
	Params     []string `json:"params,omitempty"`
	Returns    []string `json:"returns,omitempty"`
	Comment    string   `json:"comment,omitempty"`
}

// Validate validates the code generation request
func (g *CodeGenerator) Validate(req CodeGenerateRequest) error {
	if req.Type == "" {
		return fmt.Errorf("code type is required")
	}
	
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	
	if req.OutputPath == "" {
		return fmt.Errorf("output path is required")
	}
	
	validTypes := []string{"handler", "entity", "repository", "service", "usecase", "dto"}
	valid := false
	for _, vt := range validTypes {
		if req.Type == vt {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid code type: %s (valid types: %v)", req.Type, validTypes)
	}
	
	return nil
}

// generateHandler generates a handler file
func (g *CodeGenerator) generateHandler(req CodeGenerateRequest) ([]byte, error) {
	tmpl := `package {{.Package}}

import (
	"context"
	"net/http"
)

// {{.Name}}Handler handles HTTP requests for {{.Name}}
type {{.Name}}Handler struct {
	// Add dependencies here
}

// New{{.Name}}Handler creates a new {{.Name}} handler
func New{{.Name}}Handler() *{{.Name}}Handler {
	return &{{.Name}}Handler{}
}

// Handle processes the HTTP request
func (h *{{.Name}}Handler) Handle(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Implement handler logic
}
`
	return g.executeTemplate(tmpl, req)
}

// generateEntity generates an entity file
func (g *CodeGenerator) generateEntity(req CodeGenerateRequest) ([]byte, error) {
	tmpl := `package {{.Package}}

import (
	"time"
)

// {{.Name}} represents a {{.Name}} entity
type {{.Name}} struct {
{{range .Fields}}	{{.Name}} {{.Type}} ` + "`{{.Tag}}`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{end}}
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\"`" + `
}
`
	return g.executeTemplate(tmpl, req)
}

// generateRepository generates a repository file
func (g *CodeGenerator) generateRepository(req CodeGenerateRequest) ([]byte, error) {
	tmpl := `package {{.Package}}

import (
	"context"
)

// {{.Name}}Repository defines the interface for {{.Name}} persistence
type {{.Name}}Repository interface {
{{range .Methods}}	{{.Name}}({{range $i, $p := .Params}}{{if $i}}, {{end}}{{$p}}{{end}}) {{if .Returns}}({{range $i, $r := .Returns}}{{if $i}}, {{end}}{{$r}}{{end}}){{end}}
{{end}}
}

// {{.Name | lower}}Repository implements {{.Name}}Repository
type {{.Name | lower}}Repository struct {
	// Add dependencies here
}

// New{{.Name}}Repository creates a new {{.Name}} repository
func New{{.Name}}Repository() {{.Name}}Repository {
	return &{{.Name | lower}}Repository{}
}
`
	return g.executeTemplate(tmpl, req)
}

// generateService generates a service file
func (g *CodeGenerator) generateService(req CodeGenerateRequest) ([]byte, error) {
	tmpl := `package {{.Package}}

import (
	"context"
)

// {{.Name}}Service defines the interface for {{.Name}} business logic
type {{.Name}}Service interface {
{{range .Methods}}	{{.Name}}({{range $i, $p := .Params}}{{if $i}}, {{end}}{{$p}}{{end}}) {{if .Returns}}({{range $i, $r := .Returns}}{{if $i}}, {{end}}{{$r}}{{end}}){{end}}
{{end}}
}

// {{.Name | lower}}Service implements {{.Name}}Service
type {{.Name | lower}}Service struct {
	// Add dependencies here
}

// New{{.Name}}Service creates a new {{.Name}} service
func New{{.Name}}Service() {{.Name}}Service {
	return &{{.Name | lower}}Service{}
}
`
	return g.executeTemplate(tmpl, req)
}

// generateUseCase generates a use case file
func (g *CodeGenerator) generateUseCase(req CodeGenerateRequest) ([]byte, error) {
	tmpl := `package {{.Package}}

import (
	"context"
)

// {{.Name}}UseCase implements the {{.Name}} use case
type {{.Name}}UseCase struct {
	// Add dependencies here
}

// New{{.Name}}UseCase creates a new {{.Name}} use case
func New{{.Name}}UseCase() *{{.Name}}UseCase {
	return &{{.Name}}UseCase{}
}

// Execute executes the use case
func (uc *{{.Name}}UseCase) Execute(ctx context.Context) error {
	// Implement use case logic
	return nil
}
`
	return g.executeTemplate(tmpl, req)
}

// generateDTO generates a DTO file
func (g *CodeGenerator) generateDTO(req CodeGenerateRequest) ([]byte, error) {
	tmpl := `package {{.Package}}

// {{.Name}}DTO represents the data transfer object for {{.Name}}
type {{.Name}}DTO struct {
{{range .Fields}}	{{.Name}} {{.Type}} ` + "`{{.Tag}}`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{end}}
}
`
	return g.executeTemplate(tmpl, req)
}

// executeTemplate executes a template with the given data
func (g *CodeGenerator) executeTemplate(tmplStr string, req CodeGenerateRequest) ([]byte, error) {
	tmpl, err := template.New("code").Funcs(template.FuncMap{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"title": strings.Title,
	}).Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, req); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return []byte(buf.String()), nil
}
