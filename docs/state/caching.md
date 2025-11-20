# Caching (Cache)

Este documento descreve o sistema de cache do mcp-fulfillment-ops.

## Visão Geral

O sistema de cache melhora performance através de armazenamento temporário de dados frequentemente acessados.

## Níveis de Cache

### 1. L1 Cache (In-Memory)

Cache em memória local da aplicação:

- **Velocidade**: Extremamente rápido
- **Capacidade**: Limitada pela RAM
- **Escopo**: Processo único

### 2. L2 Cache (Distributed)

Cache distribuído usando Redis ou similar:

- **Velocidade**: Muito rápido
- **Capacidade**: Escalável
- **Escopo**: Múltiplos processos/servidores

### 3. L3 Cache (Database Query Cache)

Cache de queries de banco de dados:

- **Velocidade**: Rápido
- **Capacidade**: Grande
- **Escopo**: Aplicação inteira

## Estratégias de Cache

### Cache-Aside

A aplicação gerencia o cache:

```
1. Verifica cache
2. Se não encontrado, busca no banco
3. Armazena no cache
4. Retorna dados
```

### Write-Through

Escreve no cache e no banco simultaneamente:

```
1. Escreve no cache
2. Escreve no banco
3. Retorna sucesso
```

### Write-Back

Escreve no cache primeiro, banco depois:

```
1. Escreve no cache
2. Retorna sucesso
3. Escreve no banco assincronamente
```

## Invalidação

### Time-Based

Cache expira após tempo determinado:

```go
cache.Set(ctx, key, value, 5*time.Minute)
```

### Event-Based

Cache invalidado por eventos:

```go
cache.InvalidateOnEvent(ctx, "mcp.updated", key)
```

### Manual

Cache invalidado manualmente:

```go
cache.Invalidate(ctx, key)
```

## Configuração

```yaml
state:
  cache:
    l1:
      enabled: true
      ttl: "5m"
      max_size: "100MB"
    l2:
      enabled: true
      provider: "redis"
      ttl: "1h"
      max_size: "1GB"
```

## Referências

- [State Management](../architecture/blueprint.md)
- [Performance](../architecture/performance.md)
- [Distributed State](./distributed_state.md)

