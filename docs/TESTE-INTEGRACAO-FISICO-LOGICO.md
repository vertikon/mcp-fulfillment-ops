# 🧪 Teste de Integração Físico-Lógico

## 📋 Objetivo

Validar que o fluxo **ShipOrder no Fulfillment** gera corretamente o **débito no Core Inventory** via comunicação NATS/HTTP.

## 🎯 Teste Crítico

**Fluxo a Validar:**
```
OMS → FulfillmentOrder → ShipOrder → Core Inventory (débito)
```

## 🚀 Executar Teste

### Pré-requisitos

1. **mcp-core-inventory** rodando em `http://localhost:8081`
2. **mcp-fulfillment-ops** rodando em `http://localhost:8080`
3. **NATS** rodando (se usando eventos assíncronos)
4. **PostgreSQL** rodando (para ambos os serviços)

### Opção 1: Script Automatizado

**Windows (PowerShell):**
```powershell
$env:FULFILLMENT_URL = "http://localhost:8080"
$env:CORE_INVENTORY_URL = "http://localhost:8081"
.\scripts\test-integration-physical-logical.ps1
```

**Linux/Mac (Bash):**
```bash
export FULFILLMENT_URL=http://localhost:8080
export CORE_INVENTORY_URL=http://localhost:8081
chmod +x scripts/test-integration-physical-logical.sh
./scripts/test-integration-physical-logical.sh
```

### Opção 2: Teste Go E2E

```bash
go test -v ./tests/e2e/... -run TestIntegrationFulfillmentToCoreInventory
```

### Opção 3: Teste Manual (curl)

#### Step 1: Criar produto no Core Inventory

```bash
curl -X POST http://localhost:8081/v1/adjust \
  -H "Content-Type: application/json" \
  -d '{
    "location": "CD-TEST",
    "sku": "SKU-TEST-001",
    "quantity": 100,
    "reason": "test_setup"
  }'
```

#### Step 2: Verificar estoque inicial

```bash
curl http://localhost:8081/v1/available?location=CD-TEST&sku=SKU-TEST-001
```

**Resposta esperada:**
```json
{
  "available": 100
}
```

#### Step 3: Criar FulfillmentOrder

```bash
curl -X POST http://localhost:8080/v1/outbound/start_picking \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "TEST-ORDER-001"
  }'
```

#### Step 4: Executar ShipOrder (TESTE CRÍTICO)

```bash
curl -X POST http://localhost:8080/v1/outbound/ship \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "TEST-ORDER-001"
  }'
```

**Resposta esperada:**
```json
{
  "status": "shipped"
}
```

#### Step 5: Validar débito no Core Inventory

```bash
curl http://localhost:8081/v1/available?location=CD-TEST&sku=SKU-TEST-001
```

**Resposta esperada:**
```json
{
  "available": 90  // Deve ter diminuído (assumindo quantidade de 10)
}
```

## ✅ Critérios de Sucesso

1. ✅ **ShipOrder executado com sucesso** (HTTP 200)
2. ✅ **Estoque no Core Inventory foi debitado** (diminuiu)
3. ✅ **Evento publicado no NATS** (se usando eventos)
4. ✅ **Logs mostram comunicação entre serviços**

## 🔍 Validações Adicionais

### Verificar Eventos NATS

```bash
# Se usando NATS JetStream
nats stream ls
nats consumer ls FULFILLMENT_EVENTS
```

### Verificar Logs

```bash
# Logs do Fulfillment
docker-compose logs fulfillment-ops | grep -i "ship\|inventory"

# Logs do Core Inventory
docker-compose logs core-inventory | grep -i "adjust\|reserve"
```

## 📊 Resultado Esperado

### Antes do ShipOrder
- **Estoque Core Inventory:** 100 unidades
- **FulfillmentOrder Status:** IN_PROGRESS

### Depois do ShipOrder
- **Estoque Core Inventory:** 90 unidades (debitado 10)
- **FulfillmentOrder Status:** COMPLETED
- **Evento publicado:** `fulfillment.outbound.shipped.v1`

## ⚠️ Troubleshooting

### Erro: "Core Inventory não está respondendo"

- Verifique se `mcp-core-inventory` está rodando
- Confirme a URL: `http://localhost:8081`
- Verifique logs: `docker-compose logs core-inventory`

### Erro: "ShipOrder falhou"

- Verifique se a ordem existe
- Confirme que está em status `IN_PROGRESS`
- Verifique logs do Fulfillment

### Erro: "Estoque não foi debitado"

- Verifique comunicação entre serviços
- Confirme que NATS está funcionando (se usando eventos)
- Verifique logs de ambos os serviços
- Valide que o Core Inventory recebeu o comando

## 📈 Métricas de Sucesso

- ✅ **Latência:** ShipOrder < 500ms
- ✅ **Confiabilidade:** 100% de sucesso em débito
- ✅ **Idempotência:** Múltiplos calls não duplicam débito
- ✅ **Rastreabilidade:** Trace ID presente em logs

## 🔗 Referências

- [Blueprint do MCP Fulfillment Ops](../../../../.cursor/BLOCOS/BLOCO-1-BLUEPRINT-MCP-FULFILLMENT-OPS.md)
- [Auditoria de Conformidade](../../../../.cursor/AUDITORIA/BLOCO-1-AUDITORIA-CONFORMIDADE-BLOCO-1-BLUEPRINT-MCP-FULFILLMENT-OPS.md)

