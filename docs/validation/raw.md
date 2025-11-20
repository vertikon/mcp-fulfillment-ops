# Raw Validation Data (Dados Brutos de Validação)

Este documento descreve os dados brutos de validação do mcp-fulfillment-ops.

## Visão Geral

Os dados brutos de validação contêm informações detalhadas e não processadas sobre validações, úteis para análise profunda e debugging.

## Estrutura dos Dados Brutos

### Raw Validation Data

```json
{
  "validation_id": "string (UUID)",
  "timestamp": "timestamp",
  "type": "mcp|template|config",
  "target": {
    "path": "string",
    "name": "string",
    "type": "string"
  },
  "raw_data": {
    "files_scanned": [
      {
        "path": "string",
        "size": "number",
        "checksum": "string"
      }
    ],
    "dependencies": [
      {
        "name": "string",
        "version": "string",
        "type": "string"
      }
    ],
    "code_metrics": {
      "lines_of_code": "number",
      "functions": "number",
      "complexity": "number"
    },
    "security_scan": {
      "vulnerabilities": [
        {
          "id": "string",
          "severity": "string",
          "description": "string"
        }
      ]
    }
  },
  "metadata": {
    "validator_version": "string",
    "checks_performed": ["string"],
    "duration_ms": "number"
  }
}
```

## Tipos de Dados Brutos

### File Scan Data

Dados sobre arquivos escaneados:

- **Paths**: Caminhos dos arquivos
- **Sizes**: Tamanhos dos arquivos
- **Checksums**: Checksums para integridade
- **Types**: Tipos de arquivos

### Dependency Data

Dados sobre dependências:

- **Names**: Nomes das dependências
- **Versions**: Versões das dependências
- **Types**: Tipos (direct, indirect)
- **Licenses**: Licenças das dependências

### Code Metrics Data

Métricas de código:

- **Lines of Code**: Linhas de código
- **Functions**: Número de funções
- **Complexity**: Complexidade ciclomática
- **Coverage**: Cobertura de testes

### Security Scan Data

Dados de scan de segurança:

- **Vulnerabilities**: Vulnerabilidades encontradas
- **Severities**: Níveis de severidade
- **CVEs**: CVE IDs quando aplicável
- **Recommendations**: Recomendações de correção

## Armazenamento

Os dados brutos são armazenados em `docs/validation/raw/`:

```
docs/validation/raw/
├── 2025-01-27/
│   ├── mcp-validation-001-raw.json
│   ├── template-validation-001-raw.json
│   └── config-validation-001-raw.json
```

## Uso

### Acessar Dados Brutos

```bash
# Via script de validação
./scripts/validation/validate_mcp.sh -p ./my-mcp --raw > raw-data.json
```

### Processar Dados Brutos

```go
package main

import (
    "encoding/json"
    "os"
)

func main() {
    // Ler dados brutos
    data, err := os.ReadFile("raw-data.json")
    if err != nil {
        panic(err)
    }
    
    var rawData RawValidationData
    json.Unmarshal(data, &rawData)
    
    // Processar dados
    ProcessRawData(rawData)
}
```

## Referências

- [Validation Criteria](./criteria.md)
- [Validation Reports](./reports.md)
- [Validation Scripts](../../scripts/validation/)

