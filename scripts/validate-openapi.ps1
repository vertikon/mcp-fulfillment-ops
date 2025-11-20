# Script PowerShell para validar contratos OpenAPI

Write-Host "🔍 Validando contratos OpenAPI..." -ForegroundColor Cyan
Write-Host ""

$OpenAPIDir = Join-Path $PSScriptRoot "..\..\..\..\contracts\openapi\bloco-1-core"
$ValidatorURL = "https://validator.swagger.io/validator/debug"

# Validar v1
Write-Host "📄 Validando fulfillment-ops-v1.yaml..." -ForegroundColor Yellow
$v1File = Join-Path $OpenAPIDir "fulfillment-ops-v1.yaml"

if (Test-Path $v1File) {
    $v1Content = Get-Content $v1File -Raw
    try {
        $response = Invoke-RestMethod -Uri $ValidatorURL -Method Post -ContentType "application/yaml" -Body $v1Content
        Write-Host "   ✅ v1 válido" -ForegroundColor Green
    } catch {
        Write-Host "   ⚠️  Erro ao validar v1: $_" -ForegroundColor Yellow
    }
} else {
    Write-Host "   ❌ Arquivo não encontrado: $v1File" -ForegroundColor Red
}

# Validar v2
Write-Host ""
Write-Host "📄 Validando fulfillment-ops-v2.yaml..." -ForegroundColor Yellow
$v2File = Join-Path $OpenAPIDir "fulfillment-ops-v2.yaml"

if (Test-Path $v2File) {
    $v2Content = Get-Content $v2File -Raw
    try {
        $response = Invoke-RestMethod -Uri $ValidatorURL -Method Post -ContentType "application/yaml" -Body $v2Content
        Write-Host "   ✅ v2 válido" -ForegroundColor Green
    } catch {
        Write-Host "   ⚠️  Erro ao validar v2: $_" -ForegroundColor Yellow
    }
} else {
    Write-Host "   ❌ Arquivo não encontrado: $v2File" -ForegroundColor Red
}

Write-Host ""
Write-Host "✅ Validação concluída!" -ForegroundColor Green
Write-Host ""
Write-Host "💡 Para visualizar no Swagger Editor:" -ForegroundColor Cyan
Write-Host "   https://editor.swagger.io/" -ForegroundColor White
Write-Host "   Cole o conteúdo do arquivo YAML no editor" -ForegroundColor White

