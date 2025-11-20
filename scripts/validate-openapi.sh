#!/bin/bash
# Script para validar contratos OpenAPI

set -e

echo "🔍 Validando contratos OpenAPI..."

OPENAPI_DIR="../../../../contracts/openapi/bloco-1-core"
VALIDATOR_URL="https://validator.swagger.io/validator/debug"

# Validar v1
echo ""
echo "📄 Validando fulfillment-ops-v1.yaml..."
curl -X POST "$VALIDATOR_URL" \
  -H "Content-Type: application/yaml" \
  --data-binary "@$OPENAPI_DIR/fulfillment-ops-v1.yaml" \
  | jq '.'

# Validar v2
echo ""
echo "📄 Validando fulfillment-ops-v2.yaml..."
curl -X POST "$VALIDATOR_URL" \
  -H "Content-Type: application/yaml" \
  --data-binary "@$OPENAPI_DIR/fulfillment-ops-v2.yaml" \
  | jq '.'

echo ""
echo "✅ Validação concluída!"
echo ""
echo "💡 Para visualizar no Swagger Editor:"
echo "   https://editor.swagger.io/"
echo "   Cole o conteúdo do arquivo YAML no editor"

