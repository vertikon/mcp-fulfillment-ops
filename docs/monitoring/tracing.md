# Tracing (Rastreamento)

Este documento descreve o sistema de tracing do mcp-fulfillment-ops.

## Visão Geral

O sistema de tracing rastreia requisições através de múltiplos serviços e componentes para debugging e análise de performance.

## Conceitos

### Trace

Um trace representa uma requisição completa através do sistema:

- **Trace ID**: Identificador único do trace
- **Spans**: Operações individuais dentro do trace
- **Duration**: Tempo total da requisição

### Span

Um span representa uma operação individual:

- **Span ID**: Identificador único do span
- **Parent Span**: Span pai (para hierarquia)
- **Operation Name**: Nome da operação
- **Start/End Time**: Tempo de início e fim
- **Tags**: Metadados adicionais
- **Logs**: Eventos dentro do span

## Hierarquia de Spans

```
Trace
├── Span: HTTP Request
│   ├── Span: Database Query
│   ├── Span: MCP Generation
│   │   ├── Span: Template Processing
│   │   └── Span: Code Generation
│   └── Span: Validation
```

## Instrumentação

### HTTP Handler

```go
span := tracer.StartSpan("http.request")
defer span.Finish()

span.SetTag("http.method", r.Method)
span.SetTag("http.url", r.URL.String())
```

### Database Query

```go
span := tracer.StartSpan("db.query", opentracing.ChildOf(parentSpan.Context()))
defer span.Finish()

span.SetTag("db.statement", query)
span.SetTag("db.type", "postgresql")
```

## Exportação

### Jaeger

Traces exportados para Jaeger:

```yaml
tracing:
  exporter: "jaeger"
  endpoint: "http://jaeger:14268/api/traces"
  service_name: "mcp-fulfillment-ops"
```

### OpenTelemetry

Traces exportados via OpenTelemetry:

```yaml
tracing:
  exporter: "otlp"
  endpoint: "http://otel-collector:4317"
  service_name: "mcp-fulfillment-ops"
```

## Configuração

```yaml
monitoring:
  tracing:
    enabled: true
    sampler:
      type: "probabilistic"
      param: 0.1  # Sample 10% of traces
    exporter: "jaeger"
    endpoint: "http://jaeger:14268/api/traces"
```

## Referências

- [Observability](./observability.md)
- [Logs](./logs.md)
- [Metrics](./metrics.md)
- [Dashboards](./dashboards.md)

