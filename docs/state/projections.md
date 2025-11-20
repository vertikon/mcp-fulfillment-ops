# Projections (Projeções)

Este documento descreve o sistema de projeções (projections) do mcp-fulfillment-ops para event sourcing.

## Visão Geral

Projections são visões materializadas derivadas de eventos, permitindo leitura eficiente de dados agregados.

## Conceitos

### Event Sourcing

O event sourcing armazena todos os eventos que ocorreram no sistema, permitindo reconstruir o estado atual.

### Projections

Projections são visões derivadas dos eventos, otimizadas para leitura:

- **Read Models**: Modelos otimizados para leitura
- **Aggregations**: Agregações de dados
- **Denormalized Views**: Visões desnormalizadas

## Tipos de Projections

### 1. Synchronous Projections

Projections atualizadas imediatamente após eventos:

- **Real-time**: Atualização em tempo real
- **Low Latency**: Baixa latência de leitura
- **Consistency**: Consistência forte

### 2. Asynchronous Projections

Projections atualizadas de forma assíncrona:

- **Eventual Consistency**: Consistência eventual
- **High Throughput**: Alta taxa de processamento
- **Scalability**: Escalabilidade horizontal

### 3. Snapshot Projections

Projections com snapshots periódicos:

- **Performance**: Melhor performance de leitura
- **Recovery**: Recuperação rápida após falhas
- **Storage**: Otimização de armazenamento

## Implementação

### Criar Projection

```go
projection := NewProjection("mcp-list", []EventHandler{
    OnMCPCreated,
    OnMCPUpdated,
    OnMCPDeleted,
})
```

### Processar Eventos

```go
projection.Handle(ctx, event)
```

### Ler Projection

```go
mcps, err := projection.Read(ctx, filter)
```

## Configuração

```yaml
state:
  projections:
    enabled: true
    snapshot:
      interval: "1h"
      enabled: true
    async:
      workers: 10
      batch_size: 100
```

## Referências

- [Event Sourcing](./event_sourcing.md)
- [Distributed State](./distributed_state.md)
- [Conflict Resolution](./conflict_resolution.md)

