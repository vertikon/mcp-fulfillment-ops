# 🐳 Docker - Quick Start

## ⚡ Executar com Docker Compose

```bash
# Subir todos os serviços
docker-compose up -d

# Ver logs
docker-compose logs -f fulfillment-ops

# Parar
docker-compose down
```

## 🔨 Build Manual

```bash
# Build da imagem
docker build -t mcp-fulfillment-ops:latest .

# Executar
docker run -p 8080:8080 \
  -e DATABASE_URL=postgres://... \
  -e NATS_URL=nats://... \
  -e REDIS_URL=redis://... \
  -e CORE_INVENTORY_URL=http://... \
  mcp-fulfillment-ops:latest
```

## 📚 Documentação Completa

Veja [docs/DOCKER.md](docs/DOCKER.md) para documentação detalhada.

## ✅ Verificar

```bash
# Health check
curl http://localhost:8080/health

# Status dos containers
docker-compose ps
```

