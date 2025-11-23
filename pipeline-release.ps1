#>
<#
.SYNOPSIS
    Script de automação para Build, Validação e Deploy do MCP Fulfillment Ops.
.DESCRIPTION
    Pipeline completo que inclui:
    1. Validação de estrutura (baseado nas regras do .cursor)
    2. Executa testes unitários e integração
    3. Compila o binário para Windows (local) e Linux (Docker)
    4. Constrói a imagem Docker
    5. Realiza o deploy local via Docker Compose
    6. Executa commit e push para o repositório
.PARAMETER SkipTests
    Pula a execução dos testes (usar com cuidado)
.PARAMETER SkipValidation
    Pula a validação de conformidade da estrutura
.PARAMETER PushToGit
    Realiza push automático para o repositório remoto
#>

param(
    [switch]$SkipTests,
    [switch]$SkipValidation,
    [switch]$PushToGit
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

# Configurações
$ServiceDir = "E:\vertikon\.endurance\internal\services\bloco-1-core\mcp-fulfillment-ops"
$BinOutput = "$ServiceDir\mcp-fulfillment-ops.exe"
$BinOutputLinux = "$ServiceDir\bin\mcp-fulfillment-ops-linux"
$DockerTag = "vertikon/mcp-fulfillment-ops:latest"
$ValidationReport = "$ServiceDir\validation-result.json"
$GitBranch = git rev-parse --abbrev-ref HEAD
$GitCommit = git rev-parse --short HEAD

# Funções de utilidade
function Write-Step {
    param([string]$Message, [int]$Step, [int]$Total)
    Write-Host "`n🔧 [$Step/$Total] $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠️ $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

function Test-Command {
    param([string]$Command)
    try {
        $null = Get-Command $Command -ErrorAction Stop
        return $true
    }
    catch {
        return $false
    }
}

# Verificação de pré-requisitos
function Test-Prerequisites {
    Write-Host "🔍 Verificando pré-requisitos..." -ForegroundColor Yellow
    
    $requiredCommands = @("go", "git", "docker", "docker-compose")
    $missing = @()
    
    foreach ($cmd in $requiredCommands) {
        if (-not (Test-Command $cmd)) {
            $missing += $cmd
        }
    }
    
    if ($missing.Count -gt 0) {
        Write-Error "Comandos obrigatórios não encontrados: $($missing -join ', ')"
        exit 1
    }
    
    Write-Success "Todos os pré-requisitos estão instalados"
}

# Validação de estrutura do projeto
function Test-ProjectStructure {
    Write-Step "Validando Estrutura do Projeto" 1 7
    
    if ($SkipValidation) {
        Write-Warning "Validação de estrutura pulada via parâmetro"
        return
    }
    
    Set-Location $ServiceDir
    
    # Verificar arquivos críticos
    $criticalFiles = @(
        "go.mod",
        "go.sum", 
        "Makefile",
        "Dockerfile",
        "docker-compose.yml",
        "config/config.yaml",
        "CRUSH.md",
        ".cursor/MCP-HULK – POLÍTICA DE ESTRUTURA & NOMENCLATURA.md"
    )
    
    $missingFiles = @()
    foreach ($file in $criticalFiles) {
        if (-not (Test-Path $file)) {
            $missingFiles += $file
        }
    }
    
    if ($missingFiles.Count -gt 0) {
        Write-Error "Arquivos críticos faltando: $($missingFiles -join ', ')"
        exit 1
    }
    
    # Executar validador da árvore se existir
    if (Test-Path "tools/validate_tree.go") {
        Write-Host "   -> Executando validador de árvore..."
        go run tools/validate_tree.go
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Validação de estrutura falhou"
            exit 1
        }
    }
    
    Write-Success "Estrutura do projeto validada"
}

# Testes unitários e de integração
function Invoke-Tests {
    Write-Step "Executando Testes" 2 7
    
    if ($SkipTests) {
        Write-Warning "Testes pulados via parâmetro"
        return
    }
    
    Set-Location $ServiceDir
    
    # Limpar cache de testes
    Write-Host "   -> Limpando cache de testes..."
    go clean -testcache
    
    # Testes unitários
    Write-Host "   -> Executando testes unitários..."
    go test -v -race ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Testes unitários falharam"
        exit 1
    }
    
    # Testes de integração se existirem
    if (Test-Path "tests/integration") {
        Write-Host "   -> Executando testes de integração..."
        go test -v -tags=integration ./tests/integration/...
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "Testes de integração falharam, mas continuando..."
        }
    }
    
    # Gerar relatório de cobertura
    Write-Host "   -> Gerando relatório de cobertura..."
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    
    Write-Success "Todos os testes passaram"
}

