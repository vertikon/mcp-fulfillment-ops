package generators

import (
	"fmt"
)

// RustGenerator generates Rust projects
type RustGenerator struct {
	*BaseGenerator
}

// NewRustGenerator creates a new Rust generator
func NewRustGenerator(templateDir string) *RustGenerator {
	base := NewBaseGenerator("Rust Generator", "rust", templateDir)
	return &RustGenerator{
		BaseGenerator: base,
	}
}

// Validate validates rust generator
func (g *RustGenerator) Validate() error {
	if err := g.BaseGenerator.Validate(); err != nil {
		return err
	}

	if g.stack != "rust" {
		return fmt.Errorf("generator stack mismatch: expected 'rust', got '%s'", g.stack)
	}

	return nil
}

// PythonGenerator generates Python projects
type PythonGenerator struct {
	*BaseGenerator
}

// NewPythonGenerator creates a new Python generator
func NewPythonGenerator(templateDir string) *PythonGenerator {
	base := NewBaseGenerator("Python Generator", "python", templateDir)
	return &PythonGenerator{
		BaseGenerator: base,
	}
}

// Validate validates python generator
func (g *PythonGenerator) Validate() error {
	if err := g.BaseGenerator.Validate(); err != nil {
		return err
	}

	if g.stack != "python" {
		return fmt.Errorf("generator stack mismatch: expected 'python', got '%s'", g.stack)
	}

	return nil
}

// JavaGenerator generates Java projects
type JavaGenerator struct {
	*BaseGenerator
}

// NewJavaGenerator creates a new Java generator
func NewJavaGenerator(templateDir string) *JavaGenerator {
	base := NewBaseGenerator("Java Generator", "java", templateDir)
	return &JavaGenerator{
		BaseGenerator: base,
	}
}

// Validate validates java generator
func (g *JavaGenerator) Validate() error {
	if err := g.BaseGenerator.Validate(); err != nil {
		return err
	}

	if g.stack != "java" {
		return fmt.Errorf("generator stack mismatch: expected 'java', got '%s'", g.stack)
	}

	return nil
}

// CppGenerator generates C++ projects
type CppGenerator struct {
	*BaseGenerator
}

// NewCppGenerator creates a new C++ generator
func NewCppGenerator(templateDir string) *CppGenerator {
	base := NewBaseGenerator("C++ Generator", "cpp", templateDir)
	return &CppGenerator{
		BaseGenerator: base,
	}
}

// Validate validates cpp generator
func (g *CppGenerator) Validate() error {
	if err := g.BaseGenerator.Validate(); err != nil {
		return err
	}

	if g.stack != "cpp" {
		return fmt.Errorf("generator stack mismatch: expected 'cpp', got '%s'", g.stack)
	}

	return nil
}
