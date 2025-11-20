# Exemplo de Template

Este documento demonstra como usar templates no mcp-fulfillment-ops.

## Cenário

Gerar um projeto MCP usando um template Go básico.

## Passo 1: Listar Templates Disponíveis

```bash
# Via CLI
./bin/mcp-cli template list

# Via script
./scripts/generation/generate_template.sh --help
```

## Passo 2: Gerar Projeto do Template

```bash
# Via script
./scripts/generation/generate_template.sh \
  --template go \
  --name my-mcp-project \
  --output ./output/my-mcp
```

## Passo 3: Usar Template Programaticamente

```go
package main

import (
    "context"
    "github.com/vertikon/mcp-fulfillment-ops/tools/generators"
)

func main() {
    ctx := context.Background()
    
    // Criar gerador de template
    generator := generators.NewTemplateGenerator("./templates")
    
    // Request de geração
    req := generators.TemplateGenerateRequest{
        TemplateName: "go",
        ProjectName:  "my-mcp-project",
        OutputPath:   "./output/my-mcp",
        Features:     []string{"http", "grpc"},
        Config: map[string]interface{}{
            "package": "com.example.mcp",
            "version": "1.0.0",
        },
    }
    
    // Gerar projeto
    result, err := generator.GenerateFromTemplate(ctx, req)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Project generated: %s\n", result.Path)
    fmt.Printf("Files created: %d\n", len(result.FilesCreated))
}
```

## Passo 4: Template Customizado

Criar um template customizado em `templates/custom/`:

```
templates/custom/
├── main.go.tmpl
├── config.yaml.tmpl
└── template.yaml
```

### template.yaml

```yaml
name: custom
description: Custom template
stack: go
variables:
  - name: package
    type: string
    required: true
  - name: version
    type: string
    default: "1.0.0"
```

### main.go.tmpl

```go
package {{.package}}

import "fmt"

func main() {
    fmt.Println("Hello from {{.package}} v{{.version}}")
}
```

## Passo 5: Usar Template Customizado

```bash
./scripts/generation/generate_template.sh \
  --template custom \
  --name my-custom-project \
  --output ./output/custom \
  --config config.json
```

## Resultado Esperado

```
Project generated: ./output/my-mcp
Files created: 15
```

## Referências

- [Templates](../architecture/blueprint.md)
- [Generation Scripts](../../scripts/generation/)
- [MCP Example](./mcp_example.md)

