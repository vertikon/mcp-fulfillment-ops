# MCP Schema

Este documento descreve o schema do protocolo MCP (Model Context Protocol) usado no mcp-fulfillment-ops.

## Estrutura do Schema

### Entidade MCP

O MCP é a entidade central do protocolo, representando um projeto completo de Model Context Protocol.

```json
{
  "id": "string (UUID)",
  "name": "string (required, unique)",
  "description": "string (optional)",
  "stack": "string (required)",
  "path": "string (optional)",
  "features": "array of Feature objects",
  "context": "KnowledgeContext object (optional)",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

### Stack Types

Os tipos de stack suportados:

- `mcp-go-premium` - Stack Go premium com todas as funcionalidades
- `go` - Stack Go básico
- `tinygo` - Stack TinyGo para WebAssembly
- `wasm` - Stack WebAssembly
- `web` - Stack Web (TypeScript/React)

### Feature Object

```json
{
  "name": "string",
  "enabled": "boolean",
  "config": "object (optional)"
}
```

### KnowledgeContext Object

```json
{
  "knowledge_id": "string",
  "documents": "array of strings",
  "embeddings": "object (map of string to float arrays)",
  "metadata": "object"
}
```

## Schema de Banco de Dados

### Tabela `mcps`

```sql
CREATE TABLE IF NOT EXISTS mcps (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    stack VARCHAR(50) NOT NULL,
    path VARCHAR(500),
    features JSONB,
    context JSONB,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_mcps_stack ON mcps(stack);
CREATE INDEX IF NOT EXISTS idx_mcps_name ON mcps(name);
CREATE INDEX IF NOT EXISTS idx_mcps_created_at ON mcps(created_at);
```

## Validação do Schema

O schema é validado através de:

1. **Validação de Entidade**: Verifica campos obrigatórios e tipos
2. **Validação de Stack**: Verifica se o stack é suportado
3. **Validação de Features**: Verifica estrutura das features
4. **Validação de Context**: Verifica estrutura do knowledge context

## Referências

- [Protocolo MCP](./protocol.md)
- [Registry MCP](./registry.md)
- [Tools MCP](./tools.md)
- [Handlers MCP](./handlers.md)

