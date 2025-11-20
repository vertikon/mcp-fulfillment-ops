#!/bin/bash
# Script to install pre-commit hook for tree validation
# Usage: ./scripts/setup/pre-commit-install.sh

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HOOK_SOURCE="${PROJECT_ROOT}/.git/hooks/pre-commit"
HOOK_TARGET="${PROJECT_ROOT}/.git/hooks/pre-commit"

echo "🔧 Installing pre-commit hook for tree validation..."

# Check if .git directory exists
if [ ! -d "${PROJECT_ROOT}/.git" ]; then
    echo "❌ Error: .git directory not found. Are you in a git repository?"
    exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p "${PROJECT_ROOT}/.git/hooks"

# Copy pre-commit hook
if [ -f "${PROJECT_ROOT}/.git/hooks/pre-commit" ]; then
    echo "⚠️  Pre-commit hook already exists. Backing up..."
    cp "${PROJECT_ROOT}/.git/hooks/pre-commit" "${PROJECT_ROOT}/.git/hooks/pre-commit.backup.$(date +%Y%m%d_%H%M%S)"
fi

# Copy the hook
cp "${PROJECT_ROOT}/.git/hooks/pre-commit" "${HOOK_TARGET}" 2>/dev/null || \
cat > "${HOOK_TARGET}" << 'HOOK_CONTENT'
#!/bin/bash
# Pre-commit hook for mcp-fulfillment-ops tree structure validation
# This hook validates the project structure before allowing commits

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🔍 Running pre-commit tree validation...${NC}"

# Get project root
PROJECT_ROOT="$(git rev-parse --show-toplevel)"
cd "$PROJECT_ROOT"

# Check if validate-tree exists
VALIDATE_TOOL="${PROJECT_ROOT}/bin/validate-tree"
if [ ! -f "$VALIDATE_TOOL" ] && [ ! -f "${VALIDATE_TOOL}.exe" ]; then
    echo -e "${YELLOW}⚠️  validate-tree not found. Building...${NC}"
    go build -o bin/validate-tree ./tools/validate_tree.go
    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ Failed to build validate-tree${NC}"
        exit 1
    fi
fi

# Determine executable
if [ -f "${VALIDATE_TOOL}.exe" ]; then
    VALIDATE_TOOL="${VALIDATE_TOOL}.exe"
fi

# Run validation (non-strict mode for pre-commit to allow warnings)
echo -e "${GREEN}📊 Validating tree structure...${NC}"

$VALIDATE_TOOL \
    --original .cursor/mcp-fulfillment-ops-ARVORE-FULL.md \
    --commented .cursor/ARVORE-ARQUIVOS-DIRETORIOS-COMENTADA.md \
    --root . \
    --format text > /tmp/tree-validation.txt 2>&1 || VALIDATION_EXIT=$?

if [ -n "$VALIDATION_EXIT" ]; then
    echo -e "${RED}❌ Tree validation failed${NC}"
    echo ""
    echo "Validation output:"
    cat /tmp/tree-validation.txt
    echo ""
    echo -e "${YELLOW}💡 Tip: Run './bin/validate-tree --format markdown' for detailed report${NC}"
    exit 1
fi

# Check compliance from output
COMPLIANCE=$(grep -oP 'Compliance:\s+\K\d+\.\d+' /tmp/tree-validation.txt | head -1 || echo "0")

if [ -n "$COMPLIANCE" ]; then
    # Convert to integer for comparison (95.0 -> 95)
    COMPLIANCE_INT=$(echo "$COMPLIANCE" | cut -d. -f1)
    
    if [ "$COMPLIANCE_INT" -lt 95 ]; then
        echo -e "${RED}❌ Compliance below threshold (95%)${NC}"
        echo -e "${RED}   Current compliance: ${COMPLIANCE}%${NC}"
        cat /tmp/tree-validation.txt
        exit 1
    fi
    
    echo -e "${GREEN}✅ Tree validation passed (Compliance: ${COMPLIANCE}%)${NC}"
else
    echo -e "${GREEN}✅ Tree validation passed${NC}"
fi

exit 0
HOOK_CONTENT

# Make hook executable
chmod +x "${HOOK_TARGET}"

echo "✅ Pre-commit hook installed successfully"
echo ""
echo "The hook will now validate tree structure before each commit."
echo "To disable temporarily, use: git commit --no-verify"

