# BLOCO-1 - Documentação Técnica: MCP Fulfillment Ops

## 📋 Visão Geral

O **MCP Fulfillment Ops** é o serviço orquestrador de operações logísticas do ecossistema Vertikon, responsável por gerenciar o fluxo físico de produtos desde o recebimento até a expedição. Atua como o **Braço Operacional** do BLOCO-1, complementando o **Cérebro Financeiro** (`mcp-core-inventory`).

### 🎯 Responsabilidades Principais

- **Gestão de Ordens de Fulfillment**: Criação e acompanhamento de ordens de expedição
- **Processamento de Inbound**: Recebimento e armazenamento de produtos
- **Picking e Packing**: Separação e embalagem de pedidos
- **Shipping**: Expedição e rastreio de entregas
- **Gestão de Devoluções**: Processamento de retornos e reposições
- **Controle de Estoque Físico**: Sincronização com o Core Inventory
- **Integração por Eventos**: Comunicação assíncrona via NATS JetStream

## 🏗️ Arquitetura do Serviço

### Domínio de Fulfillment

```
┌─────────────────────────────────────────────────────────────┐
│                 MCP Fulfillment Ops                 │
│                                                   │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │
│  │   Inbound   │  │  Outbound   │  │ Returns  │ │
│  │ (Recebimento)│  │(Expedição) │  │(Devolução)│ │
│  └─────────────┘  └─────────────┘  └──────────┘ │
│                                                   │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │
│  │ Transfers  │  │ Cycle Count │  │   Picking│ │
│  │(Transferência)│  │(Contagem)  │  │(Separação)│ │
│  └─────────────┘  └─────────────┘  └──────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Fluxo de Processamento

```mermaid
sequenceDiagram
    participant OMS as Order Management
    participant FO as Fulfillment Ops
    participant CI as Core Inventory
    participant WMS as Sistema WMS

    OMS->>FO: oms.order.ready_to_pick
    FO->>CI: POST /api/v1/reservations
    CI-->>FO: reservation.confirmed
    FO->>FO: Criar FulfillmentOrder
    FO->>WMS: Iniciar Picking
    WMS-->>FO: Picking Concluído
    FO->>FO: Ship Order
    FO->>CI: POST /api/v1/reservations/confirm
    CI-->>FO: inventory.updated
    FO->>OMS: fulfillment.order.completed
