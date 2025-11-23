package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// WebGenerator generates web projects (React, Vue, Angular)
type WebGenerator struct {
	*BaseGenerator
	framework string
}

// NewWebGenerator creates a new web generator
func NewWebGenerator(templateDir string) *WebGenerator {
	base := NewBaseGenerator("Web Generator", "web", templateDir)
	return &WebGenerator{
		BaseGenerator: base,
		framework:     "react", // Default framework
	}
}

// SetFramework sets the web framework to use
func (g *WebGenerator) SetFramework(framework string) {
	g.framework = framework
}

// Generate implements Generator interface
func (g *WebGenerator) Generate(ctx interface{}, req GenerateRequest) (*GenerateResult, error) {
	// Determine framework from config
	if fw, ok := req.Config["framework"].(string); ok {
		g.framework = fw
	}

	// Call base generator with web-specific modifications
	result, err := g.BaseGenerator.Generate(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add web-specific post-processing
	if err := g.postProcessWebProject(result.Path, req); err != nil {
		return nil, fmt.Errorf("Web post-processing failed: %w", err)
	}

	return result, nil
}

// getTemplateFiles returns web-specific template files
func (g *WebGenerator) getTemplateFiles(req GenerateRequest) ([]TemplateFile, error) {
	files := []TemplateFile{
		// Basic project files
		{
			SourcePath: filepath.Join(g.templateDir, "package.json.tmpl"),
			TargetPath: "package.json",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "package-lock.json"),
			TargetPath: "package-lock.json",
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
			SourcePath: filepath.Join(g.templateDir, "Dockerfile.tmpl"),
			TargetPath: "Dockerfile",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "docker-compose.yml.tmpl"),
			TargetPath: "docker-compose.yml",
			IsTemplate: true,
		},

		// Configuration files
		{
			SourcePath: filepath.Join(g.templateDir, "vite.config.ts.tmpl"),
			TargetPath: "vite.config.ts",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "tsconfig.json.tmpl"),
			TargetPath: "tsconfig.json",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "tailwind.config.js.tmpl"),
			TargetPath: "tailwind.config.js",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, ".eslintrc.js.tmpl"),
			TargetPath: ".eslintrc.js",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, ".prettierrc.tmpl"),
			TargetPath: ".prettierrc",
			IsTemplate: true,
		},

		// Public files
		{
			SourcePath: filepath.Join(g.templateDir, "public", "index.html.tmpl"),
			TargetPath: "public/index.html",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "public", "favicon.ico"),
			TargetPath: "public/favicon.ico",
			IsTemplate: false,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "public", "manifest.json.tmpl"),
			TargetPath: "public/manifest.json",
			IsTemplate: true,
		},

		// Source files
		{
			SourcePath: filepath.Join(g.templateDir, "src", "main.tsx.tmpl"),
			TargetPath: "src/main.tsx",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "src", "App.tsx.tmpl"),
			TargetPath: "src/App.tsx",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "src", "index.css.tmpl"),
			TargetPath: "src/index.css",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "src", "vite-env.d.ts"),
			TargetPath: "src/vite-env.d.ts",
			IsTemplate: false,
		},

		// Component directories and files
		{
			SourcePath: filepath.Join(g.templateDir, "src", "components", "ui", "Button.tsx.tmpl"),
			TargetPath: "src/components/ui/Button.tsx",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "src", "components", "ui", "Card.tsx.tmpl"),
			TargetPath: "src/components/ui/Card.tsx",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "src", "components", "layouts", "Header.tsx.tmpl"),
			TargetPath: "src/components/layouts/Header.tsx",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "src", "components", "layouts", "Footer.tsx.tmpl"),
			TargetPath: "src/components/layouts/Footer.tsx",
			IsTemplate: true,
		},

		// Hooks
		{
			SourcePath: filepath.Join(g.templateDir, "src", "hooks", "useApi.ts.tmpl"),
			TargetPath: "src/hooks/useApi.ts",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "src", "hooks", "useLocalStorage.ts.tmpl"),
			TargetPath: "src/hooks/useLocalStorage.ts",
			IsTemplate: true,
		},
	}

	// Add framework-specific files
	switch g.framework {
	case "react":
		files = append(files, []TemplateFile{
			{
				SourcePath: filepath.Join(g.templateDir, "react", "src", "App.tsx.tmpl"),
				TargetPath: "src/App.tsx",
				IsTemplate: true,
			},
			{
				SourcePath: filepath.Join(g.templateDir, "react", "src", "main.tsx.tmpl"),
				TargetPath: "src/main.tsx",
				IsTemplate: true,
			},
		}...)
	case "vue":
		files = append(files, []TemplateFile{
			{
				SourcePath: filepath.Join(g.templateDir, "vue", "src", "App.vue.tmpl"),
				TargetPath: "src/App.vue",
				IsTemplate: true,
			},
			{
				SourcePath: filepath.Join(g.templateDir, "vue", "src", "main.ts.tmpl"),
				TargetPath: "src/main.ts",
				IsTemplate: true,
			},
		}...)
	case "angular":
		files = append(files, []TemplateFile{
			{
				SourcePath: filepath.Join(g.templateDir, "angular", "src", "app", "app.component.ts.tmpl"),
				TargetPath: "src/app/app.component.ts",
				IsTemplate: true,
			},
			{
				SourcePath: filepath.Join(g.templateDir, "angular", "src", "app", "app.module.ts.tmpl"),
				TargetPath: "src/app/app.module.ts",
				IsTemplate: true,
			},
		}...)
	}

	// Add feature-specific files
	if g.hasFeature(req.Features, "testing") {
		files = append(files, []TemplateFile{
			{
				SourcePath: filepath.Join(g.templateDir, "src", "components", "ui", "Button.test.tsx.tmpl"),
				TargetPath: "src/components/ui/Button.test.tsx",
				IsTemplate: true,
			},
			{
				SourcePath: filepath.Join(g.templateDir, "src", "App.test.tsx.tmpl"),
				TargetPath: "src/App.test.tsx",
				IsTemplate: true,
			},
			{
				SourcePath: filepath.Join(g.templateDir, "src", "setupTests.ts.tmpl"),
				TargetPath: "src/setupTests.ts",
				IsTemplate: true,
			},
		}...)
	}

	return files, nil
}

