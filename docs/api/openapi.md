# OpenAPI Documentation

Este documento descreve a documentação OpenAPI do mcp-fulfillment-ops.

## Visão Geral

A API HTTP do mcp-fulfillment-ops é documentada usando OpenAPI 3.0, permitindo geração automática de clientes e documentação interativa.

## Especificação

A especificação OpenAPI está disponível em:

- **YAML**: `docs/api/openapi.yaml`
- **JSON**: Pode ser gerado a partir do YAML

## Endpoints Principais

### MCP Management

- `POST /api/v1/mcps` - Criar MCP
- `GET /api/v1/mcps` - Listar MCPs
- `GET /api/v1/mcps/{id}` - Obter MCP
- `PUT /api/v1/mcps/{id}` - Atualizar MCP
- `DELETE /api/v1/mcps/{id}` - Deletar MCP

### MCP Generation

- `POST /api/v1/mcps/generate` - Gerar MCP
- `GET /api/v1/mcps/generate/{job_id}` - Status da geração

### MCP Validation

- `POST /api/v1/mcps/validate` - Validar MCP

### Knowledge Management

- `POST /api/v1/knowledge` - Criar base de conhecimento
- `GET /api/v1/knowledge` - Listar bases de conhecimento
- `GET /api/v1/knowledge/{id}` - Obter base de conhecimento

## Autenticação

A API usa autenticação Bearer Token:

```http
Authorization: Bearer <token>
```

## Exemplos

### Criar MCP

```bash
curl -X POST http://localhost:8080/api/v1/mcps \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-mcp",
    "description": "My MCP project",
    "stack": "mcp-go-premium"
  }'
```

### Gerar MCP

```bash
curl -X POST http://localhost:8080/api/v1/mcps/generate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": "go-basic",
    "parameters": {
      "name": "my-mcp",
      "package": "com.example"
    }
  }'
```

## Geração de Clientes

### OpenAPI Generator

```bash
openapi-generator generate \
  -i docs/api/openapi.yaml \
  -g go \
  -o ./clients/go
```

## Referências

- [OpenAPI Specification](https://swagger.io/specification/)
- [gRPC Documentation](./grpc.md)
- [AsyncAPI Documentation](./asyncapi.md)

