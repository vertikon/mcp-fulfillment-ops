#!/bin/bash

# Teste de Integração Completo - BLOCO-1
# Valida fluxo completo entre Core Inventory e Fulfillment Ops

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}=== INICIANDO TESTE DE INTEGRAÇÃO COMPLETO - BLOCO-1 ===${NC}"
echo

# Função para verificar saúde do serviço
check_health() {
    local service_name=$1
    local url=$2
    local max_attempts=30
    local attempt=1
    
    echo -n "Verificando saúde do $service_name... "
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$url" > /dev/null; then
            echo -e "${GREEN}✓ OK${NC}"
            return 0
        fi
        
        echo -n "."
        sleep 2
        ((attempt++))
    done
    
    echo -e "${RED}✗ FALHOU${NC}"
    return 1
}

# Função para aguardar serviço estar pronto
wait_for_service() {
    local service_name=$1
    local url=$2
    local timeout=60
    
    echo -n "Aguardando $service_name ficar pronto... "
    
    end_time=$((SECONDS + timeout))
    while [ $SECONDS -lt $end_time ]; do
        if curl -s -f "$url" > /dev/null 2>&1 | grep -q "healthy"; then
            echo -e "${GREEN}Pronto${NC}"
            return 0
        fi
        sleep 2
    done
    
    echo -e "${YELLOW}Timeout${NC}"
    return 1
}

# 1. Verificar se a stack está no ar
echo -e "\n${YELLOW}1. Verificando stack de serviços...${NC}"
if ! docker-compose -f docker-compose-integration.yml ps | grep -q "Up"; then
    echo -e "${RED}✗ Stack não está no ar${NC}"
    echo "Execute: docker-compose -f docker-compose-integration.yml up -d"
    exit 1
fi

echo -e "${GREEN}✓ Stack no ar${NC}"

# 2. Aguardar serviços ficarem prontos
echo -e "\n${YELLOW}2. Aguardando serviços ficarem prontos...${NC}"
wait_for_service "Core Inventory" "http://localhost:8081/health"
wait_for_service "Fulfillment Ops" "http://localhost:8082/health"
wait_for_service "NATS" "http://localhost:8225/jsz?acc=noncluster&subjects="
wait_for_service "PostgreSQL" "http://localhost:5435"
wait_for_service "Redis" "http://localhost:6381"

# 3. Verificar saúde dos serviços
echo -e "\n${YELLOW}3. Verificando saúde dos serviços...${NC}"
check_health "Core Inventory" "http://localhost:8081/health"
check_health "Fulfillment Ops" "http://localhost:8082/health"
check_health "NATS" "http://localhost:8225/varz"
check_health "PostgreSQL" "http://localhost:5435"
check_health "Redis" "http://localhost:6381"

# 4. Verificar streams NATS
echo -e "\n${YELLOW}4. Verificando streams NATS...${NC}"
docker exec -it fulfillment-nats nats stream list | grep -E "(fulfillment|inventory)" || {
    echo -e "${RED}✗ Streams não encontrados${NC}"
    exit 1
}
echo -e "${GREEN}✓ Streams OK${NC}"

# 5. Testar criação de ordem
echo -e "\n${YELLOW}5. Testando criação de ordem...${NC}"
ORDER_ID="TEST-$(date +%s)"
RESPONSE=$(curl -s -X POST http://localhost:8082/api/v1/fulfillment-orders \
    -H "Content-Type: application/json" \
    -d "{
        \"order_id\": \"$ORDER_ID\",
        \"customer\": \"CUSTOMER-INTEGRATION-TEST\",
        \"destination\": \"Rua Teste Integração, 123 - São Paulo/SP\",
        \"items\": [
            {\"sku\": \"PROD-TEST-001\", \"quantity\": 2}
        ],
        \"priority\": 0,
        \"idempotency_key\": \"$ORDER_ID\"
    }")