// hasFeature checks if a feature is enabled
func (g *WebGenerator) hasFeature(features []string, feature string) bool {
	for _, f := range features {
		if strings.EqualFold(f, feature) {
			return true
		}
	}
	return false
}

// postProcessWebProject performs web-specific post-processing
func (g *WebGenerator) postProcessWebProject(projectPath string, req GenerateRequest) error {
	g.logger.Info("Starting web project post-processing",
		zap.String("project", req.Name),
		zap.String("path", projectPath),
		zap.String("framework", g.framework),
		zap.Strings("features", req.Features))

	// Verify package.json exists
	packageJsonPath := filepath.Join(projectPath, "package.json")
	if _, err := os.Stat(packageJsonPath); os.IsNotExist(err) {
		g.logger.Warn("package.json not found, skipping post-processing",
			zap.String("path", packageJsonPath))
		return nil
	}

	// Verify required directories exist
	requiredDirs := []string{"src", "public", "configs"}
	for _, dir := range requiredDirs {
		dirPath := filepath.Join(projectPath, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			g.logger.Debug("Optional directory not found",
				zap.String("directory", dir))
		}
	}

	// Verify framework-specific files
	switch g.framework {
	case "react":
		reactFiles := []string{"src/index.tsx", "src/App.tsx"}
		for _, file := range reactFiles {
			filePath := filepath.Join(projectPath, file)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				g.logger.Debug("React file not found",
					zap.String("file", file))
			}
		}
	case "vue":
		vueFiles := []string{"src/main.ts", "src/App.vue"}
		for _, file := range vueFiles {
			filePath := filepath.Join(projectPath, file)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				g.logger.Debug("Vue file not found",
					zap.String("file", file))
			}
		}
	}

	g.logger.Info("Web project post-processing completed",
		zap.String("project", req.Name),
		zap.String("framework", g.framework),
		zap.Strings("features", req.Features))

	return nil
}

// Validate validates web generator
func (g *WebGenerator) Validate() error {
	if err := g.BaseGenerator.Validate(); err != nil {
		return err
	}

	// Web-specific validation
	if g.stack != "web" {
		return fmt.Errorf("generator stack mismatch: expected 'web', got '%s'", g.stack)
	}

	validFrameworks := []string{"react", "vue", "angular"}
	if g.framework == "" {
		g.framework = "react" // Default to React
	}

	validFramework := false
	for _, fw := range validFrameworks {
		if g.framework == fw {
			validFramework = true
			break
		}
	}

	if !validFramework {
		return fmt.Errorf("invalid framework: %s, valid options: %v", g.framework, validFrameworks)
	}

	return nil
}
