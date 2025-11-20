# 🚀 Quick Start - Deploy

## ⚡ Início Rápido

### 1. Configurar Ambiente

```powershell
# Windows
.\scripts\setup-env.ps1

# Linux/Mac
chmod +x scripts/setup-env.sh
./scripts/setup-env.sh
```

### 2. Executar Migrations

```bash
psql $DATABASE_URL -f internal/adapters/postgres/migrations/0001_create_fulfillment_tables.sql
```

### 3. Executar Testes

```bash
go test ./tests/domain/... -v
```

### 4. Validar OpenAPI

```powershell
# Windows
.\scripts\validate-openapi.ps1

# Linux/Mac
chmod +x scripts/validate-openapi.sh
./scripts/validate-openapi.sh
```

### 5. Build e Executar

```bash
go build -o bin/mcp-fulfillment-ops ./cmd/fulfillment-ops
./bin/mcp-fulfillment-ops
```

### 6. Testar Integração

```bash
chmod +x scripts/test-integration.sh
./scripts/test-integration.sh
```

## 📚 Documentação Completa

Veja [docs/DEPLOY.md](docs/DEPLOY.md) para documentação detalhada.

## 🔗 Links Úteis

- **Swagger Editor**: https://editor.swagger.io/
- **OpenAPI Validator**: https://validator.swagger.io/
- **Blueprint**: `.cursor/BLOCOS/BLOCO-1-BLUEPRINT-MCP-FULFILLMENT-OPS.md`
- **Auditoria**: `.cursor/AUDITORIA/BLOCO-1-AUDITORIA-CONFORMIDADE-BLOCO-1-BLUEPRINT-MCP-FULFILLMENT-OPS.md`