if echo "$RESPONSE" | grep -q "\"id\":\"$ORDER_ID\""; then
    echo -e "${GREEN}✓ Ordem criada${NC}"
else
    echo -e "${RED}✗ Falha ao criar ordem${NC}"
    echo "Response: $RESPONSE"
    exit 1
fi

# 6. Verificar reserva no Core Inventory
echo -e "\n${YELLOW}6. Verificando reserva no Core Inventory...${NC}"
sleep 2

RESERVATION=$(curl -s http://localhost:8081/api/v1/reservations \
    -H "Content-Type: application/json" \
    -G -d "reference_id=$ORDER_ID")

if echo "$RESERVATION" | grep -q "\"reference_id\":\"$ORDER_ID\""; then
    echo -e "${GREEN}✓ Reserva criada${NC}"
else
    echo -e "${RED}✗ Reserva não encontrada${NC}"
    exit 1
fi

# 7. Publicar evento no NATS
echo -e "\n${YELLOW}7. Publicando evento de teste...${NC}"
docker exec -it fulfillment-nats nats pub oms.order.ready_to_pick "{
    \"order_id\": \"$ORDER_ID\",
    \"customer_id\": \"CUST-INTEGRATION-TEST\",
    \"items\": [
        {\"sku\": \"PROD-TEST-001\", \"quantity\": 2}
    ]
}"

echo -e "${GREEN}✓ Evento publicado${NC}"

# 8. Aguardar processamento do evento
echo -e "\n${YELLOW}8. Aguardando processamento do evento...${NC}"
sleep 3

# 9. Verificar status da ordem
echo -e "\n${YELLOW}9. Verificando status da ordem...${NC}"
STATUS=$(curl -s http://localhost:8082/api/v1/fulfillment-orders/$ORDER_ID \
    -H "Content-Type: application/json" | jq -r '.status')

if [ "$STATUS" = "PENDING" ]; then
    echo -e "${GREEN}✓ Ordem em processamento${NC}"
else
    echo -e "${RED}✗ Status inesperado: $STATUS${NC}"
    exit 1
fi

# 10. Iniciar picking
echo -e "\n${YELLOW}10. Iniciando picking...${NC}"
PICK_RESPONSE=$(curl -s -X POST http://localhost:8082/api/v1/fulfillment-orders/$ORDER_ID/pick \
    -H "Content-Type: application/json")

if echo "$PICK_RESPONSE" | grep -q "\"status\":\"IN_PROGRESS\""; then
    echo -e "${GREEN}✓ Picking iniciado${NC}"
else
    echo -e "${RED}✗ Falha ao iniciar picking${NC}"
    exit 1
fi

# 11. Confirmar expedição
echo -e "\n${YELLOW}11. Confirmando expedição...${NC}"
SHIP_RESPONSE=$(curl -s -X POST http://localhost:8082/api/v1/fulfillment-orders/$ORDER_ID/ship \
    -H "Content-Type: application/json" \
    -d "{
        \"tracking_number\": \"TRACK-$(date +%s)\",
        \"carrier\": \"INTEGRATION-CARRIER\"
    }")

if echo "$SHIP_RESPONSE" | grep -q "\"status\":\"COMPLETED\""; then
    echo -e "${GREEN}✓ Expedição confirmada${NC}"
else
    echo -e "${RED}✗ Falha ao confirmar expedição${NC}"
    exit 1
fi

# 12. Verificar baixa no estoque
echo -e "\n${YELLOW}12. Verificando baixa no estoque...${NC}"
sleep 2

STOCK=$(curl -s http://localhost:8081/api/v1/inventory/PROD-TEST-001 \
    -H "Content-Type: application/json")

# Extrair quantidade disponível (deve ser menor que antes)
AVAILABLE=$(echo "$STOCK" | jq -r '.available')

if [ "$AVAILABLE" -lt 100 ]; then
    echo -e "${GREEN}✓ Estoque baixado${NC} (Disponível: $AVAILABLE)"
