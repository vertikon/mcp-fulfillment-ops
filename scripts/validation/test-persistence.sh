#!/bin/bash

# Teste de Persistência e Consistência - BLOCO-1
# Valida persistência de dados e consistência entre Core Inventory e Fulfillment Ops

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}=== INICIANDO TESTE DE PERSISTÊNCIA - BLOCO-1 ===${NC}"
echo

# Configurações
BASE_URL_FULFILLMENT="http://localhost:8082"
BASE_URL_INVENTORY="http://localhost:8081"
POSTGRES_HOST="localhost"
POSTGRES_PORT="5435"
POSTGRES_DB="fulfillment"
POSTGRES_USER="fulfillment"

# Função para executar SQL no PostgreSQL
exec_sql() {
    local sql=$1
    docker exec -i fulfillment-postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$sql"
}

# Função para verificar se tabela existe
table_exists() {
    local table=$1
    local count=$(exec_sql "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = '$table';")
    [ "$count" -gt 0 ]
}

# 1. Verificar schema do banco
echo -e "\n${YELLOW}1. Verificando schema do banco...${NC}"
echo -e "Tabelas encontradas:"
exec_sql "\dt" | awk 'NR>1 {print "  - " $1}'

# Verificar se tabelas críticas existem
CRITICAL_TABLES=("fulfillment_orders" "inventory_reservations" "inventory_stock" "inventory_batches")

echo -e "\n${YELLOW}Verificando tabelas críticas...${NC}"
for table in "${CRITICAL_TABLES[@]}"; do
    if table_exists "$table"; then
        echo -e "${GREEN}✓ $table${NC}"
    else
        echo -e "${RED}✗ $table não encontrada${NC}"
    fi
done

# 2. Testar criação de ordem com persistência
echo -e "\n${YELLOW}2. Testando criação de ordem...${NC}"
ORDER_ID="PERSIST-$(date +%s)"
TIMESTAMP=$(date -Iseconds)

# Criar ordem
ORDER_RESPONSE=$(curl -s -X POST "$BASE_URL_FULFILLMENT/api/v1/fulfillment-orders" \
    -H "Content-Type: application/json" \
    -d "{
        \"order_id\": \"$ORDER_ID\",
        \"customer\": \"CUSTOMER-PERSIST-TEST\",
        \"destination\": \"Rua Teste Persistência, 456 - São Paulo/SP\",
        \"items\": [
            {\"sku\": \"PROD-PERSIST-001\", \"quantity\": 5}
        ],
        \"priority\": 0,
        \"idempotency_key\": \"$ORDER_ID\"
    }")

if echo "$ORDER_RESPONSE" | grep -q "\"id\":\"$ORDER_ID\""; then
    echo -e "${GREEN}✓ Ordem criada${NC}"
else
    echo -e "${RED}✗ Falha ao criar ordem${NC}"
    echo "Response: $ORDER_RESPONSE"
    exit 1
fi

# 3. Verificar persistência da ordem
echo -e "\n${YELLOW}3. Verificando persistência da ordem...${NC}"
sleep 2

# Consultar ordem no banco
ORDER_DB=$(exec_sql "SELECT * FROM fulfillment_orders WHERE order_id = '$ORDER_ID'")

if echo "$ORDER_DB" | grep -q "$ORDER_ID"; then
    echo -e "${GREEN}✓ Ordem persistida${NC}"
else
    echo -e "${RED}✗ Ordem não encontrada no banco${NC}"
    exit 1
fi

# 4. Verificar reserva no Core Inventory
echo -e "\n${YELLOW}4. Verificando reserva no Core Inventory...${NC}"
sleep 1

# Consultar reserva
RESERVATION=$(curl -s "$BASE_URL_INVENTORY/api/v1/reservations" \
    -H "Content-Type: application/json" \
    -G -d "reference_id=$ORDER_ID")

if echo "$RESERVATION" | grep -q "\"reference_id\":\"$ORDER_ID\""; then
    echo -e "${GREEN}✓ Reserva encontrada${NC}"
else
    echo -e "${RED}✗ Reserva não encontrada${NC}"
    exit 1
fi

# 5. Publicar evento no NATS
echo -e "\n${YELLOW}5. Publicando evento de teste...${NC}"
docker exec -it fulfillment-nats nats pub oms.order.ready_to_pick "{
    \"order_id\": \"$ORDER_ID\",
    \"customer_id\": \"CUST-PERSIST-TEST\",
    \"items\": [
        {\"sku\": \"PROD-PERSIST-001\", \"quantity\": 5}
    ]
}"

echo -e "${GREEN}✓ Evento publicado${NC}"

# 6. Aguardar processamento do evento
echo -e "\n${YELLOW}6. Aguardando processamento do evento...${NC}"
sleep 3

