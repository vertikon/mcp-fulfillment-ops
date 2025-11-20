// Package main provides tree validation tool for mcp-fulfillment-ops
// Validates project structure against official tree definitions
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var (
	originalTreePath   string
	commentedTreePath string
	projectRoot       string
	outputFormat      string
	strictMode        bool
)

var rootCmd = &cobra.Command{
	Use:   "validate-tree",
	Short: "Validate mcp-fulfillment-ops project structure",
	Long:  "Validates project structure against original and commented tree definitions",
	RunE:  runValidation,
}

func init() {
	rootCmd.Flags().StringVarP(&originalTreePath, "original", "o", ".cursor/mcp-fulfillment-ops-ARVORE-FULL.md", "Path to original tree file")
	rootCmd.Flags().StringVarP(&commentedTreePath, "commented", "c", ".cursor/ARVORE-ARQUIVOS-DIRETORIOS-COMENTADA.md", "Path to commented tree file")
	rootCmd.Flags().StringVarP(&projectRoot, "root", "r", ".", "Project root directory")
	rootCmd.Flags().StringVarP(&outputFormat, "format", "f", "json", "Output format: json, markdown, text")
	rootCmd.Flags().BoolVarP(&strictMode, "strict", "s", false, "Strict mode: fail on any missing files")
}

func main() {
	if err := logger.Init("info", true); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		logger.Error("Validation failed", zap.Error(err))
		os.Exit(1)
	}
}

// ValidationResult represents the validation results
type ValidationResult struct {
	Summary          SummaryResult              `json:"summary"`
	OriginalOnly     []string                   `json:"original_only"`
	CommentedOnly    []string                   `json:"commented_only"`
	ImplementationOnly []string                  `json:"implementation_only"`
	Missing          []MissingFile              `json:"missing"`
	Extra            []ExtraFile                 `json:"extra"`
	BlockCompliance  map[string]BlockCompliance `json:"block_compliance"`
	Errors           []string                    `json:"errors,omitempty"`
}

// SummaryResult contains summary statistics
type SummaryResult struct {
	TotalOriginalFiles      int `json:"total_original_files"`
	TotalCommentedFiles     int `json:"total_commented_files"`
	TotalImplementationFiles int `json:"total_implementation_files"`
	CommonFiles             int `json:"common_files"`
	OriginalOnlyCount      int `json:"original_only_count"`
	CommentedOnlyCount     int `json:"commented_only_count"`
	ImplementationOnlyCount int `json:"implementation_only_count"`
	MissingCount           int `json:"missing_count"`
	ExtraCount             int `json:"extra_count"`
	CompliancePercent      float64 `json:"compliance_percent"`
}

// MissingFile represents a file that should exist but doesn't
type MissingFile struct {
	Path     string `json:"path"`
	Expected string `json:"expected"`
	Block    string `json:"block"`
	Severity string `json:"severity"`
}

// ExtraFile represents a file that exists but shouldn't
type ExtraFile struct {
	Path     string `json:"path"`
	Category string `json:"category"`
	Action   string `json:"action"`
}

// BlockCompliance represents compliance status for a block
type BlockCompliance struct {
	Block           string  `json:"block"`
	ExpectedFiles   int     `json:"expected_files"`
	FoundFiles      int     `json:"found_files"`
	MissingFiles    int     `json:"missing_files"`
	CompliancePercent float64 `json:"compliance_percent"`
	Status          string  `json:"status"`
}