```

## 📊 Entidades de Domínio

### 1. FulfillmentOrder (Expedição)

```go
type FulfillmentOrder struct {
    ID             string     `json:"id"`
    OrderID        string     `json:"order_id"`        // ID do Pedido OMS
    Customer       string     `json:"customer"`
    Destination    string     `json:"destination"`     // Endereço de entrega
    Status         Status     `json:"status"`          // PENDING | IN_PROGRESS | COMPLETED
    Items          []Item     `json:"items"`           // Items do pedido
    Priority       int        `json:"priority"`        // 0-Normal, 1-Express
    IdempotencyKey string     `json:"idempotency_key"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
    ShippedAt      *time.Time `json:"shipped_at,omitempty"`
}
```

### 2. InboundShipment (Recebimento)

```go
type InboundShipment struct {
    ID             string     `json:"id"`
    ReferenceID    string     `json:"reference_id"`    // Número da NF/PO
    Origin         string     `json:"origin"`          // Fornecedor
    Destination    string     `json:"destination"`     // Armazém
    Status         Status     `json:"status"`          // PENDING | IN_PROGRESS | COMPLETED
    Items          []Item     `json:"items"`           // Items recebidos
    IdempotencyKey string     `json:"idempotency_key"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
    CompletedAt    *time.Time `json:"completed_at,omitempty"`
}
```

### 3. TransferOrder (Transferência)

```go
type TransferOrder struct {
    ID             string     `json:"id"`
    LocationFrom   string     `json:"location_from"`    // Armazém origem
    LocationTo     string     `json:"location_to"`      // Armazém destino
    Status         Status     `json:"status"`          // PENDING | IN_PROGRESS | COMPLETED
    Items          []Item     `json:"items"`           // Items transferidos
    IdempotencyKey string     `json:"idempotency_key"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
    CompletedAt    *time.Time `json:"completed_at,omitempty"`
}
```

### 4. ReturnOrder (Devolução)

```go
type ReturnOrder struct {
    ID             string     `json:"id"`
    OriginalOrderID string     `json:"original_order_id"` // Pedido original
    Reason         string     `json:"reason"`           // Motivo da devolução
    Status         Status     `json:"status"`          // PENDING | IN_PROGRESS | COMPLETED
    Items          []Item     `json:"items"`           // Items devolvidos
    IdempotencyKey string     `json:"idempotency_key"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
    CompletedAt    *time.Time `json:"completed_at,omitempty"`
}
```

### 5. CycleCountTask (Contagem Cíclica)

```go
type CycleCountTask struct {
    ID             string     `json:"id"`
    Location       string     `json:"location"`         // Área de contagem
    SKUs           []string   `json:"skus"`             // SKUs a contar
    Status         Status     `json:"status"`          // PENDING | IN_PROGRESS | COMPLETED
    CountedItems   []CountedItem `json:"counted_items"`  // Items contados
    IdempotencyKey string     `json:"idempotency_key"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
    CompletedAt    *time.Time `json:"completed_at,omitempty"`
}
```

## 🔄 Máquina de Estados

### Estados das Ordens

```mermaid
stateDiagram-v2
    [*] --> PENDING
    PENDING --> IN_PROGRESS: StartPicking()
    IN_PROGRESS --> COMPLETED: Ship()
    IN_PROGRESS --> CANCELLED: Cancel()
    PENDING --> CANCELLED: Cancel()
    COMPLETED --> [*]
    CANCELLED --> [*]
```

### Validações de Transição

- **PENDING → IN_PROGRESS**: Apenas se todos os itens estiverem reservados no Core Inventory
- **IN_PROGRESS → COMPLETED**: Apenas se picking estiver concluído no WMS
- **IN_PROGRESS/PENDING → CANCELLED**: A qualquer momento, com liberação automática de reservas
- **COMPLETED → ***: Estado terminal, não permite mais transições

## 🔌 Integrações

### 1. Core Inventory (Ledger ACID)

**Reserva de Estoque:**
```http
POST /api/v1/reservations
Content-Type: application/json

{
  "sku": "PROD-001",
  "location": "WH-01",
  "quantity": 10,
  "idempotency_key": "fulfillment-order-123"
}
```

**Confirmação de Baixa:**
```http
POST /api/v1/reservations/{id}/confirm
Content-Type: application/json

{
  "quantity": 10,
  "reference_id": "fulfillment-order-123"
}
```

### 2. Eventos NATS JetStream

**Eventos Publicados:**
- `fulfillment.order.created`
- `fulfillment.order.picked`
- `fulfillment.order.shipped`
- `fulfillment.inventory.updated`
- `fulfillment.inbound.completed`
- `fulfillment.transfer.completed`

**Eventos Consumidos:**
- `oms.order.ready_to_pick`
- `inventory.reservation.confirmed`
- `inventory.adjustment.completed`
- `wms.picking.completed`

### 3. WMS (Warehouse Management System)

**Integração via API REST:**
```http
POST /api/v1/wms/picking-jobs
Content-Type: application/json

{
  "fulfillment_order_id": "FO-123",
  "items": [
    {"sku": "PROD-001", "quantity": 2, "location": "A-01-03"}
  ],
  "priority": "normal"
}
```

## 📋 APIs Principais

### 1. Gestão de Ordens de Fulfillment

```http
# Criar ordem
POST /api/v1/fulfillment-orders
{
  "order_id": "ORD-12345",
  "customer": "CUSTOMER-001",
  "destination": "Rua A, 123 - São Paulo/SP",
  "items": [{"sku": "PROD-001", "quantity": 2}],
  "priority": 0
}

# Iniciar picking
POST /api/v1/fulfillment-orders/{order_id}/pick

# Confirmar expedição
POST /api/v1/fulfillment-orders/{order_id}/ship

# Consultar ordem
GET /api/v1/fulfillment-orders/{order_id}
```

### 2. Gestão de Inbound

```http
# Criar recebimento
POST /api/v1/inbound-shipments
{
  "reference_id": "NF-12345",
  "origin": "FORNECEDOR-001",
  "destination": "WH-01",
  "items": [{"sku": "PROD-001", "quantity": 100}]
}

# Confirmar recebimento
POST /api/v1/inbound-shipments/{id}/complete
```

### 3. Gestão de Transferências

```http
# Criar transferência
POST /api/v1/transfer-orders
{
  "location_from": "WH-01",
  "location_to": "WH-02",
  "items": [{"sku": "PROD-001", "quantity": 50}]
}
```

## 🗄️ Schema de Banco de Dados

### Tabelas Principais

```sql
-- Ordens de Expedição
CREATE TABLE fulfillment_orders (
    id VARCHAR(255) PRIMARY KEY,
    order_id VARCHAR(255) NOT NULL,
    customer VARCHAR(255) NOT NULL,
    destination VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    items JSONB NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    idempotency_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    shipped_at TIMESTAMP
);

-- Recebimentos
CREATE TABLE inbound_shipments (
    id VARCHAR(255) PRIMARY KEY,
    reference_id VARCHAR(255) NOT NULL,
    origin VARCHAR(255) NOT NULL,
    destination VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    items JSONB NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP
);

-- Transferências
CREATE TABLE transfer_orders (
    id VARCHAR(255) PRIMARY KEY,
    location_from VARCHAR(255) NOT NULL,
    location_to VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    items JSONB NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP
);

-- Devoluções
CREATE TABLE return_orders (
    id VARCHAR(255) PRIMARY KEY,
    original_order_id VARCHAR(255) NOT NULL,
    reason VARCHAR(500),
    status VARCHAR(50) NOT NULL,
    items JSONB NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP
);

-- Contagens Cíclicas
CREATE TABLE cycle_count_tasks (
    id VARCHAR(255) PRIMARY KEY,
    location VARCHAR(255) NOT NULL,
    skus JSONB NOT NULL,
    status VARCHAR(50) NOT NULL,
    counted_items JSONB NOT NULL DEFAULT '[]',
    idempotency_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP
);
```

## 📊 Métricas e Monitoramento

### KPIs de Negócio

- **Volume de Ordens**: Total de ordens criadas por período
- **Tempo de Ciclo**: Tempo médio desde criação até expedição
- **Taxa de Picking**: Eficiência do processo de separação
- **Precisão de Inventário**: Diferença entre estoque físico e sistema
- **Taxa de Devolução**: Percentual de pedidos devolvidos

### Métricas Técnicas

- **Latência das APIs**: P50, P95, P99
- **Taxa de Erro**: Por endpoint e tipo de erro
- **Throughput**: Ordens processadas por segundo
- **Integração NATS**: Eventos publicados/consumidos
- **Conexão com Core Inventory**: Taxa de sucesso/falha

## 🔧 Configuração

### Variáveis de Ambiente

```bash
# Configurações Principais
ENVIRONMENT=development
HTTP_PORT=:8080

# Banco de Dados
DATABASE_URL=postgresql://user:pass@localhost:5432/fulfillment

# Mensageria
NATS_URL=nats://localhost:4222

# Cache
REDIS_URL=redis://localhost:6379

# Integração Core Inventory
CORE_INVENTORY_URL=http://localhost:8081
CORE_INVENTORY_TIMEOUT=30s
CORE_INVENTORY_RETRIES=3

# Configurações de Negócio
FULFILLMENT_DEFAULT_PRIORITY=0
FULFILLMENT_RESERVATION_TIMEOUT=300s
FULFILLMENT_AUTO_CONFIRM_SHIPPING=true
```

## 🚀 Deploy e Operação

### Docker Compose

```bash
# Subir serviço individual
docker-compose up -d

# Subir com dependências
docker-compose -f docker-compose-integration.yml up -d

# Verificar logs
docker-compose logs -f fulfillment-ops
```

### Health Checks

```http
GET /health
{
  "status": "healthy",
  "timestamp": "2025-11-22T12:00:00Z",
  "version": "1.0.0",
  "dependencies": {
    "database": "healthy",
    "nats": "healthy",
    "redis": "healthy",
    "core_inventory": "healthy"
  }
}
```

## 🧪 Testes

### Testes de Integração

```bash
# Executar suite completa
go test ./tests/integration/...

# Teste específico de API
go test ./tests/api/fulfillment_orders_test.go

# Teste de integração com Core Inventory
go test ./tests/integration/core_inventory_test.go
```

### Testes de Carga

```bash
# Simular pico de criação de ordens
k6 run tests/load/create_orders.js

# Testar processamento de eventos
k6 run tests/load/nats_events.js
```

## 🔒 Segurança

### Autenticação e Autorização

- **JWT Tokens**: Validação de tokens em todas as APIs
- **RBAC**: Controle de acesso baseado em papéis
- **Rate Limiting**: Limitação por cliente/IP
- **CORS**: Configuração restritiva de origens

### Validações

- **Idempotência**: Todas as operações críticas usam idempotency_key
- **Validação de Entrada**: Schema validation para todos os payloads
- **Sanitização**: Remoção de dados maliciosos
- **SQL Injection Protection**: Uso exclusivo de prepared statements

## 📚 Referências

- [BLOCO-1 Blueprint](../BLOCO-1-BLUEPRINT.md)
- [Core Inventory Documentation](../mcp-core-inventory/docs/)
- [API Reference](./docs/api/)
- [NATS Subjects](./docs/NATS_SUBJECTS.md)
- [Deployment Guide](./README-DEPLOY.md)

---

**Status**: ✅ **Implementação Completa e Produção-Ready**