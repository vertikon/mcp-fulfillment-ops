# ⚡ GUIA RÁPIDO DE VALIDAÇÃO - mcp-fulfillment-ops

**Para uso imediato da ferramenta de validação**

---

## 🚀 Início Rápido

### 1. Compilar a Ferramenta

```bash
# Compilar
go build -o bin/validate-tree ./tools/validate_tree.go

# Ou usar o script
chmod +x scripts/validation/validate_project_structure.sh
./scripts/validation/validate_project_structure.sh
```

### 2. Executar Validação Básica

```bash
# Validação simples (formato JSON)
./bin/validate-tree

# Validação com relatório Markdown
./bin/validate-tree --format markdown > relatorio.md

# Validação em modo strict (falha se não conforme)
./bin/validate-tree --strict
```

### 3. Verificar Resultados

```bash
# Ver compliance geral
./bin/validate-tree --format text | grep Compliance

# Ver compliance por BLOCO
./bin/validate-tree --format markdown | grep "BLOCO-"
```

---

## 📋 Comandos Úteis

### Validação Completa

```bash
./bin/validate-tree \
  --original .cursor/mcp-fulfillment-ops-ARVORE-FULL.md \
  --commented .cursor/ARVORE-ARQUIVOS-DIRETORIOS-COMENTADA.md \
  --root . \
  --format markdown \
  --strict
```

### Gerar Relatório para Auditoria

```bash
DATE=$(date +%Y-%m-%d)
./bin/validate-tree --format markdown > ".cursor/audits/validation-${DATE}.md"
```

### Validação Pré-Commit

```bash
# Adicionar ao .git/hooks/pre-commit
#!/bin/bash
./bin/validate-tree --strict
if [ $? -ne 0 ]; then
  echo "❌ Validação de estrutura falhou"
  exit 1
fi
```

---

## 🔍 Interpretação Rápida

### Compliance ≥ 95%
✅ **OK** - Pode prosseguir

### Compliance 90-95%
⚠️ **Revisar** - Verificar arquivos extras

### Compliance < 90%
❌ **Bloquear** - Corrigir estrutura

---

## 🛠️ Troubleshooting Rápido

### Erro: "failed to load original tree"
```bash
# Verificar se arquivo existe
ls -la .cursor/mcp-fulfillment-ops-ARVORE-FULL.md

# Especificar caminho correto
./bin/validate-tree --original /caminho/correto/ARVORE-FULL.md
```

### Erro: "compliance below threshold"
```bash
# Ver relatório detalhado
./bin/validate-tree --format markdown > report.md
cat report.md

# Verificar arquivos faltantes
./bin/validate-tree --format json | jq '.missing[]'
```

### Performance Lenta
```bash
# Excluir diretórios grandes (editar validate_tree.go)
# Adicionar mais ignoredDirs
```

---

## 📚 Documentação Completa

- **Ferramenta:** `tools/README-VALIDATE-TREE.md`
- **Checklist:** `.cursor/CHECKLIST-AUDITORIA.md`
- **Mapa de Diferenças:** `.cursor/MAPA-DIFERENCAS-mcp-fulfillment-ops.md`

---

**Última Atualização:** 2025-01-27