func runValidation(cmd *cobra.Command, args []string) error {
	logger.Info("Starting tree validation")

	// Load trees
	originalFiles, err := loadTreeFile(originalTreePath)
	if err != nil {
		return fmt.Errorf("failed to load original tree: %w", err)
	}

	commentedFiles, err := loadTreeFile(commentedTreePath)
	if err != nil {
		return fmt.Errorf("failed to load commented tree: %w", err)
	}

	// Scan implementation
	implementationFiles, err := scanImplementation(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to scan implementation: %w", err)
	}

	// Perform validation
	result := validateTrees(originalFiles, commentedFiles, implementationFiles)

	// Output results
	if err := outputResults(result); err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	// Exit with error code if validation failed
	if strictMode && result.Summary.MissingCount > 0 {
		return fmt.Errorf("validation failed: %d files missing", result.Summary.MissingCount)
	}

	if result.Summary.CompliancePercent < 95.0 {
		logger.Warn("Compliance below threshold", zap.Float64("compliance", result.Summary.CompliancePercent))
		if strictMode {
			return fmt.Errorf("compliance below threshold: %.2f%%", result.Summary.CompliancePercent)
		}
	}

	return nil
}

// loadTreeFile loads and parses a tree file
func loadTreeFile(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	files := make(map[string]bool)
	lines := strings.Split(string(data), "\n")

	// Regex to match file paths in tree format
	fileRegex := regexp.MustCompile(`├──\s+(📄|📁)?\s*([^\s#]+)`)
	dirRegex := regexp.MustCompile(`├──\s+📁\s*([^\s/]+)/`)

	currentPath := ""
	for _, line := range lines {
		// Skip comments and empty lines
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Match files
		if matches := fileRegex.FindStringSubmatch(line); len(matches) > 2 {
			fileName := matches[2]
			// Remove emoji and clean
			fileName = strings.TrimSpace(strings.TrimPrefix(fileName, "📄"))
			fileName = strings.TrimSpace(strings.TrimPrefix(fileName, "📁"))
			
			if fileName != "" && !strings.Contains(fileName, "#") {
				fullPath := filepath.Join(currentPath, fileName)
				files[normalizePath(fullPath)] = true
			}
		}

		// Match directories
		if matches := dirRegex.FindStringSubmatch(line); len(matches) > 1 {
			dirName := matches[1]
			currentPath = filepath.Join(currentPath, dirName)
		}
	}

	return files, nil
}

