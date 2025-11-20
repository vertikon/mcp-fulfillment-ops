# Metrics (Métricas)

Este documento descreve o sistema de métricas do mcp-fulfillment-ops.

## Visão Geral

O sistema de métricas coleta e expõe métricas sobre o desempenho e comportamento do sistema.

## Tipos de Métricas

### 1. Counter

Contador que só aumenta:

- **Uso**: Contar eventos (requisições, erros, etc.)
- **Exemplo**: `http_requests_total{method="GET", status="200"}`

### 2. Gauge

Valor que pode aumentar ou diminuir:

- **Uso**: Medir estado atual (memória, conexões, etc.)
- **Exemplo**: `memory_usage_bytes`, `active_connections`

### 3. Histogram

Distribuição de valores:

- **Uso**: Medir latência, tamanhos, etc.
- **Exemplo**: `http_request_duration_seconds`

### 4. Summary

Resumo estatístico:

- **Uso**: Similar ao histogram mas com quantis pré-calculados
- **Exemplo**: `http_request_duration_seconds{quantile="0.95"}`

## Métricas Principais

### HTTP Metrics

```
http_requests_total{method, status, endpoint}
http_request_duration_seconds{method, endpoint}
http_request_size_bytes{method, endpoint}
http_response_size_bytes{method, endpoint}
```

### MCP Metrics

```
mcp_created_total{stack}
mcp_generated_total{stack}
mcp_validated_total{status}
mcp_generation_duration_seconds{stack}
```

### System Metrics

```
cpu_usage_percent
memory_usage_bytes
disk_usage_bytes
goroutines_total
```

## Exposição de Métricas

### Prometheus

Métricas expostas em formato Prometheus:

```
GET /metrics
```

### Formato

```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",status="200"} 1234
http_requests_total{method="POST",status="200"} 567
```

## Configuração

```yaml
monitoring:
  metrics:
    enabled: true
    path: "/metrics"
    port: 9090
    interval: "15s"
```

## Referências

- [Observability](./observability.md)
- [Logs](./logs.md)
- [Tracing](./tracing.md)
- [Dashboards](./dashboards.md)

