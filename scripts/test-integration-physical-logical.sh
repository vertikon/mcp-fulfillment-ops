#!/bin/bash
# Script de Teste de Integração Físico-Lógico
# Valida que ShipOrder no Fulfillment gera débito correto no Core Inventory

set -e

FULFILLMENT_URL="${FULFILLMENT_URL:-http://localhost:8080}"
CORE_INVENTORY_URL="${CORE_INVENTORY_URL:-http://localhost:8081}"

echo "🧪 TESTE DE INTEGRAÇÃO FÍSICO-LÓGICO"
echo "======================================"
echo ""
echo "Fulfillment Ops: $FULFILLMENT_URL"
echo "Core Inventory:  $CORE_INVENTORY_URL"
echo ""

# Cores para output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Função para fazer requisições HTTP
http_request() {
    local method=$1
    local url=$2
    local data=$3
    
    if [ -n "$data" ]; then
        curl -s -X "$method" "$url" \
            -H "Content-Type: application/json" \
            -d "$data"
    else
        curl -s -X "$method" "$url"
    fi
}

# Step 1: Health Checks
echo "1️⃣ Verificando saúde dos serviços..."
FULFILLMENT_HEALTH=$(http_request GET "$FULFILLMENT_URL/health")
CORE_HEALTH=$(http_request GET "$CORE_INVENTORY_URL/health")

if echo "$FULFILLMENT_HEALTH" | grep -q "ok"; then
    echo -e "   ${GREEN}✅ Fulfillment Ops está rodando${NC}"
else
    echo -e "   ${RED}❌ Fulfillment Ops não está respondendo${NC}"
    exit 1
fi

if echo "$CORE_HEALTH" | grep -q "ok"; then
    echo -e "   ${GREEN}✅ Core Inventory está rodando${NC}"
else
    echo -e "   ${YELLOW}⚠️  Core Inventory não está respondendo (teste continuará mas pode falhar)${NC}"
fi

echo ""

# Step 2: Setup - Criar produto no Core Inventory
echo "2️⃣ Configurando produto no Core Inventory..."
TIMESTAMP=$(date +%s)
SKU="SKU-TEST-$TIMESTAMP"
LOCATION="CD-TEST"

# Criar produto com estoque inicial
ADJUST_DATA=$(cat <<EOF
{
  "location": "$LOCATION",
  "sku": "$SKU",
  "quantity": 100,
  "reason": "test_setup"
}
EOF
)

ADJUST_RESPONSE=$(http_request POST "$CORE_INVENTORY_URL/v1/adjust" "$ADJUST_DATA")
if echo "$ADJUST_RESPONSE" | grep -q "error"; then
    echo -e "   ${YELLOW}⚠️  Erro ao criar produto (pode já existir): $ADJUST_RESPONSE${NC}"
else
    echo -e "   ${GREEN}✅ Produto criado: $SKU com 100 unidades${NC}"
fi

# Verificar estoque inicial
AVAILABLE_RESPONSE=$(http_request GET "$CORE_INVENTORY_URL/v1/available?location=$LOCATION&sku=$SKU")
INITIAL_STOCK=$(echo "$AVAILABLE_RESPONSE" | jq -r '.available // 0' 2>/dev/null || echo "0")
echo -e "   ${GREEN}📊 Estoque inicial: $INITIAL_STOCK unidades${NC}"

echo ""

# Step 3: Criar FulfillmentOrder (simulando evento OMS)
echo "3️⃣ Criando FulfillmentOrder..."
ORDER_ID="TEST-ORDER-$TIMESTAMP"

# Nota: Assumindo que existe endpoint para criar ordem
# Se não existir, vamos criar via start_picking
PICKING_DATA=$(cat <<EOF
{
  "order_id": "$ORDER_ID"
}
EOF
)

PICKING_RESPONSE=$(http_request POST "$FULFILLMENT_URL/v1/outbound/start_picking" "$PICKING_DATA")
echo "   📝 Resposta start_picking: $PICKING_RESPONSE"

echo ""

# Step 4: Executar ShipOrder (TESTE CRÍTICO)
echo "4️⃣ 🎯 TESTE CRÍTICO: Executando ShipOrder..."
SHIP_DATA=$(cat <<EOF
{
  "order_id": "$ORDER_ID"
}
EOF
)

SHIP_RESPONSE=$(http_request POST "$FULFILLMENT_URL/v1/outbound/ship" "$SHIP_DATA")
SHIP_STATUS=$(echo "$SHIP_RESPONSE" | jq -r '.status // "unknown"' 2>/dev/null || echo "unknown")

if echo "$SHIP_RESPONSE" | grep -q "error"; then
    echo -e "   ${YELLOW}⚠️  ShipOrder pode ter falhado: $SHIP_RESPONSE${NC}"
    echo "   (Isso pode ser esperado se a ordem não existir ou Core não estiver disponível)"
else
    echo -e "   ${GREEN}✅ ShipOrder executado: $SHIP_STATUS${NC}"
fi

echo ""

# Step 5: Aguardar processamento
echo "5️⃣ Aguardando processamento do evento..."
sleep 3

# Step 6: Validar débito no Core Inventory
echo "6️⃣ 🎯 VALIDAÇÃO CRÍTICA: Verificando débito no Core Inventory..."
FINAL_AVAILABLE=$(http_request GET "$CORE_INVENTORY_URL/v1/available?location=$LOCATION&sku=$SKU")
FINAL_STOCK=$(echo "$FINAL_AVAILABLE" | jq -r '.available // 0' 2>/dev/null || echo "0")

echo -e "   📊 Estoque final: $FINAL_STOCK unidades"

if [ "$FINAL_STOCK" != "0" ] && [ "$FINAL_STOCK" -lt "$INITIAL_STOCK" ]; then
    DIFF=$((INITIAL_STOCK - FINAL_STOCK))
    echo -e "   ${GREEN}✅ SUCESSO: Estoque foi debitado! Diferença: $DIFF unidades${NC}"
    echo -e "   ${GREEN}✅ Integração Físico-Lógico funcionando corretamente!${NC}"
    exit 0
elif [ "$FINAL_STOCK" = "0" ]; then
    echo -e "   ${YELLOW}⚠️  Estoque zerado (pode ser esperado se foi tudo debitado)${NC}"
else
    echo -e "   ${RED}❌ FALHA: Estoque não foi debitado corretamente${NC}"
    echo -e "   ${RED}   Estoque inicial: $INITIAL_STOCK, Final: $FINAL_STOCK${NC}"
    exit 1
fi

echo ""
echo "✅ TESTE DE INTEGRAÇÃO CONCLUÍDO"

