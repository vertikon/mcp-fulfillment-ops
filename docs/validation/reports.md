# Validation Reports (Relatórios de Validação)

Este documento descreve os relatórios de validação do MCP-HULK.

## Visão Geral

Os relatórios de validação documentam resultados de validações de MCPs, templates e configurações.

## Estrutura do Relatório

### Relatório de Validação

```json
{
  "id": "string (UUID)",
  "type": "mcp|template|config",
  "target": {
    "path": "string",
    "name": "string"
  },
  "status": "passed|failed|warning",
  "validated_at": "timestamp",
  "duration": "duration",
  "checks": [
    {
      "name": "string",
      "status": "passed|failed|warning",
      "message": "string",
      "details": "object"
    }
  ],
  "errors": [
    {
      "code": "string",
      "message": "string",
      "location": "string"
    }
  ],
  "warnings": [
    {
      "code": "string",
      "message": "string",
      "location": "string"
    }
  ]
}
```

## Tipos de Validação

### MCP Validation

Validação de projetos MCP:

- **Structure**: Estrutura de diretórios
- **Dependencies**: Dependências e versões
- **Security**: Segurança e vulnerabilidades
- **Code Quality**: Qualidade do código

### Template Validation

Validação de templates:

- **Structure**: Estrutura do template
- **Variables**: Variáveis e tipos
- **Syntax**: Sintaxe dos templates
- **Completeness**: Completude do template

### Config Validation

Validação de configurações:

- **Schema**: Schema YAML/JSON
- **Values**: Valores e tipos
- **Consistency**: Consistência entre arquivos
- **Required Fields**: Campos obrigatórios

## Geração de Relatórios

### Via Script

```bash
# Validar MCP
./scripts/validation/validate_mcp.sh -p ./my-mcp > validation-report.json

# Validar Template
./scripts/validation/validate_template.sh -p ./template > validation-report.json

# Validar Config
./scripts/validation/validate_config.sh -p ./config.yaml > validation-report.json
```

### Programaticamente

```go
package main

import (
    "context"
    "github.com/vertikon/mcp-fulfillment-ops/tools/validators"
)

func main() {
    ctx := context.Background()
    
    // Validar MCP
    validator := validators.NewMCPValidator()
    result, err := validator.ValidateMCP(ctx, validators.MCPValidateRequest{
        Path:             "./my-mcp",
        StrictMode:       true,
        CheckSecurity:    true,
        CheckDependencies: true,
    })
    if err != nil {
        panic(err)
    }
    
    // Gerar relatório
    report := GenerateReport(result)
    SaveReport(report, "validation-report.json")
}
```

## Armazenamento

Os relatórios são armazenados em `docs/validation/reports/`:

```
docs/validation/reports/
├── 2025-01-27/
│   ├── mcp-validation-001.json
│   ├── template-validation-001.json
│   └── config-validation-001.json
```

## Referências

- [Validation Criteria](./criteria.md)
- [Raw Data](./raw.md)
- [Validation Scripts](../../scripts/validation/)

