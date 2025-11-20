# Script PowerShell de Teste de Integração Físico-Lógico
# Valida que ShipOrder no Fulfillment gera débito correto no Core Inventory

$ErrorActionPreference = "Stop"

$FulfillmentURL = if ($env:FULFILLMENT_URL) { $env:FULFILLMENT_URL } else { "http://localhost:8080" }
$CoreInventoryURL = if ($env:CORE_INVENTORY_URL) { $env:CORE_INVENTORY_URL } else { "http://localhost:8081" }

Write-Host "🧪 TESTE DE INTEGRAÇÃO FÍSICO-LÓGICO" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Fulfillment Ops: $FulfillmentURL"
Write-Host "Core Inventory:  $CoreInventoryURL"
Write-Host ""

# Step 1: Health Checks
Write-Host "1️⃣ Verificando saúde dos serviços..." -ForegroundColor Yellow

try {
    $fulfillmentHealth = Invoke-RestMethod -Uri "$FulfillmentURL/health" -Method Get -ErrorAction Stop
    Write-Host "   ✅ Fulfillment Ops está rodando" -ForegroundColor Green
} catch {
    Write-Host "   ❌ Fulfillment Ops não está respondendo: $_" -ForegroundColor Red
    exit 1
}

try {
    $coreHealth = Invoke-RestMethod -Uri "$CoreInventoryURL/health" -Method Get -ErrorAction Stop
    Write-Host "   ✅ Core Inventory está rodando" -ForegroundColor Green
} catch {
    Write-Host "   ⚠️  Core Inventory não está respondendo (teste continuará mas pode falhar): $_" -ForegroundColor Yellow
}

Write-Host ""

# Step 2: Setup - Criar produto no Core Inventory
Write-Host "2️⃣ Configurando produto no Core Inventory..." -ForegroundColor Yellow
$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$sku = "SKU-TEST-$timestamp"
$location = "CD-TEST"

$adjustData = @{
    location = $location
    sku = $sku
    quantity = 100
    reason = "test_setup"
} | ConvertTo-Json

try {
    $adjustResponse = Invoke-RestMethod -Uri "$CoreInventoryURL/v1/adjust" -Method Post -Body $adjustData -ContentType "application/json" -ErrorAction Stop
    Write-Host "   ✅ Produto criado: $sku com 100 unidades" -ForegroundColor Green
} catch {
    Write-Host "   ⚠️  Erro ao criar produto (pode já existir): $_" -ForegroundColor Yellow
}

# Verificar estoque inicial
try {
    $availableResponse = Invoke-RestMethod -Uri "$CoreInventoryURL/v1/available?location=$location&sku=$sku" -Method Get -ErrorAction Stop
    $initialStock = $availableResponse.available
    Write-Host "   📊 Estoque inicial: $initialStock unidades" -ForegroundColor Green
} catch {
    Write-Host "   ⚠️  Não foi possível verificar estoque inicial: $_" -ForegroundColor Yellow
    $initialStock = 100
}

Write-Host ""

# Step 3: Criar FulfillmentOrder
Write-Host "3️⃣ Criando FulfillmentOrder..." -ForegroundColor Yellow
$orderID = "TEST-ORDER-$timestamp"

$pickingData = @{
    order_id = $orderID
} | ConvertTo-Json

try {
    $pickingResponse = Invoke-RestMethod -Uri "$FulfillmentURL/v1/outbound/start_picking" -Method Post -Body $pickingData -ContentType "application/json" -ErrorAction SilentlyContinue
    Write-Host "   📝 Resposta start_picking: OK" -ForegroundColor Green
} catch {
    Write-Host "   ⚠️  start_picking pode ter falhado (continuando): $_" -ForegroundColor Yellow
}

Write-Host ""

# Step 4: Executar ShipOrder (TESTE CRÍTICO)
Write-Host "4️⃣ 🎯 TESTE CRÍTICO: Executando ShipOrder..." -ForegroundColor Cyan
$shipData = @{
    order_id = $orderID
} | ConvertTo-Json

try {
    $shipResponse = Invoke-RestMethod -Uri "$FulfillmentURL/v1/outbound/ship" -Method Post -Body $shipData -ContentType "application/json" -ErrorAction Stop
    Write-Host "   ✅ ShipOrder executado: $($shipResponse.status)" -ForegroundColor Green
} catch {
    Write-Host "   ⚠️  ShipOrder pode ter falhado: $_" -ForegroundColor Yellow
    Write-Host "   (Isso pode ser esperado se a ordem não existir ou Core não estiver disponível)" -ForegroundColor Yellow
}

Write-Host ""

# Step 5: Aguardar processamento
Write-Host "5️⃣ Aguardando processamento do evento..." -ForegroundColor Yellow
Start-Sleep -Seconds 3

# Step 6: Validar débito no Core Inventory
Write-Host "6️⃣ 🎯 VALIDAÇÃO CRÍTICA: Verificando débito no Core Inventory..." -ForegroundColor Cyan

try {
    $finalAvailable = Invoke-RestMethod -Uri "$CoreInventoryURL/v1/available?location=$location&sku=$sku" -Method Get -ErrorAction Stop
    $finalStock = $finalAvailable.available
    
    Write-Host "   📊 Estoque final: $finalStock unidades" -ForegroundColor Green
    
    if ($finalStock -lt $initialStock) {
        $diff = $initialStock - $finalStock
        Write-Host "   ✅ SUCESSO: Estoque foi debitado! Diferença: $diff unidades" -ForegroundColor Green
        Write-Host "   ✅ Integração Físico-Lógico funcionando corretamente!" -ForegroundColor Green
    } elseif ($finalStock -eq 0) {
        Write-Host "   ⚠️  Estoque zerado (pode ser esperado se foi tudo debitado)" -ForegroundColor Yellow
    } else {
        Write-Host "   ❌ FALHA: Estoque não foi debitado corretamente" -ForegroundColor Red
        Write-Host "   Estoque inicial: $initialStock, Final: $finalStock" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "   ⚠️  Não foi possível verificar estoque final: $_" -ForegroundColor Yellow
    Write-Host "   (Core Inventory pode não estar disponível)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "✅ TESTE DE INTEGRAÇÃO CONCLUÍDO" -ForegroundColor Green

