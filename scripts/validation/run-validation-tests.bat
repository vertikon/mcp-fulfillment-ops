@echo off
echo ================================================
echo === EXECUTANDO TESTES DE VALIDAÇÃO - BLOCO-1 ===
echo.

echo 1. Verificando ambiente...
docker-compose -f docker-compose-integration.yml ps | findstr "Up"
if %ERRORLEVEL% NEQ "" (
    echo. ❌ ERRO: Serviços não estão no ar
    echo. Execute: docker-compose -f docker-compose-integration.yml up -d
    goto :end
)
echo. ✅ Serviços no ar

echo.
echo 2. Executando teste de integração...
cd ..\mcp-fulfillment-ops\scripts\validation
call .\test-integration-complete.bat

echo.
echo 3. Verificando resultado...
if %ERRORLEVEL% NEQ "" (
    echo. ✅ Teste de integração concluído com sucesso
) else (
    echo. ❌ ERRO: Falha no teste de integração
)

echo.
echo 4. Criando relatório...
cd ..\mcp-fulfillment-ops\scripts\validation
call .\generate-report.bat

echo.
echo ================================================
echo === TESTES DE VALIDAÇÃO CONCLUÍDOS ===
echo.

:end
pause