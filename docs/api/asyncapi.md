# AsyncAPI Documentation

Este documento descreve a documentação AsyncAPI do mcp-fulfillment-ops para eventos assíncronos.

## Visão Geral

O mcp-fulfillment-ops usa eventos assíncronos para comunicação entre serviços, documentados usando AsyncAPI 2.0.

## Especificação

A especificação AsyncAPI está disponível em:

- **YAML**: `docs/api/asyncapi.yaml`
- **JSON**: Pode ser gerado a partir do YAML

## Canais de Eventos

### mcp.created

Evento publicado quando um MCP é criado:

```yaml
channels:
  mcp.created:
    publish:
      message:
        payload:
          type: object
          properties:
            id:
              type: string
            name:
              type: string
            stack:
              type: string
            created_at:
              type: string
              format: date-time
```

### mcp.updated

Evento publicado quando um MCP é atualizado:

```yaml
channels:
  mcp.updated:
    publish:
      message:
        payload:
          type: object
          properties:
            id:
              type: string
            name:
              type: string
            updated_at:
              type: string
              format: date-time
```

### mcp.generated

Evento publicado quando um MCP é gerado:

```yaml
channels:
  mcp.generated:
    publish:
      message:
        payload:
          type: object
          properties:
            job_id:
              type: string
            status:
              type: string
            mcp_id:
              type: string
```

## Servidores

### NATS

```yaml
servers:
  nats:
    url: nats://localhost:4222
    protocol: nats
    description: NATS server for event streaming
```

## Assinaturas

### Assinar Eventos

```go
subscriber.Subscribe("mcp.created", func(event Event) {
    // Process event
})
```

### Publicar Eventos

```go
publisher.Publish("mcp.created", event)
```

## Referências

- [AsyncAPI Specification](https://www.asyncapi.com/)
- [OpenAPI Documentation](./openapi.md)
- [gRPC Documentation](./grpc.md)
- [Messaging Infrastructure](../../internal/infrastructure/messaging/)

