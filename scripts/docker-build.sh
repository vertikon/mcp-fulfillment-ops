#!/bin/bash
# Script para build e execução via Docker

set -e

echo "🐳 Build e execução via Docker"
echo ""

# Build da imagem
echo "🔨 Construindo imagem Docker..."
docker build -t mcp-fulfillment-ops:latest .

if [ $? -eq 0 ]; then
    echo "✅ Imagem construída com sucesso"
else
    echo "❌ Falha ao construir imagem"
    exit 1
fi

echo ""
echo "🚀 Para executar:"
echo "   docker-compose up -d"
echo ""
echo "Ou executar standalone:"
echo "   docker run -p 8080:8080 \\"
echo "     -e DATABASE_URL=postgres://... \\"
echo "     -e NATS_URL=nats://... \\"
echo "     -e REDIS_URL=redis://... \\"
echo "     -e CORE_INVENTORY_URL=http://... \\"
echo "     mcp-fulfillment-ops:latest"

