#!/bin/bash
# Script para configurar variáveis de ambiente

set -e

ENV_FILE=".env"
ENV_EXAMPLE=".env.example"

echo "⚙️  Configurando variáveis de ambiente..."

# Criar .env.example se não existir
if [ ! -f "$ENV_EXAMPLE" ]; then
    cat > "$ENV_EXAMPLE" << 'EOF'
# Database
DATABASE_URL=postgres://user:password@localhost:5432/fulfillment?sslmode=disable

# NATS
NATS_URL=nats://localhost:4222

# Redis
REDIS_URL=redis://localhost:6379

# Core Inventory
CORE_INVENTORY_URL=http://localhost:8081

# HTTP Server
HTTP_PORT=:8080

# Environment
ENV=development
EOF
    echo "✅ Criado $ENV_EXAMPLE"
fi

# Criar .env se não existir
if [ ! -f "$ENV_FILE" ]; then
    cp "$ENV_EXAMPLE" "$ENV_FILE"
    echo "✅ Criado $ENV_FILE a partir de $ENV_EXAMPLE"
    echo ""
    echo "⚠️  IMPORTANTE: Edite $ENV_FILE com os valores corretos para seu ambiente"
else
    echo "ℹ️  $ENV_FILE já existe, não foi sobrescrito"
fi

echo ""
echo "📋 Variáveis de ambiente configuradas!"
echo ""
echo "Para carregar as variáveis:"
echo "  source $ENV_FILE"
echo ""
echo "Ou exporte manualmente:"
echo "  export DATABASE_URL=..."
echo "  export NATS_URL=..."
echo "  etc."

