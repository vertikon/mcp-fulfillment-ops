package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/vertikon/mcp-fulfillment-ops/tools/validators"
)

func main() {
	var toolType, path string
	var strictMode, checkSecurity, checkDependencies bool
	var help bool

	flag.StringVar(&toolType, "type", "", "Tool type: mcp, template, config, code")
	flag.StringVar(&path, "path", "", "Path to validate (required)")
	flag.BoolVar(&strictMode, "strict", false, "Enable strict mode")
	flag.BoolVar(&checkSecurity, "security", false, "Check security")
	flag.BoolVar(&checkDependencies, "dependencies", false, "Check dependencies")
	flag.BoolVar(&help, "help", false, "Show help")
	flag.Parse()

	if help || toolType == "" || path == "" {
		usage()
		if path == "" {
			os.Exit(1)
		}
	}

	ctx := context.Background()

	var result interface{}
	var err error

	switch toolType {
	case "mcp":
		req := validators.MCPValidateRequest{
			Path:              path,
			StrictMode:        strictMode,
			CheckSecurity:     checkSecurity,
			CheckDependencies: checkDependencies,
		}
		validator := validators.NewMCPValidator()
		result, err = validator.ValidateMCP(ctx, req)

	case "template":
		req := validators.TemplateValidateRequest{
			Path:       path,
			StrictMode: strictMode,
		}
		validator := validators.NewTemplateValidator()
		result, err = validator.ValidateTemplate(ctx, req)

	case "config":
		req := validators.ConfigValidateRequest{
			Path: path,
			// StrictMode removido - não existe no struct
		}
		validator := validators.NewConfigValidator()
		result, err = validator.ValidateConfig(ctx, req)

	case "code":
		req := validators.CodeValidateRequest{
			Path:       path,
			StrictMode: strictMode,
		}
		validator := validators.NewCodeValidator()
		result, err = validator.ValidateCode(ctx, req)

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

	// Exit with error code if validation failed
	// Check if result has Valid field (common in validation results)
	if resultMap, ok := result.(map[string]interface{}); ok {
		if valid, ok := resultMap["valid"].(bool); ok && !valid {
			os.Exit(1)
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: %s -type TYPE -path PATH [OPTIONS]

Tool types:
  mcp      - Validate MCP project
  template - Validate template
  config   - Validate configuration file
  code     - Validate code

Options:
  -type TYPE        Tool type (required)
  -path PATH        Path to validate (required)
  -strict           Enable strict mode
  -security         Check security (for MCP)
  -dependencies     Check dependencies (for MCP)
  -help             Show this help

Examples:
  %s -type mcp -path ./my-mcp -strict -security
  %s -type template -path ./my-template -strict
  %s -type config -path ./config.yaml
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}