// scanImplementation scans the actual project directory
func scanImplementation(root string) (map[string]bool, error) {
	files := make(map[string]bool)
	ignoredDirs := map[string]bool{
		".git": true,
		"node_modules": true,
		".cache": true,
		"vendor": true,
		"bin": true,
		"dist": true,
		"build": true,
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories
		if info.IsDir() {
			baseName := filepath.Base(path)
			if ignoredDirs[baseName] {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Normalize path
		normalized := normalizePath(relPath)
		files[normalized] = true

		return nil
	})

	return files, err
}

// normalizePath normalizes a file path for comparison
func normalizePath(path string) string {
	// Convert to forward slashes
	path = filepath.ToSlash(path)
	// Remove leading ./
	if strings.HasPrefix(path, "./") {
		path = path[2:]
	}
	// Remove leading /
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	return strings.ToLower(path)
}

// validateTrees performs the actual validation
func validateTrees(original, commented, implementation map[string]bool) ValidationResult {
	result := ValidationResult{
		BlockCompliance: make(map[string]BlockCompliance),
	}

	// Find files only in original
	for file := range original {
		if !commented[file] && !implementation[file] {
			result.OriginalOnly = append(result.OriginalOnly, file)
		}
	}

	// Find files only in commented
	for file := range commented {
		if !original[file] && !implementation[file] {
			result.CommentedOnly = append(result.CommentedOnly, file)
		}
	}

	// Find files only in implementation
	for file := range implementation {
		if !original[file] && !commented[file] {
			result.ImplementationOnly = append(result.ImplementationOnly, file)
		}
	}

	// Find missing files (in original but not in implementation)
	for file := range original {
		if !implementation[file] {
			block := detectBlock(file)
			severity := "low"
			if isCriticalFile(file) {
				severity = "high"
			}
			result.Missing = append(result.Missing, MissingFile{
				Path:     file,
				Expected: file,
				Block:    block,
				Severity: severity,
			})
		}
	}

	// Find extra files (in implementation but not in original)
	for file := range implementation {
		if !original[file] {
			category := categorizeExtraFile(file)
			action := getActionForExtraFile(category)
			result.Extra = append(result.Extra, ExtraFile{
				Path:     file,
				Category: category,
				Action:   action,
			})
		}
	}

	// Calculate summary
	result.Summary = SummaryResult{
		TotalOriginalFiles:      len(original),
		TotalCommentedFiles:     len(commented),
		TotalImplementationFiles: len(implementation),
		OriginalOnlyCount:       len(result.OriginalOnly),
		CommentedOnlyCount:      len(result.CommentedOnly),
		ImplementationOnlyCount: len(result.ImplementationOnly),
		MissingCount:            len(result.Missing),
		ExtraCount:              len(result.Extra),
	}

	// Calculate compliance
	if len(original) > 0 {
		commonCount := 0
		for file := range original {
			if implementation[file] {
				commonCount++
			}
		}
		result.Summary.CommonFiles = commonCount
		result.Summary.CompliancePercent = float64(commonCount) / float64(len(original)) * 100.0
	}

	// Calculate block compliance
	result.BlockCompliance = calculateBlockCompliance(original, implementation)

	return result
}

// detectBlock detects which block a file belongs to
func detectBlock(path string) string {
	pathLower := strings.ToLower(path)
	
	blocks := map[string]string{
		"cmd/":                    "BLOCO-1",
		"internal/core/":          "BLOCO-1",
		"internal/mcp/":           "BLOCO-2",
		"internal/state/":         "BLOCO-3",
		"internal/monitoring/":    "BLOCO-4",
		"internal/versioning/":    "BLOCO-5",
		"internal/ai/":            "BLOCO-6",
		"internal/infrastructure/": "BLOCO-7",
		"internal/interfaces/":    "BLOCO-8",
		"internal/security/":      "BLOCO-9",
		"templates/":              "BLOCO-10",
		"tools/":                  "BLOCO-11",
		"cmd/mcp-init/":           "BLOCO-11",
		"config/":                 "BLOCO-12",
		"scripts/":                "BLOCO-13",
		"docs/":                   "BLOCO-14",
	}

	for prefix, block := range blocks {
		if strings.HasPrefix(pathLower, prefix) {
			return block
		}
	}

	return "UNKNOWN"
}

// isCriticalFile checks if a file is critical
func isCriticalFile(path string) bool {
	criticalPatterns := []string{
		"main.go",
		"go.mod",
		"config.go",
		"handler.go",
	}
	
	pathLower := strings.ToLower(path)
	for _, pattern := range criticalPatterns {
		if strings.Contains(pathLower, pattern) {
			return true
		}
	}
	return false
}

// categorizeExtraFile categorizes an extra file
func categorizeExtraFile(path string) string {
	pathLower := strings.ToLower(path)
	
	if strings.Contains(pathLower, ".cursor/") {
		return "documentation"
	}
	if strings.Contains(pathLower, "blueprint") || strings.Contains(pathLower, "relatorio") {
		return "documentation"
	}
	if strings.Contains(pathLower, ".md") && !strings.Contains(pathLower, "readme") {
		return "documentation"
	}
	if strings.Contains(pathLower, ".tmp") || strings.Contains(pathLower, ".bak") {
		return "temporary"
	}
	if strings.Contains(pathLower, ".cache") || strings.Contains(pathLower, "coverage") {
		return "build_artifact"
	}
	
	return "unknown"
}

// getActionForExtraFile determines action for extra file
func getActionForExtraFile(category string) string {
	switch category {
	case "documentation":
		return "keep"
	case "temporary":
		return "remove"
	case "build_artifact":
		return "ignore"
	default:
		return "review"
	}
}

// calculateBlockCompliance calculates compliance per block
func calculateBlockCompliance(original, implementation map[string]bool) map[string]BlockCompliance {
	blockFiles := make(map[string]map[string]bool)
	
	// Group files by block
	for file := range original {
		block := detectBlock(file)
		if blockFiles[block] == nil {
			blockFiles[block] = make(map[string]bool)
		}
		blockFiles[block][file] = true
	}
	
	compliance := make(map[string]BlockCompliance)
	
	for block, files := range blockFiles {
		expected := len(files)
		found := 0
		
		for file := range files {
			if implementation[file] {
				found++
			}
		}
		
		missing := expected - found
		compliancePercent := 100.0
		if expected > 0 {
			compliancePercent = float64(found) / float64(expected) * 100.0
		}
		
		status := "✅ Complete"
		if missing > 0 {
			status = "⚠️ Partial"
		}
		if compliancePercent < 50 {
			status = "❌ Incomplete"
		}
		
		compliance[block] = BlockCompliance{
			Block:            block,
			ExpectedFiles:   expected,
			FoundFiles:      found,
			MissingFiles:    missing,
			CompliancePercent: compliancePercent,
			Status:          status,
		}
	}
	
	return compliance
}

// outputResults outputs validation results
func outputResults(result ValidationResult) error {
	switch outputFormat {
	case "json":
		return outputJSON(result)
	case "markdown":
		return outputMarkdown(result)
	case "text":
		return outputText(result)
	default:
		return fmt.Errorf("unknown output format: %s", outputFormat)
	}
}

// outputJSON outputs results as JSON
func outputJSON(result ValidationResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// outputMarkdown outputs results as Markdown
func outputMarkdown(result ValidationResult) error {
	fmt.Println("# Tree Validation Report")
	fmt.Println()
	fmt.Printf("**Compliance:** %.2f%%\n", result.Summary.CompliancePercent)
	fmt.Println()
	fmt.Println("## Summary")
	fmt.Println()
	fmt.Printf("- Original Files: %d\n", result.Summary.TotalOriginalFiles)
	fmt.Printf("- Commented Files: %d\n", result.Summary.TotalCommentedFiles)
	fmt.Printf("- Implementation Files: %d\n", result.Summary.TotalImplementationFiles)
	fmt.Printf("- Common Files: %d\n", result.Summary.CommonFiles)
	fmt.Printf("- Missing Files: %d\n", result.Summary.MissingCount)
	fmt.Printf("- Extra Files: %d\n", result.Summary.ExtraCount)
	fmt.Println()
	
	if len(result.BlockCompliance) > 0 {
		fmt.Println("## Block Compliance")
		fmt.Println()
		fmt.Println("| Block | Expected | Found | Missing | Compliance | Status |")
		fmt.Println("|-------|----------|-------|---------|------------|--------|")
		for _, block := range result.BlockCompliance {
			fmt.Printf("| %s | %d | %d | %d | %.2f%% | %s |\n",
				block.Block, block.ExpectedFiles, block.FoundFiles,
				block.MissingFiles, block.CompliancePercent, block.Status)
		}
		fmt.Println()
	}
	
	return nil
}

// outputText outputs results as plain text
func outputText(result ValidationResult) error {
	fmt.Printf("Tree Validation Report\n")
	fmt.Printf("=====================\n\n")
	fmt.Printf("Compliance: %.2f%%\n\n", result.Summary.CompliancePercent)
	fmt.Printf("Summary:\n")
	fmt.Printf("  Original Files: %d\n", result.Summary.TotalOriginalFiles)
	fmt.Printf("  Commented Files: %d\n", result.Summary.TotalCommentedFiles)
	fmt.Printf("  Implementation Files: %d\n", result.Summary.TotalImplementationFiles)
	fmt.Printf("  Common Files: %d\n", result.Summary.CommonFiles)
	fmt.Printf("  Missing Files: %d\n", result.Summary.MissingCount)
	fmt.Printf("  Extra Files: %d\n\n", result.Summary.ExtraCount)
	
	if len(result.BlockCompliance) > 0 {
		fmt.Println("Block Compliance:")
		for _, block := range result.BlockCompliance {
			fmt.Printf("  %s: %.2f%% (%d/%d) %s\n",
				block.Block, block.CompliancePercent,
				block.FoundFiles, block.ExpectedFiles, block.Status)
		}
	}
	
	return nil
}

