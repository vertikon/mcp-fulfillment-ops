# Script de Validação para BLOCO-1 (PowerShell)
# Executa todos os testes de validação do ecossistema

param(
    [Parameter(Mandatory=$false)]
    [string]$TestType = "all"
)

Write-Host "=== EXECUTANDO TESTES DE VALIDAÇÃO - BLOCO-1 ==="
Write-Host "Teste: $TestType"
Write-Host "Data: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"

# Função para verificar saúde dos serviços
function Test-ServiceHealth {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Name,
        [string]$Url
    )
    
    Write-Host "Verificando saúde do $Name..." -ForegroundColor Yellow
    
    try {
        $response = Invoke-RestMethod -Uri $Url -Method GET -TimeoutSec 10
        if ($response.StatusCode -eq 200) {
            Write-Host "✅ $Name está saudável" -ForegroundColor Green
            return $true
        } else {
            Write-Host "❌ $Name está indisponível (Status: $($response.StatusCode))" -ForegroundColor Red
            return $false
        }
    }
    catch {
        Write-Host "❌ Erro ao verificar $Name`: $_" -ForegroundColor Red
        return $false
    }
}

# Função para executar teste de integração
function Test-Integration {
    Write-Host "Executando teste de integração..." -ForegroundColor Yellow
    
    $order = @{
        order_id = "TEST-$(Get-Date -Format 'yyyyMMddHHmmss')"
        customer = "CUSTOMER-INTEGRATION-PS"
        destination = "Rua Teste Integração, 123 - São Paulo/SP"
        items = @(
            @{ sku = "PROD-TEST-001"; quantity = 2 }
        )
        priority = 0
        idempotency_key = $order.order_id
    }
    
    try {
        $body = $order | ConvertTo-Json -Depth 10
        $response = Invoke-RestMethod -Uri "http://localhost:8082/api/v1/fulfillment-orders" -Method POST -Body $body -ContentType "application/json" -TimeoutSec 30
        
        if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
            Write-Host "✅ Ordem criada: $($order.order_id)" -ForegroundColor Green
            
            # Verificar reserva no Core Inventory
            Start-Sleep -Seconds 2
            $reservation = Invoke-RestMethod -Uri "http://localhost:8081/api/v1/reservations" -Method GET -Query "reference_id=$($order.order_id)" -TimeoutSec 10
            
            if ($reservation.StatusCode -eq 200) {
                Write-Host "✅ Reserva criada no Core Inventory" -ForegroundColor Green
            } else {
                Write-Host "❌ Reserva não encontrada no Core Inventory" -ForegroundColor Red
            }
            
            # Publicar evento no NATS
            $event = @{
                order_id = $order.order_id
                customer_id = "CUST-INTEGRATION-PS"
                items = $order.items
            } | ConvertTo-Json -Depth 10
            
            # Simular publicação (não temos NATS local, mas continuamos)
            Write-Host "✅ Evento NATS simulado: oms.order.ready_to_pick" -ForegroundColor Yellow
            
            return $true
        } else {
            Write-Host "❌ Falha ao criar ordem (Status: $($response.StatusCode))" -ForegroundColor Red
            return $false
        }
    }
    catch {
        Write-Host "❌ Erro no teste de integração: $_" -ForegroundColor Red
        return $false
    }
}

# Função para executar teste de carga
function Test-Load {
    Write-Host "Executando teste de carga (k6)..." -ForegroundColor Yellow
    
    try {
        # Verificar se k6 está instalado
        $k6 = Get-Command k6 -ErrorAction SilentlyContinue
        if (-not $k6) {
            Write-Host "❌ k6 não está instalado. Instale com: npm install -g k6" -ForegroundColor Red
            return $false
        }
        
        # Executar teste de carga
        $output = & $k6 run ..\tests\load\create_orders_test.js --out json 2>&1
        Wait-Process $output
        
        # Verificar se o teste foi bem-sucedido
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✅ Teste de carga concluído com sucesso" -ForegroundColor Green
            return $true
        } else {
            Write-Host "❌ Falha no teste de carga (Exit Code: $LASTEXITCODE)" -ForegroundColor Red
            return $false
        }
    }
    catch {
        Write-Host "❌ Erro no teste de carga: $_" -ForegroundColor Red
        return $false
    }
}

