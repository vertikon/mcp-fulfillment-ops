# Script PowerShell para configurar variáveis de ambiente

$EnvFile = ".env"
$EnvExample = ".env.example"

Write-Host "⚙️  Configurando variáveis de ambiente..." -ForegroundColor Cyan
Write-Host ""

# Criar .env.example se não existir
if (-not (Test-Path $EnvExample)) {
    @"
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
"@ | Out-File -FilePath $EnvExample -Encoding UTF8
    Write-Host "✅ Criado $EnvExample" -ForegroundColor Green
}

# Criar .env se não existir
if (-not (Test-Path $EnvFile)) {
    Copy-Item $EnvExample $EnvFile
    Write-Host "✅ Criado $EnvFile a partir de $EnvExample" -ForegroundColor Green
    Write-Host ""
    Write-Host "⚠️  IMPORTANTE: Edite $EnvFile com os valores corretos para seu ambiente" -ForegroundColor Yellow
} else {
    Write-Host "ℹ️  $EnvFile já existe, não foi sobrescrito" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "📋 Variáveis de ambiente configuradas!" -ForegroundColor Green
Write-Host ""
Write-Host "Para carregar as variáveis no PowerShell:" -ForegroundColor Cyan
Write-Host "  Get-Content $EnvFile | ForEach-Object { if (`$_ -match '^([^#].*?)=(.*)$') { [Environment]::SetEnvironmentVariable(`$matches[1], `$matches[2], 'Process') } }" -ForegroundColor White