# 7. Verificar status da ordem
echo -e "\n${YELLOW}7. Verificando status da ordem...${NC}"
STATUS=$(curl -s "$BASE_URL_FULFILLMENT/api/v1/fulfillment-orders/$ORDER_ID" \
    -H "Content-Type: application/json" | jq -r '.status')

if [ "$STATUS" = "PENDING" ]; then
    echo -e "${GREEN}✓ Ordem em processamento${NC}"
else
    echo -e "${RED}✗ Status inesperado: $STATUS${NC}"
    exit 1
fi

# 8. Iniciar picking
echo -e "\n${YELLOW}8. Iniciando picking...${NC}"
PICK_RESPONSE=$(curl -s -X POST "$BASE_URL_FULFILLMENT/api/v1/fulfillment-orders/$ORDER_ID/pick" \
    -H "Content-Type: application/json")

if echo "$PICK_RESPONSE" | grep -q "\"status\":\"IN_PROGRESS\""; then
    echo -e "${GREEN}✓ Picking iniciado${NC}"
else
    echo -e "${RED}✗ Falha ao iniciar picking${NC}"
    exit 1
fi

# 9. Confirmar expedição
echo -e "\n${YELLOW}9. Confirmando expedição...${NC}"
SHIP_RESPONSE=$(curl -s -X POST "$BASE_URL_FULFILLMENT/api/v1/fulfillment-orders/$ORDER_ID/ship" \
    -H "Content-Type: application/json" \
    -d "{
        \"tracking_number\": \"TRACK-$(date +%s)\",
        \"carrier\": \"PERSISTENCE-CARRIER\"
    }")

if echo "$SHIP_RESPONSE" | grep -q "\"status\":\"COMPLETED\""; then
    echo -e "${GREEN}✓ Expedição confirmada${NC}"
else
    echo -e "${RED}✗ Falha ao confirmar expedição${NC}"
    exit 1
fi

# 10. Verificar baixa no estoque
echo -e "\n${YELLOW}10. Verificando baixa no estoque...${NC}"
sleep 2

STOCK=$(curl -s "$BASE_URL_INVENTORY/api/v1/inventory/PROD-PERSIST-001" \
    -H "Content-Type: application/json")

# Extrair quantidade disponível (deve ser menor que antes)
AVAILABLE=$(echo "$STOCK" | jq -r '.available')

if [ "$AVAILABLE" -lt 100 ]; then
    echo -e "${GREEN}✓ Estoque baixado${NC} (Disponível: $AVAILABLE)"
else
    echo -e "${RED}✗ Estoque não foi baixado${NC}"
    exit 1
fi

# 11. Verificar eventos publicados
echo -e "\n${YELLOW}11. Verificando eventos publicados...${NC}"
docker exec -it fulfillment-nats nats stream info fulfillment.engine.tasks | grep -q "messages" || {
    echo -e "${RED}✗ Sem mensagens no stream${NC}"
    exit 1
}
echo -e "${GREEN}✓ Eventos publicados${NC}"