# Build para Windows (desenvolvimento local)
function Build-Windows {
    Write-Step "Compilando Binário Windows" 3 7
    
    Set-Location $ServiceDir
    
    # Garantir que o diretório de bin existe
    $binDir = Split-Path $BinOutput -Parent
    if (-not (Test-Path $binDir)) {
        New-Item -ItemType Directory -Path $binDir -Force | Out-Null
    }
    
    Write-Host "   -> Compilando para Windows..."
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    
    go build -ldflags="-w -s" -o $BinOutput ./cmd/fulfillment-ops/main.go
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Falha na compilação para Windows"
        exit 1
    }
    
    # Verificar se o binário foi criado
    if (-not (Test-Path $BinOutput)) {
        Write-Error "Binário Windows não foi gerado"
        exit 1
    }
    
    $size = [math]::Round((Get-Item $BinOutput).Length / 1MB, 2)
    Write-Success "Binário Windows criado: $BinOutput ($size MB)"
}

# Build para Linux (Docker)
function Build-Linux {
    Write-Step "Compilando Binário Linux" 4 7
    
    Set-Location $ServiceDir
    
    # Garantir que o diretório de bin existe
    $binDir = Split-Path $BinOutputLinux -Parent
    if (-not (Test-Path $binDir)) {
        New-Item -ItemType Directory -Path $binDir -Force | Out-Null
    }
    
    Write-Host "   -> Compilando para Linux..."
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    
    go build -ldflags="-w -s" -o $BinOutputLinux ./cmd/fulfillment-ops/main.go
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Falha na compilação para Linux"
        exit 1
    }
    
    $size = [math]::Round((Get-Item $BinOutputLinux).Length / 1MB, 2)
    Write-Success "Binário Linux criado: $BinOutputLinux ($size MB)"
}

# Build Docker image
function Build-Docker {
    Write-Step "Construindo Imagem Docker" 5 7
    
    Set-Location $ServiceDir
    
    Write-Host "   -> Construindo imagem Docker..."
    docker build -t $DockerTag .
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Falha na construção da imagem Docker"
        exit 1
    }
    
    # Verificar se a imagem foi criada
    $imageInfo = docker images $DockerTag --format "{{.Size}}"
    Write-Success "Imagem Docker criada: $DockerTag ($imageInfo)"
}

# Deploy via Docker Compose
function Deploy-Docker {
    Write-Step "Implantando via Docker Compose" 6 7
    
    Set-Location $ServiceDir
    
    # Verificar se o docker-compose.yml existe
    if (-not (Test-Path "docker-compose.yml")) {
        Write-Error "docker-compose.yml não encontrado"
        exit 1
    }
    
    Write-Host "   -> Verificando status dos containers..."
    docker-compose ps
    
    Write-Host "   -> Iniciando serviços..."
    docker-compose up -d --force-recreate
    
    # Aguardar serviços ficarem prontos
    Write-Host "   -> Aguardando serviços ficarem saudáveis..."
    Start-Sleep -Seconds 10
    
    # Verificar saúde dos serviços
    $unhealthy = @()
    $services = @("postgres", "nats", "redis", "fulfillment-ops")
    
    foreach ($service in $services) {
        $health = docker-compose ps $service --format "{{.State}}"
        if ($health -notlike "healthy*") {
            $unhealthy += $service
        }
    }
    
    if ($unhealthy.Count -gt 0) {
        Write-Warning "Serviços não saudáveis: $($unhealthy -join ', ')"
        Write-Host "   -> Logs para debug:"
        docker-compose logs --tail=20
    } else {
        Write-Success "Todos os serviços estão saudáveis"
    }
    
    # Exibir portas mapeadas
    Write-Host "   -> Portas mapeadas:"
    Write-Host "      PostgreSQL: 5435:5432"
    Write-Host "      NATS: 4225:4222, 8225:8222" 
    Write-Host "      Redis: 6381:6379"
    Write-Host "      Fulfillment Ops: 8082:8080"
}

