#!/bin/bash
# Script de validação de estrutura do projeto mcp-fulfillment-ops
# Uso: ./scripts/validation/validate_project_structure.sh [--strict] [--format json|markdown|text]

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configurações
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ORIGINAL_TREE="${PROJECT_ROOT}/.cursor/mcp-fulfillment-ops-ARVORE-FULL.md"
COMMENTED_TREE="${PROJECT_ROOT}/.cursor/ARVORE-ARQUIVOS-DIRETORIOS-COMENTADA.md"
VALIDATE_TOOL="${PROJECT_ROOT}/bin/validate-tree"
OUTPUT_FORMAT="${OUTPUT_FORMAT:-markdown}"
STRICT_MODE="${STRICT_MODE:-false}"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --strict)
            STRICT_MODE="true"
            shift
            ;;
        --format)
            OUTPUT_FORMAT="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--strict] [--format json|markdown|text]"
            exit 1
            ;;
    esac
done

echo -e "${GREEN}🔍 Validando estrutura do projeto mcp-fulfillment-ops...${NC}"
echo ""

# Verificar se a ferramenta existe
if [ ! -f "$VALIDATE_TOOL" ] && [ ! -f "${VALIDATE_TOOL}.exe" ]; then
    echo -e "${YELLOW}⚠️  Ferramenta validate-tree não encontrada. Compilando...${NC}"
    cd "$PROJECT_ROOT"
    go build -o bin/validate-tree ./tools/validate_tree.go
    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ Erro ao compilar ferramenta de validação${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Ferramenta compilada com sucesso${NC}"
    echo ""
fi

# Determinar executável (Windows ou Unix)
if [ -f "${VALIDATE_TOOL}.exe" ]; then
    VALIDATE_TOOL="${VALIDATE_TOOL}.exe"
fi

# Verificar se os arquivos de árvore existem
if [ ! -f "$ORIGINAL_TREE" ]; then
    echo -e "${RED}❌ Árvore original não encontrada: $ORIGINAL_TREE${NC}"
    exit 1
fi

if [ ! -f "$COMMENTED_TREE" ]; then
    echo -e "${RED}❌ Árvore comentada não encontrada: $COMMENTED_TREE${NC}"
    exit 1
fi

# Executar validação
echo -e "${GREEN}📊 Executando validação...${NC}"
echo ""

VALIDATE_CMD="$VALIDATE_TOOL"
VALIDATE_CMD="$VALIDATE_CMD --original \"$ORIGINAL_TREE\""
VALIDATE_CMD="$VALIDATE_CMD --commented \"$COMMENTED_TREE\""
VALIDATE_CMD="$VALIDATE_CMD --root \"$PROJECT_ROOT\""
VALIDATE_CMD="$VALIDATE_CMD --format $OUTPUT_FORMAT"

if [ "$STRICT_MODE" = "true" ]; then
    VALIDATE_CMD="$VALIDATE_CMD --strict"
fi

# Executar e capturar resultado
OUTPUT_DIR="${PROJECT_ROOT}/.cursor/validation-reports"
mkdir -p "$OUTPUT_DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="${OUTPUT_DIR}/validation-report-${TIMESTAMP}.${OUTPUT_FORMAT}"

eval $VALIDATE_CMD > "$REPORT_FILE" 2>&1
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ Validação concluída com sucesso${NC}"
    echo ""
    echo "📄 Relatório salvo em: $REPORT_FILE"
    echo ""
    
    # Mostrar resumo se for markdown ou text
    if [ "$OUTPUT_FORMAT" = "markdown" ] || [ "$OUTPUT_FORMAT" = "text" ]; then
        echo "📊 Resumo:"
        head -20 "$REPORT_FILE"
    fi
    
    exit 0
else
    echo -e "${RED}❌ Validação falhou${NC}"
    echo ""
    echo "📄 Relatório de erros salvo em: $REPORT_FILE"
    echo ""
    
    # Mostrar erros
    if [ "$OUTPUT_FORMAT" = "markdown" ] || [ "$OUTPUT_FORMAT" = "text" ]; then
        cat "$REPORT_FILE"
    fi
    
    exit $EXIT_CODE
fi