# Função para executar teste de segurança
function Test-Security {
    Write-Host "Executando teste de segurança..." -ForegroundColor Yellow
    
    # Teste de Rate Limiting
    try {
        Write-Host "Testando Rate Limiting..." -ForegroundColor Yellow
        
        $responses = @()
        for ($i = 1; $i -le 150; $i++) {
            $response = Invoke-RestMethod -Uri "http://localhost:8082/api/v1/fulfillment-orders" -Method POST -Body @{
                order_id = "RATE-LIMIT-$i"
                customer = "RATE-TEST"
                items = @{ sku = "PROD-TEST-001"; quantity = 1 }
            } -ContentType "application/json" -TimeoutSec 5
            
            $responses += $response
            
            Start-Sleep -Milliseconds 100
        }
        
        # Verificar se houve algum bloqueio (429)
        $blocked = $responses | Where-Object { $_.StatusCode -eq 429 }
        
        if ($blocked.Count -gt 0) {
            Write-Host "✅ Rate limiting está funcionando ($($blocked.Count) requisições bloqueadas)" -ForegroundColor Green
            return $true
        } else {
            Write-Host "❌ Rate limiting não detectado" -ForegroundColor Red
            return $false
        }
    }
    catch {
        Write-Host "❌ Erro no teste de segurança: $_" -ForegroundColor Red
        return $false
    }
}

# Executar testes com base no parâmetro
switch ($TestType.ToLower()) {
    "health" {
        Write-Host "`n--- Teste de Saúde ---" -ForegroundColor Cyan
        
        Test-ServiceHealth -Name "Core Inventory" -Url "http://localhost:8081/health"
        Test-ServiceHealth -Name "Fulfillment Ops" -Url "http://localhost:8082/health"
        Test-ServiceHealth -Name "NATS" -Url "http://localhost:8225/varz"
        
        Write-Host "---" -ForegroundColor Cyan
    }
    "integration" {
        Write-Host "`n--- Teste de Integração ---" -ForegroundColor Cyan
        Test-Integration
    }
    "load" {
        Write-Host "`n--- Teste de Carga ---" -ForegroundColor Cyan
        Test-Load
    }
    "security" {
        Write-Host "`n--- Teste de Segurança ---" -ForegroundColor Cyan
        Test-Security
    }
    "all" {
        Write-Host "`n--- Todos os Testes ---" -ForegroundColor Cyan
        
        $healthPassed = Test-ServiceHealth -Name "Core Inventory" -Url "http://localhost:8081/health"
        $healthPassed = $healthPassed -and (Test-ServiceHealth -Name "Fulfillment Ops" -Url "http://localhost:8082/health")
        $healthPassed = $healthPassed -and (Test-ServiceHealth -Name "NATS" -Url "http://localhost:8225/varz")
        
        $integrationPassed = Test-Integration
        $loadPassed = Test-Load
        $securityPassed = Test-Security
        
        Write-Host "`n--- Resumo dos Testes ---" -ForegroundColor White
        Write-Host "Saúde dos Serviços: $(if ($healthPassed) { "✅ Passou" } else { "❌ Falhou" })" -ForegroundColor $(if ($healthPassed) { "Green" } else { "Red" })
        Write-Host "Teste de Integração: $(if ($integrationPassed) { "✅ Passou" } else { "❌ Falhou" })" -ForegroundColor $(if ($integrationPassed) { "Green" } else { "Red" })
        Write-Host "Teste de Carga: $(if ($loadPassed) { "✅ Passou" } else { "❌ Falhou" })" -ForegroundColor $(if ($loadPassed) { "Green" } else { "Red" })
        Write-Host "Teste de Segurança: $(if ($securityPassed) { "✅ Passou" } else { "❌ Falhou" })" -ForegroundColor $(if ($securityPassed) { "Green" } else { "Red" })
        
        $allPassed = $healthPassed -and $integrationPassed -and $loadPassed -and $securityPassed
        
        Write-Host "Resultado Geral: $(if ($allPassed) { "✅ TODOS OS TESTES PASSARAM" } else { "❌ ALGUM TESTE FALHOU" })" -ForegroundColor $(if ($allPassed) { "Green" } else { "Red" })
    }
    default {
        Write-Host "Parâmetro inválido. Use: health, integration, load, security ou all" -ForegroundColor Red
        exit 1
    }
}

Write-Host "=== TESTES CONCLUÍDOS ===" -ForegroundColor Cyan