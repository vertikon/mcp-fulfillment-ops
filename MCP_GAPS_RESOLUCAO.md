# 🔧 Resolução de GAPs - Análise MCP

**Data:** 2025-11-21  
**Projeto:** mcp-fulfillment-ops  
**Total de Violações:** 72 (severidade média)

## 📊 Resumo Executivo

- ✅ **Servidor MCP:** Funcionando corretamente
- ⚠️ **Erros de Compilação:** 20+ erros críticos encontrados
- 🔧 **Correções Automáticas:** Disponíveis para ~30% dos problemas
- 📝 **Correções Manuais:** Necessárias para ~70% dos problemas

## 🔴 Erros Críticos de Compilação (Prioridade ALTA)

### 1. `cmd/tools-validator/main.go:60`
**Erro:** `unknown field StrictMode in struct literal`
**Solução:** Remover campo `StrictMode` ou atualizar struct `ConfigValidateRequest`

### 2. `internal/ai/knowledge/indexer_test.go`
**Erro:** `not enough arguments in call to NewIndexer`
**Solução:** Adicionar parâmetro `Embedder` em todas as chamadas de `NewIndexer`

### 3. `internal/core/crush/memory_optimizer.go:614`
**Erro:** `undefined: runtime.SetGCPercent`
**Solução:** Usar `debug.SetGCPercent()` do pacote `runtime/debug`

### 4. `internal/core/scheduler/scheduler.go:62`
**Erro:** `undefined: nats.ErrStreamNameExist`
**Solução:** Verificar versão do NATS e usar constante correta ou tratar erro de forma diferente

### 5. `internal/core/state/store.go`
**Erro:** Imports não usados (`logger`, `zap`)
**Solução:** Remover imports não utilizados

### 6. `internal/domain/services/ai_domain_service.go:33`
**Erro:** `context.documents undefined` (deveria ser `Documents()`)
**Solução:** Usar método `Documents()` ao invés de campo `documents`

### 7. `internal/infrastructure/compute/serverless/cloud_functions.go:14`
**Erro:** `undefined: FunctionConfig`
**Solução:** Definir tipo `FunctionConfig` ou importar do pacote correto

### 8. `internal/infrastructure/persistence/relational/postgres_knowledge_repository.go`
**Erro:** Variável `knowledge` não usada e tipo incorreto no return
**Solução:** Corrigir lógica de retorno e remover variável não usada

### 9. `internal/mcp/generators/generator_factory.go`
**Erro:** `req.Stack undefined`
**Solução:** Adicionar campo `Stack` ao tipo `GenerateRequest` ou remover referências

### 10. `internal/security/config/integration.go:215`
**Erro:** Campos `Resource`, `Action`, `Description` não existem em `PolicyRuleConfig`
**Solução:** Atualizar struct `PolicyRuleConfig` ou corrigir acesso aos campos

## 🟡 Correções Automáticas Disponíveis

### Imports Não Usados (6 casos)
- `internal/core/state/store.go` - remover `logger` e `zap`
- `internal/interfaces/http/handlers_outbound.go` - remover `fulfillment`
- `tools/deployers/hybrid_deployer.go` - remover `kubernetes`

### Variáveis Não Usadas (15+ casos)
- Prefixar com `_` ou remover completamente

### Problemas de Estilo (20+ casos)
- Executar `gofmt -w` e `goimports -w`

## 🟠 APIs Deprecated (2 casos)

### 1. `strings.Title` → `golang.org/x/text/cases`
**Arquivo:** `internal/ai/core/prompt_builder.go:119`
**Solução:** 
```go
import "golang.org/x/text/cases"
import "golang.org/x/text/language"

// Antes:
title := strings.Title(text)

// Depois:
caser := cases.Title(language.English)
title := caser.String(text)
```

### 2. Jaeger Exporter → OTLP
**Arquivo:** `internal/observability/tracing.go:8`
**Solução:** Migrar para `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp`

## 📋 Plano de Ação

### Fase 1: Correções Automáticas (Imediato)
1. ✅ Executar `goimports -w .` para remover imports não usados
2. ✅ Executar `gofmt -w .` para formatação
3. ⚠️ Corrigir variáveis não usadas (prefixar com `_`)

### Fase 2: Erros de Compilação (Crítico)
1. Corrigir chamadas de `NewIndexer` e `NewKnowledgeStore`
2. Corrigir `runtime.SetGCPercent` → `debug.SetGCPercent`
3. Corrigir `nats.ErrStreamNameExist`
4. Corrigir acesso a `context.Documents()`
5. Definir/corrigir `FunctionConfig`
6. Corrigir struct `PolicyRuleConfig`

### Fase 3: APIs Deprecated (Médio Prazo)
1. Migrar `strings.Title` para `golang.org/x/text/cases`
2. Migrar Jaeger exporter para OTLP

### Fase 4: Melhorias de Estilo (Baixa Prioridade)
1. Corrigir uso de `nil Context` → `context.TODO()`
2. Corrigir verificações desnecessárias de `nil`
3. Melhorar mensagens de erro (não capitalizadas)

## 🚀 Comandos de Correção Rápida

```bash
# 1. Remover imports não usados
goimports -w .

# 2. Formatar código
gofmt -w .

# 3. Verificar compilação
go build ./...

# 4. Executar testes
go test ./...
```

## 📝 Notas

- Todos os erros são de severidade **média** (não críticos para execução)
- A maioria são problemas de compilação que impedem build
- Alguns problemas podem ser resolvidos automaticamente
- APIs deprecated precisam de migração cuidadosa

