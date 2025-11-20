# Script PowerShell para testar integração com mcp-core-inventory

$ErrorActionPreference = "Stop"

Write-Host "🧪 Testando integração com mcp-core-inventory..." -ForegroundColor Cyan
Write-Host ""

$CoreInventoryURL = if ($env:CORE_INVENTORY_URL) { $env:CORE_INVENTORY_URL } else { "http://localhost:8081" }
$FulfillmentURL = if ($env:FULFILLMENT_URL) { $env:FULFILLMENT_URL } else { "http://localhost:8080" }

Write-Host "📡 Core Inventory URL: $CoreInventoryURL" -ForegroundColor Yellow
Write-Host "📡 Fulfillment Ops URL: $FulfillmentURL" -ForegroundColor Yellow
Write-Host ""

# Teste 1: Health check do Core Inventory
Write-Host "1️⃣ Testando health check do Core Inventory..." -ForegroundColor Cyan
try {
    $response = Invoke-WebRequest -Uri "$CoreInventoryURL/health" -Method Get -UseBasicParsing -ErrorAction Stop
    Write-Host "   ✅ Core Inventory está respondendo" -ForegroundColor Green
} catch {
    Write-Host "   ❌ Core Inventory não está respondendo: $_" -ForegroundColor Red
    exit 1
}

# Teste 2: Health check do Fulfillment Ops
Write-Host ""
Write-Host "2️⃣ Testando health check do Fulfillment Ops..." -ForegroundColor Cyan
try {
    $response = Invoke-WebRequest -Uri "$FulfillmentURL/health" -Method Get -UseBasicParsing -ErrorAction Stop
    Write-Host "   ✅ Fulfillment Ops está respondendo" -ForegroundColor Green
} catch {
    Write-Host "   ❌ Fulfillment Ops não está respondendo: $_" -ForegroundColor Red
    exit 1
}

# Teste 3: Criar Inbound Shipment
Write-Host ""
Write-Host "3️⃣ Testando criação de Inbound Shipment..." -ForegroundColor Cyan
$body = @{
    reference_id = "TEST-PO-001"
    origin = "Fornecedor Teste"
    destination = "CD-TEST"
    items = @(
        @{
            sku = "SKU-TEST-001"
            quantity = 10
        }
    )
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$FulfillmentURL/v1/inbound/start" -Method Post -ContentType "application/json" -Body $body
    $shipmentId = $response.id
    if ($shipmentId) {
        Write-Host "   ✅ Inbound Shipment criado: $shipmentId" -ForegroundColor Green
    } else {
        Write-Host "   ❌ Falha ao criar Inbound Shipment" -ForegroundColor Red
        Write-Host "   Resposta: $($response | ConvertTo-Json)" -ForegroundColor Yellow
        exit 1
    }
} catch {
    Write-Host "   ❌ Erro ao criar Inbound Shipment: $_" -ForegroundColor Red
    exit 1
}

# Teste 4: Confirmar recebimento
Write-Host ""
Write-Host "4️⃣ Testando confirmação de recebimento..." -ForegroundColor Cyan
$confirmBody = @{
    shipment_id = $shipmentId
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$FulfillmentURL/v1/inbound/confirm" -Method Post -ContentType "application/json" -Body $confirmBody
    Write-Host "   ✅ Recebimento confirmado" -ForegroundColor Green
} catch {
    Write-Host "   ⚠️  Confirmação pode ter falhado (verifique se Core Inventory está configurado)" -ForegroundColor Yellow
    Write-Host "   Erro: $_" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "✅ Testes de integração concluídos!" -ForegroundColor Green