# 12. Testar idempotência
echo -e "\n${YELLOW}12. Testando idempotência...${NC}"
IDEM_RESPONSE=$(curl -s -X POST "$BASE_URL_FULFILLMENT/api/v1/fulfillment-orders" \
    -H "Content-Type: application/json" \
    -d "{
        \"order_id\": \"$ORDER_ID-REPEAT\",
        \"customer\": \"CUSTOMER-PERSIST-TEST\",
        \"items\": [
            {\"sku\": \"PROD-PERSIST-001\", \"quantity\": 3}
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

# 13. Testar concorrência
echo -e "\n${YELLOW}13. Testando concorrência...${NC}"

# Criar múltiplas ordens concorrentes
for i in {1..3}; do
    CONCURRENT_ORDER_ID="CONCURRENT-$ORDER_ID-$i"
    curl -s -X POST "$BASE_URL_FULFILLMENT/api/v1/fulfillment-orders" \
        -H "Content-Type: application/json" \
        -d "{
            \"order_id\": \"$CONCURRENT_ORDER_ID\",
            \"customer\": \"CUSTOMER-CONCURRENT-$i\",
            \"items\": [
                {\"sku\": \"PROD-CONCURRENT-001\", \"quantity\": 1}
            ],
            \"priority\": 0,
            \"idempotency_key\": \"$CONCURRENT_ORDER_ID\"
        }" > /dev/null &
done

# Aguardar processamento
sleep 5

# Verificar se todas as ordens foram criadas
CONCURRENT_COUNT=$(exec_sql "SELECT COUNT(*) FROM fulfillment_orders WHERE order_id LIKE 'CONCURRENT-$ORDER_ID-%'")

if [ "$CONCURRENT_COUNT" -eq 3 ]; then
    echo -e "${GREEN}✓ Ordens concorrentes criadas${NC}"
else
    echo -e "${RED}✗ Apenas $CONCURRENT_COUNT ordens criadas${NC}"
    exit 1
fi

# 14. Testar consistência de estoque
echo -e "\n${YELLOW}14. Testando consistência de estoque...${NC}"

# Verificar estoque disponível
STOCK_BEFORE=$(exec_sql "SELECT available FROM inventory_stock WHERE sku = 'PROD-PERSIST-001'")
STOCK_RESERVED=$(exec_sql "SELECT SUM(reserved_quantity) FROM inventory_reservations WHERE sku = 'PROD-PERSIST-001' AND status = 'PENDING'")

echo -e "Estoque disponível: $STOCK_BEFORE"
echo -e "Estoque reservado: $STOCK_RESERVED"

# Verificar se não há overselling
if [ "$STOCK_RESERVED" -gt "$STOCK_BEFORE" ]; then
    echo -e "${RED}✗ Overselling detectado${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Estoque consistente${NC}"
fi

# 15. Testar rollback
echo -e "\n${YELLOW}15. Testando rollback...${NC}"

# Criar ordem para teste de rollback
ROLLBACK_ORDER_ID="ROLLBACK-$ORDER_ID"
ROLLBACK_RESPONSE=$(curl -s -X POST "$BASE_URL_FULFILLMENT/api/v1/fulfillment-orders" \
    -H "Content-Type: application/json" \
    -d "{
        \"order_id\": \"$ROLLBACK_ORDER_ID\",
        \"customer\": \"CUSTOMER-ROLLBACK-TEST\",
        \"items\": [
            {\"sku\": \"PROD-ROLLBACK-001\", \"quantity\": 10}
        ],
        \"priority\": 0,
        \"idempotency_key\": \"$ROLLBACK_ORDER_ID\"
    }")

if echo "$ROLLBACK_RESPONSE" | grep -q "\"id\":\"$ROLLBACK_ORDER_ID\""; then
    echo -e "${GREEN}✓ Ordem de rollback criada${NC}"
else
    echo -e "${RED}✗ Falha ao criar ordem${NC}"
    exit 1
fi

# Cancelar ordem (simular rollback)
CANCEL_RESPONSE=$(curl -s -X POST "$BASE_URL_FULFILLMENT/api/v1/fulfillment-orders/$ROLLBACK_ORDER_ID/cancel" \
    -H "Content-Type: application/json")

if echo "$CANCEL_RESPONSE" | grep -q "\"status\":\"CANCELLED\""; then
    echo -e "${GREEN}✓ Ordem cancelada${NC}"
else
    echo -e "${RED}✗ Falha ao cancelar${NC}"
    exit 1
fi

# Verificar se reservas foram liberadas
sleep 2
RELEASED_STOCK=$(exec_sql "SELECT SUM(reserved_quantity) FROM inventory_reservations WHERE reference_id = '$ROLLBACK_ORDER_ID' AND status = 'CANCELLED'")

if [ -n "$RELEASED_STOCK" ]; then
    echo -e "${GREEN}✓ Reservas liberadas${NC}"
else
    echo -e "${RED}✗ Reservas não liberadas${NC}"
    exit 1
fi

# 16. Gerar relatório
echo -e "\n${YELLOW}16. Gerando relatório...${NC}"

REPORT_FILE="persistence-test-report-$(date +%Y%m%d-%H%M%S).json"
cat > "$REPORT_FILE" << EOF
{
  "test_id": "$ORDER_ID",
  "timestamp": "$TIMESTAMP",
  "tests": {
    "order_creation": {
      "status": "ok",
      "order_id": "$ORDER_ID"
    },
    "order_persistence": {
      "status": "ok",
      "order_id": "$ORDER_ID"
    },
    "reservation_verification": {
      "status": "ok",
      "reference_id": "$ORDER_ID"
    },
    "event_processing": {
      "status": "ok"
    },
    "picking": {
      "status": "ok"
    },
    "shipping": {
      "status": "ok"
    },
    "inventory_deduction": {
      "status": "ok",
      "available": $STOCK_BEFORE,
      "reserved": $STOCK_RESERVED
    },
    "idempotency": {
      "status": "ok"
    },
    "concurrency": {
      "status": "ok",
      "orders_created": 3
    },
    "stock_consistency": {
      "status": "ok"
    },
    "rollback": {
      "status": "ok",
      "order_id": "$ROLLBACK_ORDER_ID",
      "reservations_released": $RELEASED_STOCK
    }
  },
  "result": "success"
}
EOF

echo -e "\n${GREEN}=== TESTE DE PERSISTÊNCIA CONCLUÍDO COM SUCESSO ===${NC}"
echo -e "Ordem de teste: $ORDER_ID"
echo -e "Todos os componentes do BLOCO-1 estão funcionando corretamente!"
echo -e "\n${YELLOW}Relatório gerado: $REPORT_FILE${NC}"