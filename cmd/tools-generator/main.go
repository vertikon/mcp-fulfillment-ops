package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vertikon/mcp-fulfillment-ops/tools/generators"
)

func main() {
	var toolType, name, stack, path, outputPath, configPath string
	var features string
	var help bool

	flag.StringVar(&toolType, "type", "", "Tool type: mcp, template, config, code")
	flag.StringVar(&name, "name", "", "Name of the resource to generate")
	flag.StringVar(&stack, "stack", "", "Stack/template type (for MCP/template generation)")
	flag.StringVar(&path, "path", "", "Output path")
	flag.StringVar(&outputPath, "output", "", "Output path (alias for path)")
	flag.StringVar(&configPath, "config", "", "Path to JSON config file")
	flag.StringVar(&features, "features", "", "Comma-separated list of features")
	flag.BoolVar(&help, "help", false, "Show help")
	flag.Parse()

	if help || toolType == "" {
		usage()
		os.Exit(1)
	}

	// Use outputPath if path is empty
	if path == "" {
		path = outputPath
	}

	// Get project root (assume we're in cmd/tools-generator)
	projectRoot := filepath.Join(filepath.Dir(os.Args[0]), "../..")
	if absRoot, err := filepath.Abs(projectRoot); err == nil {
		projectRoot = absRoot
	}

	ctx := context.Background()

	var result interface{}
	var err error

	switch toolType {
	case "mcp":
		if name == "" || path == "" {
			fmt.Fprintf(os.Stderr, "Error: name and path are required for MCP generation\n")
			os.Exit(1)
		}
		req := generators.MCPGenerateRequest{
			Name:  name,
			Stack: stack,
			Path:  path,
		}
		if features != "" {
			req.Features = splitFeatures(features)
		}
		if configPath != "" {
			req.Config = loadConfig(configPath)
		}
		gen := generators.NewMCPGenerator(filepath.Join(projectRoot, "templates"))
		result, err = gen.GenerateMCP(ctx, req)

	case "template":
		if name == "" || path == "" {
			fmt.Fprintf(os.Stderr, "Error: name and path are required for template generation\n")
			os.Exit(1)
		}
		if stack == "" {
			fmt.Fprintf(os.Stderr, "Error: template name (--stack) is required\n")
			os.Exit(1)
		}
		req := generators.TemplateGenerateRequest{
			TemplateName: stack,
			ProjectName:  name,
			OutputPath:   path,
		}
		if features != "" {
			req.Features = splitFeatures(features)
		}
		if configPath != "" {
			req.Config = loadConfig(configPath)
		}
		gen := generators.NewTemplateGenerator(filepath.Join(projectRoot, "templates"))
		result, err = gen.GenerateFromTemplate(ctx, req)

	case "config":
		if name == "" || path == "" {
			fmt.Fprintf(os.Stderr, "Error: name, type and path are required for config generation\n")
			os.Exit(1)
		}
		if stack == "" {
			fmt.Fprintf(os.Stderr, "Error: config type (--stack) is required\n")
			os.Exit(1)
		}
		req := generators.ConfigGenerateRequest{
			Name:       name,
			Type:       stack,
			OutputPath: path,
		}
		if configPath != "" {
			req.Config = loadConfig(configPath)
		}
		gen := generators.NewConfigGenerator()
		result, err = gen.GenerateConfig(ctx, req)

	case "code":
		if name == "" || path == "" {
			fmt.Fprintf(os.Stderr, "Error: name, type and path are required for code generation\n")
			os.Exit(1)
		}
		if stack == "" {
			fmt.Fprintf(os.Stderr, "Error: code type (--stack) is required\n")
			os.Exit(1)
		}
		req := generators.CodeGenerateRequest{
			Name:       name,
			Type:       stack,
			OutputPath: path,
		}
		if configPath != "" {
			req.Config = loadConfig(configPath)
		}
		gen := generators.NewCodeGenerator()
		result, err = gen.GenerateCode(ctx, req)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown tool type: %s\n", toolType)
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Output result as JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling result: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: %s -type TYPE [OPTIONS]

Tool types:
  mcp      - Generate MCP project
  template - Generate template
  config   - Generate configuration file
  code     - Generate code file

Options:
  -type TYPE        Tool type (required)
  -name NAME        Name of the resource (required)
  -stack STACK      Stack/template type (for MCP/template)
  -path PATH        Output path (required)
  -output PATH      Output path (alias for -path)
  -config PATH      Path to JSON config file
  -features LIST    Comma-separated list of features
  -help             Show this help

Examples:
  %s -type mcp -name my-mcp -path ./output -stack mcp-go-premium
  %s -type template -name my-template -path ./output -stack go
  %s -type config -name app -path ./config.yaml -stack yaml
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func splitFeatures(features string) []string {
	if features == "" {
		return nil
	}
	var result []string
	for _, f := range split(features, ",") {
		if trimmed := trimSpace(f); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func loadConfig(path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}
	return config
}

func split(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
