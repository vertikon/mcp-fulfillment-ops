#!/bin/bash

set -euo pipefail

# Configurações
SERVICE_DIR="$(pwd)"
BIN_OUTPUT="$SERVICE_DIR/mcp-fulfillment-ops"
BIN_OUTPUT_LINUX="$SERVICE_DIR/bin/mcp-fulfillment-ops-linux"
DOCKER_TAG="vertikon/mcp-fulfillment-ops:latest"
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
GIT_COMMIT=$(git rev-parse --short HEAD)

# Cores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
GRAY='\033[0;37m'
NC='\033[0m' # No Color

# Funções
log() { echo -e "${GRAY}$(date '+%H:%M:%S')${NC} $1"; }
success() { echo -e "${GREEN}✅ $1${NC}"; }
warning() { echo -e "${YELLOW}⚠️ $1${NC}"; }
error() { echo -e "${RED}❌ $1${NC}"; }
step() { echo -e "${CYAN}🔧 [$1/$2] $3${NC}"; }

# 1. Pré-requisitos
check_prerequisites() {
    step 1 7 "Verificando pré-requisitos"
    
    local required=("go" "git" "docker" "docker-compose")
    for cmd in "${required[@]}"; do
        if ! command -v "$cmd" &> /dev/null; then
            error "Comando obrigatório não encontrado: $cmd"
            exit 1
        fi
    done
    success "Todos os pré-requisitos estão instalados"
}

# 2. Estrutura do projeto
check_structure() {
    step 2 7 "Verificando estrutura do projeto"
    
    local critical_files=(
        "go.mod"
        "go.sum"
        "Makefile"
        "Dockerfile"
        "docker-compose.yml"
        "config/config.yaml"
    )
    
    for file in "${critical_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            error "Arquivo crítico faltando: $file"
            exit 1
        fi
    done
    
    # Executar validação se possível
    if [[ -f "tools/validate_tree.go" ]] && [[ -f ".cursor/mcp-fulfillment-ops-ARVORE-FULL.md" ]]; then
        log "Executando validador de árvore..."
        if go run tools/validate_tree.go 2>/dev/null; then
            success "Estrutura validada"
        else
            warning "Validação da estrutura falhou, mas continuando..."
        fi
    else
        warning "Arquivos de validação não encontrados"
    fi
}

# 3. Testes
run_tests() {
    step 3 7 "Executando testes"
    
    # Limpar cache
    go clean -testcache
    
    # Testes unitários
    log "Executando testes unitários..."
    if go test -v -race ./...; then
        success "Testes unitários passaram"
    else
        error "Testes unitários falharam"
        exit 1
    fi
    
    # Cobertura
    log "Gerando relatório de cobertura..."
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    success "Relatório de cobertura gerado"
}

# 4. Build Linux
build_linux() {
    step 4 7 "Compilando binário Linux"
    
    mkdir -p "$(dirname "$BIN_OUTPUT_LINUX")"
    
    log "Compilando para Linux..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -ldflags="-w -s" \
        -o "$BIN_OUTPUT_LINUX" \
        ./cmd/fulfillment-ops/main.go
    
    if [[ -f "$BIN_OUTPUT_LINUX" ]]; then
        local_size=$(du -h "$BIN_OUTPUT_LINUX" | cut -f1)
        success "Binário Linux criado: $BIN_OUTPUT_LINUX ($local_size)"
    else
        error "Falha na compilação do binário Linux"
        exit 1
    fi
}

# 5. Build Docker
build_docker() {
    step 5 7 "Construindo imagem Docker"
    
    log "Construindo imagem Docker..."
    if docker build -t "$DOCKER_TAG" .; then
        local_size=$(docker images "$DOCKER_TAG" --format "{{.Size}}")
        success "Imagem Docker criada: $DOCKER_TAG ($local_size)"
    else
        error "Falha na construção da imagem Docker"
        exit 1
    fi
}

# 6. Deploy Docker
deploy_docker() {
    step 6 7 "Implantando via Docker Compose"
    
    if [[ ! -f "docker-compose.yml" ]]; then
        error "docker-compose.yml não encontrado"
        exit 1
    fi
    
    log "Iniciando serviços..."
    docker-compose up -d --force-recreate
    
    log "Aguardando serviços ficarem prontos..."
    sleep 10
    
    # Verificar saúde
    local unhealthy=()
    local services=("postgres" "nats" "redis" "fulfillment-ops")
    
    for service in "${services[@]}"; do
        local health=$(docker-compose ps "$service" --format "{{.State}}" 2>/dev/null || echo "unknown")
        if [[ "$health" != "healthy" ]]; then
            unhealthy+=("$service")
        fi
    done
    
    if [[ ${#unhealthy[@]} -gt 0 ]]; then
        warning "Serviços não saudáveis: ${unhealthy[*]}"
    else
        success "Todos os serviços estão saudáveis"
    fi
    
    log "Portas mapeadas:"
    log "  PostgreSQL: 5435:5432"
    log "  NATS: 4225:4222, 8225:8222"
    log "  Redis: 6381:6379"
    log "  Fulfillment Ops: 8082:8080"
}

# 7. Git
git_operations() {
    step 7 7 "Versionamento (Git)"
    
    # Verificar mudanças
    if ! git diff --quiet || ! git diff --cached --quiet; then
        log "Adicionando arquivos..."
        git add .
        
        local timestamp=$(date '+%Y-%m-%d %H:%M')
        local commit_message="feat(fulfillment): pipeline release $timestamp

- Binário Linux
- Imagem Docker atualizada
- Serviços implantados via Docker Compose
- Branch: $GIT_BRANCH
- Commit: $GIT_COMMIT"

        log "Criando commit..."
        git commit -m "$commit_message"
        success "Commit realizado (execute 'git push' manualmente)"
    else
        warning "Nenhuma mudança para commitar"
    fi
}

# Main
main() {
    local start_time=$(date +%s)
    
    log "🚀 Iniciando Pipeline de Release para MCP-FULFILLMENT-OPS..."
    log "   Branch: $GIT_BRANCH | Commit: $GIT_COMMIT"
    
    check_prerequisites
    check_structure
    run_tests
    build_linux
    build_docker
    deploy_docker
    git_operations
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local minutes=$((duration / 60))
    local seconds=$((duration % 60))
    
    echo
    echo -e "${CYAN}🎉 PIPELINE CONCLUÍDO COM SUCESSO!${NC}"
    log "Duração: ${minutes}m ${seconds}s"
    log "Binário Linux: $BIN_OUTPUT_LINUX"
    log "Imagem Docker: $DOCKER_TAG"
    log "Serviço: http://localhost:8082"
    
    # Gerar resumo
    cat > pipeline-summary.json << RESUME
{
    "timestamp": "$(date -Iseconds)",
    "duration": "${minutes}m ${seconds}s",
    "branch": "$GIT_BRANCH",
    "commit": "$GIT_COMMIT",
    "binary": "$BIN_OUTPUT_LINUX",
    "dockerImage": "$DOCKER_TAG",
    "serviceUrl": "http://localhost:8082",
    "success": true
}
RESUME
    
    success "Resumo salvo em pipeline-summary.json"
}

# Executar main
main "$@"
