#!/bin/bash
# Script de deploy para mcp-fulfillment-ops

set -e

ENV="${1:-staging}"

echo "🚀 Iniciando deploy para ambiente: $ENV"
echo ""

# Validar que estamos no diretório correto
if [ ! -f "go.mod" ]; then
    echo "❌ Erro: go.mod não encontrado. Execute este script na raiz do projeto."
    exit 1
fi

# Carregar variáveis de ambiente
if [ -f ".env.$ENV" ]; then
    echo "📋 Carregando variáveis de ambiente de .env.$ENV..."
    set -a
    source ".env.$ENV"
    set +a
elif [ -f ".env" ]; then
    echo "📋 Carregando variáveis de ambiente de .env..."
    set -a
    source ".env"
    set +a
else
    echo "⚠️  Nenhum arquivo .env encontrado. Usando variáveis do sistema."
fi

# Executar testes
echo ""
echo "🧪 Executando testes..."
if go test ./tests/domain/... -v; then
    echo "✅ Testes passaram"
else
    echo "❌ Testes falharam. Abortando deploy."
    exit 1
fi

# Build
echo ""
echo "🔨 Compilando binário..."
BINARY_NAME="mcp-fulfillment-ops"
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"

CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -o "bin/$BINARY_NAME" ./cmd/fulfillment-ops

if [ -f "bin/$BINARY_NAME" ]; then
    echo "✅ Binário compilado: bin/$BINARY_NAME"
else
    echo "❌ Falha na compilação"
    exit 1
fi

# Docker build (se Dockerfile existir)
if [ -f "Dockerfile" ]; then
    echo ""
    echo "🐳 Construindo imagem Docker..."
    IMAGE_NAME="mcp-fulfillment-ops:$ENV"
    docker build -t "$IMAGE_NAME" .
    echo "✅ Imagem Docker construída: $IMAGE_NAME"
fi

# Verificar dependências externas
echo ""
echo "🔍 Verificando dependências externas..."

# PostgreSQL
if command -v psql &> /dev/null; then
    if PGPASSWORD="${DATABASE_PASSWORD:-password}" psql -h "${DATABASE_HOST:-localhost}" -U "${DATABASE_USER:-postgres}" -d "${DATABASE_NAME:-fulfillment}" -c "SELECT 1" &> /dev/null; then
        echo "   ✅ PostgreSQL acessível"
    else
        echo "   ⚠️  PostgreSQL não acessível (pode estar OK se ainda não configurado)"
    fi
else
    echo "   ⚠️  psql não encontrado, pulando verificação PostgreSQL"
fi

# NATS
if command -v nats &> /dev/null; then
    echo "   ✅ NATS CLI encontrado"
else
    echo "   ⚠️  NATS CLI não encontrado"
fi

# Redis
if command -v redis-cli &> /dev/null; then
    if redis-cli -u "${REDIS_URL:-redis://localhost:6379}" ping &> /dev/null; then
        echo "   ✅ Redis acessível"
    else
        echo "   ⚠️  Redis não acessível (pode estar OK se ainda não configurado)"
    fi
else
    echo "   ⚠️  redis-cli não encontrado, pulando verificação Redis"
fi

echo ""
echo "✅ Deploy preparado com sucesso!"
echo ""
echo "📦 Próximos passos:"
echo "   1. Execute migrations: psql < internal/adapters/postgres/migrations/0001_create_fulfillment_tables.sql"
echo "   2. Inicie o serviço: ./bin/$BINARY_NAME"
echo "   3. Verifique health: curl http://localhost:8080/health"
echo ""
echo "🐳 Ou use Docker:"
echo "   docker run -p 8080:8080 --env-file .env.$ENV $IMAGE_NAME"

