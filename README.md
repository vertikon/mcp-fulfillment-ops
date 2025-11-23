# MCP Fulfillment Ops

Serviço de orquestração de operações logísticas do ecossistema Vertikon. Responsável por gerenciar o fluxo físico de produtos desde o recebimento até a expedição.

## 🚀 Funcionalidades

- **Gestão de Ordens de Fulfillment**: Criação e acompanhamento de ordens de expedição
- **Processamento de Inbound**: Recebimento e armazenamento de produtos
- **Picking e Packing**: Separação e embalagem de pedidos
- **Shipping**: Expedição e rastreio de entregas
- **Gestão de Devoluções**: Processamento de retornos e reposições
- **Controle de Estoque Físico**: Sincronização com o Core Inventory
- **Integração por Eventos**: Comunicação assíncrona via NATS JetStream

## 📁 Estrutura do Projeto

O projeto segue **Clean Architecture** com foco em domínio logístico:

```
├── cmd/                    # Pontos de entrada da aplicação
├── internal/               # Código privado da aplicação
│   ├── fulfillment/        # Domínio de logística
│   │   ├── entities/       # Entidades principais
│   │   ├── services/       # Serviços de domínio
│   │   └── repositories/   # Interfaces de persistência
│   ├── adapters/           # Adaptadores externos
│   ├── app/               # Configuração e inicialização
│   └── infrastructure/    # Infraestrutura externa
├── pkg/                   # Bibliotecas públicas
├── config/                # Arquivos de configuração
├── scripts/               # Scripts de automação
└── docs/                  # Documentação
```

## 🛠️ Quick Start

### Pré-requisitos

- Go 1.21 ou superior
- Docker (opcional)
- PostgreSQL
- Redis
- NATS JetStream

### Instalação

```bash
# Clonar o repositório
git clone https://github.com/vertikon/mcp-fulfillment-ops.git
cd mcp-fulfillment-ops

# Instalar dependências
make deps

# Construir a aplicação
make build
```

### Executando a Aplicação

```bash
# Executar com configuração padrão
make run

# Ou usando Go diretamente
go run ./cmd/main.go
```

## 🎯 Funcionalidades Principais

### Ordens de Fulfillment

O serviço gerencia o ciclo de vida completo das ordens de expedição:

```bash
# Criar nova ordem de fulfillment
curl -X POST http://localhost:8080/api/v1/fulfillment-orders \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "ORD-12345",
    "customer": "CUSTOMER-001",
    "destination": "Rua A, 123 - São Paulo/SP",
    "items": [
      {"sku": "PROD-001", "quantity": 2},
      {"sku": "PROD-002", "quantity": 1}
    ],
    "priority": 0
  }'

# Iniciar processo de picking
curl -X POST http://localhost:8080/api/v1/fulfillment-orders/{id}/pick

# Confirmar expedição
curl -X POST http://localhost:8080/api/v1/fulfillment-orders/{id}/ship
```

### Eventos NATS

O serviço publica e consome eventos via NATS JetStream:

**Eventos Publicados:**
- `fulfillment.order.created`
- `fulfillment.order.picked`
- `fulfillment.order.shipped`
- `fulfillment.inventory.updated`

**Eventos Consumidos:**
- `oms.order.ready_to_pick`
- `inventory.reservation.confirmed`
- `inventory.adjustment.completed`

## 🔧 Configuração

A configuração é gerenciada através de arquivos YAML no diretório `config/`:

- `config/config.yaml` - Configurações principais
- `config/infrastructure/` - Banco de dados e mensageria
- `config/environments/` - Configurações específicas por ambiente

### Variáveis de Ambiente

```bash
# Desenvolvimento
export FULFILLMENT_ENV=dev

# Produção
export FULFILLMENT_ENV=prod

# Override de caminho de configuração
export FULFILLMENT_CONFIG_PATH=/path/to/config
```

## 📊 Monitoramento & Observabilidade

Monitoramento completo integrado:

- **Métricas**: Prometheus compatível (`/metrics`)
- **Tracing**: OpenTelemetry/Jaeger
- **Logging**: Logs estruturados com trace_id
- **Health Checks**: Endpoints de liveness e readiness

### Dashboard

Acesse o dashboard de monitoramento em `http://localhost:3000` (Grafana)

## 🚀 Deploy

### Docker

```bash
# Construir imagem
make docker

# Executar container
docker run -p 8080:8080 -e FULFILLMENT_ENV=prod mcp-fulfillment-ops:latest
```

### Docker Compose

```bash
# Subir stack completa
docker-compose up -d

# Verificar status
docker-compose ps
```

### Kubernetes

```bash
# Deploy para Kubernetes
kubectl apply -f deployments/k8s/

# Verificar status do deployment
kubectl get pods -l app=mcp-fulfillment-ops
```

## 🧪 Testes

```bash
# Executar todos os testes
make test

# Executar com cobertura
make test-coverage

# Executar testes de integração
go test -v ./tests/integration/...

# Executar testes de carga
k6 run tests/load/fulfillment-flow.js
```

## 📚 Documentação

- [Guia de Arquitetura](docs/architecture/)
- [Documentação da API](docs/api/)
- [Guias de Uso](docs/guides/)
- [Exemplos](docs/examples/)
- [Troubleshooting](docs/guides/troubleshooting.md)

## 🔗 Integrações

### Core Inventory

O serviço se integra com o `mcp-core-inventory` para:

- Reservar itens no momento da criação da ordem
- Confirmar baixa de estoque na expedição
- Sincronizar ajustes de inventário

### OMS (Order Management System)

Recebe eventos do OMS para iniciar o processo de fulfillment:

```json
{
  "subject": "oms.order.ready_to_pick",
  "data": {
    "order_id": "ORD-12345",
    "customer_id": "CUST-001",
    "items": [
      {"sku": "PROD-001", "quantity": 2}
    ]
  }
}
```

## 🤝 Contribuindo

1. Fork do repositório
2. Criar branch de feature (`git checkout -b feature/amazing-feature`)
3. Commit das mudanças (`git commit -m 'Add amazing feature'`)
4. Push para o branch (`git push origin feature/amazing-feature`)
5. Abrir Pull Request

## 📄 Licença

Este projeto está licenciado sob MIT License - veja o arquivo [LICENSE](LICENSE) para detalhes.

## 🆘 Suporte

- **Issues**: [GitHub Issues](https://github.com/vertikon/mcp-fulfillment-ops/issues)
- **Discussões**: [GitHub Discussions](https://github.com/vertikon/mcp-fulfillment-ops/discussions)
- **Documentação**: [Project Docs](https://docs.vertikon.com/mcp-fulfillment-ops)

---

**Construído com ❤️ pelo Vertikon Team**