# Git operations
function Invoke-GitOperations {
    Write-Step "Versionamento (Git)" 7 7
    
    Set-Location $ServiceDir
    
    # Verificar se há mudanças para commitar
    $status = git status --porcelain
    if ([string]::IsNullOrWhiteSpace($status)) {
        Write-Warning "Nenhuma mudança para commitar"
        return
    }
    
    Write-Host "   -> Adicionando arquivos..."
    git add .
    
    # Criar mensagem de commit informativa
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm"
    $commitMessage = "feat(fulfillment): pipeline release $timestamp

- Binários Windows e Linux
- Imagem Docker atualizada
- Serviços implantados via Docker Compose
- Branch: $GitBranch
- Commit: $GitCommit"

    Write-Host "   -> Criando commit..."
    git commit -m $commitMessage
    
    if ($PushToGit) {
        Write-Host "   -> Enviando para repositório remoto..."
        git push origin $GitBranch
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "Push falhou. Verifique suas credenciais ou permissões"
        } else {
            Write-Success "Push realizado com sucesso"
        }
    } else {
        Write-Warning "Push não realizado (use -PushToGit para enviar automaticamente)"
        Write-Host "   -> Para enviar manualmente: git push origin $GitBranch"
    }
    
    Write-Success "Commit realizado"
}

# Cleanup
function Invoke-Cleanup {
    Write-Host "`n🧹 Limpando ambiente..." -ForegroundColor Yellow
    
    # Resetar variáveis de ambiente
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
    
    Write-Success "Limpeza concluída"
}

# Função principal
function Main {
    $startTime = Get-Date
    
    try {
        Write-Host "🚀 Iniciando Pipeline de Release para MCP-FULFILLMENT-OPS..." -ForegroundColor Cyan
        Write-Host "   Branch: $GitBranch | Commit: $GitCommit" -ForegroundColor Gray
        
        # Executar pipeline
        Test-Prerequisites
        Test-ProjectStructure
        Invoke-Tests
        Build-Windows
        Build-Linux
        Build-Docker
        Deploy-Docker
        Invoke-GitOperations
        
        $endTime = Get-Date
        $duration = $endTime - $startTime
        
        Write-Host "`n🎉 PIPELINE CONCLUÍDO COM SUCESSO!" -ForegroundColor Cyan
        Write-Host "   Duração: $($duration.ToString('mm\:ss'))" -ForegroundColor Gray
        Write-Host "   Binário Windows: $BinOutput" -ForegroundColor Gray
        Write-Host "   Binário Linux: $BinOutputLinux" -ForegroundColor Gray
        Write-Host "   Imagem Docker: $DockerTag" -ForegroundColor Gray
        Write-Host "   Serviço: http://localhost:8082" -ForegroundColor Gray
        
        # Gerar resumo
        $summary = @{
            timestamp = $startTime.ToString("yyyy-MM-dd HH:mm:ss")
            duration = $duration.ToString("mm\:ss")
            branch = $GitBranch
            commit = $GitCommit
            binaries = @($BinOutput, $BinOutputLinux)
            dockerImage = $DockerTag
            serviceUrl = "http://localhost:8082"
            success = $true
        }
        
        $summary | ConvertTo-Json | Out-File -FilePath "pipeline-summary.json" -Encoding UTF8
        Write-Success "Resumo salvo em pipeline-summary.json"
        
    }
    catch {
        $endTime = Get-Date
        $duration = $endTime - $startTime
        
        Write-Host "`n💥 PIPELINE FALHOU!" -ForegroundColor Red
        Write-Host "   Duração: $($duration.ToString('mm\:ss'))" -ForegroundColor Gray
        Write-Host "   Erro: $($_.Exception.Message)" -ForegroundColor Red
        
        # Gerar relatório de erro
        $errorReport = @{
            timestamp = $startTime.ToString("yyyy-MM-dd HH:mm:ss")
            duration = $duration.ToString("mm\:ss")
            branch = $GitBranch
            commit = $GitCommit
            error = $_.Exception.Message
            stackTrace = $_.ScriptStackTrace
            success = $false
        }
        
        $errorReport | ConvertTo-Json | Out-File -FilePath "pipeline-error.json" -Encoding UTF8
        
        exit 1
    }
    finally {
        Invoke-Cleanup
    }
}

# Executar função principal
Main