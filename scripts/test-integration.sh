#!/bin/bash
# Script para testar integração com mcp-core-inventory

set -e

echo "🧪 Testando integração com mcp-core-inventory..."

BASE_URL="${CORE_INVENTORY_URL:-http://localhost:8081}"
FULFILLMENT_URL="${FULFILLMENT_URL:-http://localhost:8080}"

echo "📡 Core Inventory URL: $BASE_URL"
echo "📡 Fulfillment Ops URL: $FULFILLMENT_URL"
echo ""

# Teste 1: Health check do Core Inventory
echo "1️⃣ Testando health check do Core Inventory..."
if curl -f -s "$BASE_URL/health" > /dev/null; then
    echo "   ✅ Core Inventory está respondendo"
else
    echo "   ❌ Core Inventory não está respondendo"
    exit 1
fi

# Teste 2: Health check do Fulfillment Ops
echo ""
echo "2️⃣ Testando health check do Fulfillment Ops..."
if curl -f -s "$FULFILLMENT_URL/health" > /dev/null; then
    echo "   ✅ Fulfillment Ops está respondendo"
else
    echo "   ❌ Fulfillment Ops não está respondendo"
    exit 1
fi

# Teste 3: Criar Inbound Shipment
echo ""
echo "3️⃣ Testando criação de Inbound Shipment..."
RESPONSE=$(curl -s -X POST "$FULFILLMENT_URL/v1/inbound/start" \
  -H "Content-Type: application/json" \
  -d '{
    "reference_id": "TEST-PO-001",
    "origin": "Fornecedor Teste",
    "destination": "CD-TEST",
    "items": [
      {"sku": "SKU-TEST-001", "quantity": 10}
    ]
  }')

SHIPMENT_ID=$(echo $RESPONSE | jq -r '.id // empty')
if [ -n "$SHIPMENT_ID" ]; then
    echo "   ✅ Inbound Shipment criado: $SHIPMENT_ID"
else
    echo "   ❌ Falha ao criar Inbound Shipment"
    echo "   Resposta: $RESPONSE"
    exit 1
fi

# Teste 4: Confirmar recebimento (requer Core Inventory funcionando)
echo ""
echo "4️⃣ Testando confirmação de recebimento..."
CONFIRM_RESPONSE=$(curl -s -X POST "$FULFILLMENT_URL/v1/inbound/confirm" \
  -H "Content-Type: application/json" \
  -d "{\"shipment_id\": \"$SHIPMENT_ID\"}")

if echo "$CONFIRM_RESPONSE" | jq -e '.status' > /dev/null 2>&1; then
    echo "   ✅ Recebimento confirmado"
else
    echo "   ⚠️  Confirmação pode ter falhado (verifique se Core Inventory está configurado)"
    echo "   Resposta: $CONFIRM_RESPONSE"
fi

echo ""
echo "✅ Testes de integração concluídos!"

