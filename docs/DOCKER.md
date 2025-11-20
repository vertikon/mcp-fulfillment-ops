# 🐳 Guia Docker - MCP Fulfillment Ops

## 📋 Visão Geral

O `mcp-fulfillment-ops` está completamente containerizado e pronto para deploy via Docker.

## 🏗️ Arquivos Docker

- **Dockerfile** - Build multi-stage otimizado
- **docker-compose.yml** - Ambiente completo com dependências (dev/staging)
- **docker-compose.prod.yml** - Configuração de produção
- **.dockerignore** - Otimização de build

## 🚀 Quick Start

### Build da Imagem

```bash
# Linux/Mac
chmod +x scripts/docker-build.sh
./scripts/docker-build.sh

# Windows
.\scripts\docker-build.ps1
```

Ou manualmente:

```bash
docker build -t mcp-fulfillment-ops:latest .
```

### Executar com Docker Compose (Recomendado)

```bash
# Subir todos os serviços (Postgres, NATS, Redis, Fulfillment Ops)
docker-compose up -d

# Ver logs
docker-compose logs -f fulfillment-ops

# Parar serviços
docker-compose down

# Parar e remover volumes
docker-compose down -v
```

### Executar Standalone

```bash
docker run -d \
  --name mcp-fulfillment-ops \
  -p 8080:8080 \
  -e DATABASE_URL=postgres://user:password@host:5432/fulfillment \
  -e NATS_URL=nats://host:4222 \
  -e REDIS_URL=redis://host:6379 \
  -e CORE_INVENTORY_URL=http://host:8081 \
  mcp-fulfillment-ops:latest
```

## 🔧 Configuração

### Variáveis de Ambiente

O container aceita as seguintes variáveis de ambiente:

| Variável | Descrição | Default |
|----------|-----------|---------|
| `DATABASE_URL` | URL de conexão PostgreSQL | - |
| `NATS_URL` | URL do servidor NATS | `nats://localhost:4222` |
| `REDIS_URL` | URL do servidor Redis | `redis://localhost:6379` |
| `CORE_INVENTORY_URL` | URL do mcp-core-inventory | `http://localhost:8081` |
| `HTTP_PORT` | Porta HTTP do serviço | `:8080` |
| `ENV` | Ambiente (development/staging/production) | `development` |

### Docker Compose - Desenvolvimento

O `docker-compose.yml` inclui:

- **PostgreSQL 15** - Banco de dados
- **NATS 2.10** - Message broker
- **Redis 7** - Cache e locks
- **mcp-fulfillment-ops** - Serviço principal

Todas as dependências são configuradas automaticamente com health checks.

### Docker Compose - Produção

Use `docker-compose.prod.yml` para produção:

```bash
docker-compose -f docker-compose.prod.yml up -d
```

**Diferenças:**
- Não inclui dependências (assume que já existem)
- Configurações de recursos (CPU/Memory limits)
- Logging configurado
- Restart policy: always

## 🏥 Health Checks

O container inclui health check configurado:

```bash
# Verificar status
docker ps

# Ver logs do health check
docker inspect mcp-fulfillment-ops | jq '.[0].State.Health'
```

## 📊 Monitoramento

### Logs

```bash
# Logs em tempo real
docker-compose logs -f fulfillment-ops

# Últimas 100 linhas
docker-compose logs --tail=100 fulfillment-ops

# Logs com timestamp
docker-compose logs -t fulfillment-ops
```

### Métricas

O serviço expõe endpoint `/health`:

```bash
curl http://localhost:8080/health
```

## 🔍 Troubleshooting

### Container não inicia

```bash
# Ver logs
docker logs mcp-fulfillment-ops

# Verificar variáveis de ambiente
docker inspect mcp-fulfillment-ops | jq '.[0].Config.Env'
```

### Erro de conexão com banco

```bash
# Verificar se Postgres está rodando
docker-compose ps postgres

# Testar conexão
docker-compose exec postgres psql -U fulfillment -d fulfillment -c "SELECT 1"
```

### Erro de conexão com NATS

```bash
# Verificar se NATS está rodando
docker-compose ps nats

# Testar conexão
docker-compose exec nats nats server check
```

### Rebuild após mudanças

```bash
# Rebuild sem cache
docker-compose build --no-cache fulfillment-ops

# Restart
docker-compose restart fulfillment-ops
```

## 🚢 Deploy em Produção

### 1. Build da Imagem

```bash
docker build -t registry.example.com/mcp-fulfillment-ops:v1.0.0 .
```

### 2. Push para Registry

```bash
docker push registry.example.com/mcp-fulfillment-ops:v1.0.0
```

### 3. Deploy

```bash
# Usar docker-compose.prod.yml
docker-compose -f docker-compose.prod.yml pull
docker-compose -f docker-compose.prod.yml up -d
```

### 4. Verificar Deploy

```bash
# Health check
curl https://api.example.com/health

# Ver logs
docker-compose -f docker-compose.prod.yml logs -f fulfillment-ops
```

## 🔐 Segurança

### Boas Práticas Implementadas

- ✅ Non-root user no container
- ✅ Multi-stage build (imagem mínima)
- ✅ Health checks configurados
- ✅ Secrets via variáveis de ambiente
- ✅ Networks isoladas

### Recomendações Adicionais

- Use secrets management (Docker Secrets, Vault, etc.)
- Configure TLS/HTTPS
- Use image scanning (Trivy, Snyk)
- Configure resource limits
- Use read-only filesystem quando possível

## 📚 Referências

- [Dockerfile](Dockerfile)
- [docker-compose.yml](../docker-compose.yml)
- [docker-compose.prod.yml](../docker-compose.prod.yml)
- [Guia de Deploy](DEPLOY.md)

