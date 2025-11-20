# Script PowerShell para build e execução via Docker

Write-Host "🐳 Build e execução via Docker" -ForegroundColor Cyan
Write-Host ""

# Build da imagem
Write-Host "🔨 Construindo imagem Docker..." -ForegroundColor Yellow
docker build -t mcp-fulfillment-ops:latest .

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Imagem construída com sucesso" -ForegroundColor Green
} else {
    Write-Host "❌ Falha ao construir imagem" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "🚀 Para executar:" -ForegroundColor Cyan
Write-Host "   docker-compose up -d" -ForegroundColor White
Write-Host ""
Write-Host "Ou executar standalone:" -ForegroundColor Cyan
Write-Host "   docker run -p 8080:8080 \`" -ForegroundColor White
Write-Host "     -e DATABASE_URL=postgres://... \`" -ForegroundColor White
Write-Host "     -e NATS_URL=nats://... \`" -ForegroundColor White
Write-Host "     -e REDIS_URL=redis://... \`" -ForegroundColor White
Write-Host "     -e CORE_INVENTORY_URL=http://... \`" -ForegroundColor White
Write-Host "     mcp-fulfillment-ops:latest" -ForegroundColor White

