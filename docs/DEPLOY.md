# 🚀 Guia de Deploy - MCP Fulfillment Ops

## 📋 Pré-requisitos

- Go 1.24+ instalado
- PostgreSQL 12+ rodando
- NATS Server rodando (opcional, mas recomendado)
- Redis rodando (opcional, mas recomendado)
- `mcp-core-inventory` rodando e acessível

## ⚙️ Configuração

### 1. Variáveis de Ambiente

Copie o arquivo de exemplo e configure:

```bash
cp scripts/setup-env.sh .
chmod +x scripts/setup-env.sh
./scripts/setup-env.sh
```

Edite o arquivo `.env` criado:

```bash
# Database
DATABASE_URL=postgres://user:password@localhost:5432/fulfillment?sslmode=disable

# NATS
NATS_URL=nats://localhost:4222

# Redis
REDIS_URL=redis://localhost:6379

# Core Inventory
CORE_INVENTORY_URL=http://localhost:8081

# HTTP Server
HTTP_PORT=:8080

# Environment
ENV=production
```

### 2. Executar Migrations

```bash
psql $DATABASE_URL < internal/adapters/postgres/migrations/0001_create_fulfillment_tables.sql
```

Ou manualmente:

```bash
psql -h localhost -U postgres -d fulfillment -f internal/adapters/postgres/migrations/0001_create_fulfillment_tables.sql
```

## 🧪 Testes

### Executar Testes Unitários

```bash
go test ./tests/domain/... -v
```

### Validar Contratos OpenAPI

```bash
chmod +x scripts/validate-openapi.sh
./scripts/validate-openapi.sh
```

Ou use o Swagger Editor online:
1. Acesse: https://editor.swagger.io/
2. Cole o conteúdo de `contracts/openapi/bloco-1-core/fulfillment-ops-v1.yaml`

### Testar Integração

```bash
chmod +x scripts/test-integration.sh
export CORE_INVENTORY_URL=http://localhost:8081
export FULFILLMENT_URL=http://localhost:8080
./scripts/test-integration.sh
```

## 🏗️ Build

### Build Local

```bash
go build -o bin/mcp-fulfillment-ops ./cmd/fulfillment-ops
```

### Build para Produção

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/mcp-fulfillment-ops ./cmd/fulfillment-ops
```

## 🐳 Docker

### Build da Imagem

```bash
docker build -t mcp-fulfillment-ops:latest .
```

### Executar Container

```bash
docker run -p 8080:8080 \
  -e DATABASE_URL=postgres://... \
  -e NATS_URL=nats://... \
  -e REDIS_URL=redis://... \
  -e CORE_INVENTORY_URL=http://... \
  mcp-fulfillment-ops:latest
```

Ou usando docker-compose:

```yaml
version: '3.8'
services:
  fulfillment-ops:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://user:password@postgres:5432/fulfillment
      - NATS_URL=nats://nats:4222
      - REDIS_URL=redis://redis:6379
      - CORE_INVENTORY_URL=http://core-inventory:8081
    depends_on:
      - postgres
      - nats
      - redis
```

## 🚀 Deploy Automatizado

Use o script de deploy:

```bash
chmod +x scripts/deploy.sh
./scripts/deploy.sh production
```

O script irá:
1. ✅ Executar testes
2. ✅ Compilar binário
3. ✅ Construir imagem Docker (se Dockerfile existir)
4. ✅ Verificar dependências externas

## 📊 Verificação Pós-Deploy

### Health Check

```bash
curl http://localhost:8080/health
```

Resposta esperada:
```json
{
  "status": "ok",
  "service": "mcp-fulfillment-ops"
}
```

### Testar Endpoint

```bash
# Criar Inbound Shipment
curl -X POST http://localhost:8080/v1/inbound/start \
  -H "Content-Type: application/json" \
  -d '{
    "reference_id": "PO-001",
    "origin": "Fornecedor A",
    "destination": "CD-SP",
    "items": [
      {"sku": "SKU-001", "quantity": 10}
    ]
  }'
```

## 🔍 Troubleshooting

### Erro: "Failed to connect to database"

- Verifique se PostgreSQL está rodando
- Confirme que `DATABASE_URL` está correto
- Verifique permissões do usuário

### Erro: "Failed to connect to NATS"

- Verifique se NATS está rodando: `nats server check`
- Confirme que `NATS_URL` está correto
- O serviço continuará funcionando sem NATS, mas eventos não serão publicados

### Erro: "Failed to connect to Redis"

- Verifique se Redis está rodando: `redis-cli ping`
- Confirme que `REDIS_URL` está correto
- O serviço continuará funcionando sem Redis, mas sem cache

### Erro: "core inventory returned status 500"

- Verifique se `mcp-core-inventory` está rodando
- Confirme que `CORE_INVENTORY_URL` está correto
- Verifique logs do Core Inventory

## 📈 Monitoramento

### Métricas

O serviço expõe métricas em `/metrics` (se configurado):

```bash
curl http://localhost:8080/metrics
```

### Logs

Logs são estruturados em JSON. Para desenvolvimento:

```bash
./bin/mcp-fulfillment-ops 2>&1 | jq
```

## 🔐 Segurança

### Produção

- ✅ Use HTTPS/TLS
- ✅ Configure autenticação/autorização
- ✅ Use secrets management (Vault, AWS Secrets Manager, etc.)
- ✅ Configure rate limiting
- ✅ Habilite CORS apropriadamente
- ✅ Configure firewall rules

### Variáveis Sensíveis

Nunca commite:
- Senhas de banco de dados
- Tokens de API
- Chaves de criptografia
- URLs com credenciais

Use variáveis de ambiente ou secrets management.

## 📚 Referências

- [Blueprint do MCP Fulfillment Ops](../../../../.cursor/BLOCOS/BLOCO-1-BLUEPRINT-MCP-FULFILLMENT-OPS.md)
- [Contratos OpenAPI](../../../../contracts/openapi/bloco-1-core/)
- [Auditoria de Conformidade](../../../../.cursor/AUDITORIA/BLOCO-1-AUDITORIA-CONFORMIDADE-BLOCO-1-BLUEPRINT-MCP-FULFILLMENT-OPS.md)

