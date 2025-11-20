# Logs

Este documento descreve o sistema de logs do mcp-fulfillment-ops.

## Visão Geral

O sistema de logs registra eventos e atividades do sistema para debugging, auditoria e análise.

## Níveis de Log

### 1. DEBUG

Informações detalhadas para debugging:

- Desenvolvimento e troubleshooting
- Fluxo de execução detalhado
- Valores de variáveis

### 2. INFO

Informações gerais sobre operações:

- Operações bem-sucedidas
- Estados importantes do sistema
- Métricas de operação

### 3. WARN

Avisos sobre situações anômalas:

- Operações que podem falhar
- Configurações não ideais
- Recursos esgotando

### 4. ERROR

Erros que não impedem operação:

- Falhas recuperáveis
- Operações que falharam mas sistema continua
- Timeouts e retries

### 5. FATAL

Erros críticos que impedem operação:

- Falhas não recuperáveis
- Panics
- Situações que requerem intervenção

## Formato de Logs

### Estruturado (JSON)

```json
{
  "timestamp": "2025-01-27T10:00:00Z",
  "level": "info",
  "message": "MCP created successfully",
  "mcp_id": "123e4567-e89b-12d3-a456-426614174000",
  "mcp_name": "my-mcp",
  "trace_id": "abc123",
  "span_id": "def456"
}
```

### Texto

```
2025-01-27T10:00:00Z [INFO] MCP created successfully mcp_id=123e4567 mcp_name=my-mcp
```

## Contexto de Logs

### Trace ID

Identificador único para rastrear requisições através do sistema.

### Span ID

Identificador único para rastrear operações dentro de uma requisição.

### User ID

Identificador do usuário que iniciou a operação.

## Configuração

```yaml
monitoring:
  logs:
    level: "info"  # debug, info, warn, error, fatal
    format: "json"  # json, text
    output: "stdout"  # stdout, file, syslog
    file:
      path: "/var/log/mcp-fulfillment-ops/app.log"
      max_size: "100MB"
      max_backups: 10
      max_age: 30
```

## Referências

- [Observability](./observability.md)
- [Metrics](./metrics.md)
- [Tracing](./tracing.md)