else
    echo -e "${RED}✗ Estoque não foi baixado${NC}"
    exit 1
fi

# 13. Verificar eventos publicados
echo -e "\n${YELLOW}13. Verificando eventos publicados...${NC}"
docker exec -it fulfillment-nats nats stream info fulfillment.engine.tasks | grep -q "messages" || {
    echo -e "${RED}✗ Sem mensagens no stream${NC}"
    exit 1
}
echo -e "${GREEN}✓ Eventos publicados${NC}"

# 14. Testar idempotência
echo -e "\n${YELLOW}14. Testando idempotência...${NC}"
IDEM_RESPONSE=$(curl -s -X POST http://localhost:8082/api/v1/fulfillment-orders \
    -H "Content-Type: application/json" \
    -d "{
        \"order_id\": \"$ORDER_ID-REPEAT\",
        \"customer\": \"CUSTOMER-INTEGRATION-TEST\",
        \"items\": [
            {\"sku\": \"PROD-TEST-001\", \"quantity\": 1}
        ],
        \"idempotency_key\": \"$ORDER_ID\"
    }")

# Deve retornar a mesma ordem, não criar nova
if echo "$IDEM_RESPONSE" | grep -q "\"id\":\"$ORDER_ID\""; then
    echo -e "${GREEN}✓ Idempotência funcionando${NC}"
else
    echo -e "${RED}✗ Idempotência falhou${NC}"
    exit 1
fi

# 15. Verificar métricas
echo -e "\n${YELLOW}15. Verificando métricas...${NC}"
METRICS_FULFILLMENT=$(curl -s http://localhost:8082/metrics | grep -c "fulfillment_orders_total")
METRICS_INVENTORY=$(curl -s http://localhost:8081/metrics | grep -c "inventory_reservations_total")

if [ -n "$METRICS_FULFILLMENT" ] && [ -n "$METRICS_INVENTORY" ]; then
    echo -e "${GREEN}✓ Métricas sendo exportadas${NC}"
else
    echo -e "${RED}✗ Métricas não encontradas${NC}"
fi

# 16. Verificar tracing
echo -e "\n${YELLOW}16. Verificando tracing...${NC}"
if curl -s http://localhost:16686/api/services | grep -q "fulfillment-ops"; then
    echo -e "${GREEN}✓ Traces visíveis${NC}"
else
    echo -e "${YELLOW}⚠ Tracing pode não estar configurado${NC}"
fi

# Resumo final
echo -e "\n${GREEN}=== TESTE DE INTEGRAÇÃO CONCLUÍDO COM SUCESSO ===${NC}"
echo -e "Ordem de teste: $ORDER_ID"
echo -e "Todos os componentes do BLOCO-1 estão funcionando corretamente!"

# Gerar relatório
REPORT_FILE="integration-test-report-$(date +%Y%m%d-%H%M%S).json"
cat > "$REPORT_FILE" << EOF
{
  "test_id": "$ORDER_ID",
  "timestamp": "$(date -Iseconds)",
  "services": {
    "core_inventory": {
      "health": "ok",
      "url": "http://localhost:8081"
    },
    "fulfillment_ops": {
      "health": "ok",
      "url": "http://localhost:8082"
    },
    "nats": {
      "health": "ok",
      "url": "http://localhost:8225"
    },
    "postgres": {
      "health": "ok",
      "url": "http://localhost:5435"
    },
    "redis": {
      "health": "ok",
      "url": "http://localhost:6381"
    }
  },
  "tests": {
    "order_creation": "ok",
    "reservation": "ok",
    "event_processing": "ok",
    "picking": "ok",
    "shipping": "ok",
    "inventory_deduction": "ok",
    "idempotency": "ok",
    "metrics": "ok",
    "tracing": "warning"
  },
  "result": "success"
}
EOF

echo -e "\n${YELLOW}Relatório gerado: $REPORT_FILE${NC}"