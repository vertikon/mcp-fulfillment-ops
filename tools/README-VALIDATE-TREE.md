# 🔍 Ferramenta de Validação de Árvore - mcp-fulfillment-ops

**Ferramenta:** `validate-tree`  
**Versão:** 1.0  
**Propósito:** Validar conformidade estrutural do projeto mcp-fulfillment-ops

---

## 📋 Descrição

A ferramenta `validate-tree` compara três camadas do projeto:

1. **Árvore Original** (`mcp-fulfillment-ops-ARVORE-FULL.md`) - Fonte única da verdade
2. **Árvore Comentada** (`ARVORE-ARQUIVOS-DIRETORIOS-COMENTADA.md`) - Documentação comentada
3. **Implementação Real** - Arquivos reais no sistema de arquivos

---

## 🚀 Instalação

```bash
# Compilar a ferramenta
go build -o bin/validate-tree ./tools/validate_tree.go

# Ou instalar globalmente
go install ./tools/validate_tree.go
```

---

## 💻 Uso

### Uso Básico

```bash
# Validação padrão (formato JSON)
./bin/validate-tree

# Validação com caminhos customizados
./bin/validate-tree \
  --original .cursor/mcp-fulfillment-ops-ARVORE-FULL.md \
  --commented .cursor/ARVORE-ARQUIVOS-DIRETORIOS-COMENTADA.md \
  --root .
```

### Formatos de Saída

```bash
# Formato JSON (padrão)
./bin/validate-tree --format json

# Formato Markdown
./bin/validate-tree --format markdown > relatorio.md

# Formato Texto
./bin/validate-tree --format text
```

### Modo Strict

```bash
# Falha se houver arquivos faltantes
./bin/validate-tree --strict

# Falha se compliance < 95%
./bin/validate-tree --strict --compliance-threshold 95
```

---

## 📊 Saída da Ferramenta

### Formato JSON

```json
{
  "summary": {
    "total_original_files": 430,
    "total_commented_files": 433,
    "total_implementation_files": 450,
    "common_files": 291,
    "missing_count": 0,
    "extra_count": 20,
    "compliance_percent": 97.4
  },
  "block_compliance": {
    "BLOCO-1": {
      "block": "BLOCO-1",
      "expected_files": 15,
      "found_files": 15,
      "missing_files": 0,
      "compliance_percent": 100.0,
      "status": "✅ Complete"
    }
  }
}
```

### Formato Markdown

```markdown
# Tree Validation Report

**Compliance:** 97.40%

## Summary

- Original Files: 430
- Commented Files: 433
- Implementation Files: 450
- Common Files: 291
- Missing Files: 0
- Extra Files: 20

## Block Compliance

| Block | Expected | Found | Missing | Compliance | Status |
|-------|----------|-------|---------|------------|--------|
| BLOCO-1 | 15 | 15 | 0 | 100.00% | ✅ Complete |
```

---

## 🔧 Integração com CI/CD

### GitHub Actions

```yaml
name: Validate Tree Structure

on:
  pull_request:
    branches: [main]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build validate-tree
        run: go build -o bin/validate-tree ./tools/validate_tree.go
      
      - name: Validate tree structure
        run: ./bin/validate-tree --strict --format markdown > validation-report.md
      
      - name: Upload report
        uses: actions/upload-artifact@v3
        with:
          name: validation-report
          path: validation-report.md
```

### GitLab CI

```yaml
validate_tree:
  stage: validate
  image: golang:1.21
  script:
    - go build -o bin/validate-tree ./tools/validate_tree.go
    - ./bin/validate-tree --strict --format json > validation-report.json
  artifacts:
    paths:
      - validation-report.json
    expire_in: 1 week
```

---

## 📝 Flags Disponíveis

| Flag | Descrição | Padrão |
|------|-----------|--------|
| `--original`, `-o` | Caminho para árvore original | `.cursor/mcp-fulfillment-ops-ARVORE-FULL.md` |
| `--commented`, `-c` | Caminho para árvore comentada | `.cursor/ARVORE-ARQUIVOS-DIRETORIOS-COMENTADA.md` |
| `--root`, `-r` | Diretório raiz do projeto | `.` |
| `--format`, `-f` | Formato de saída (json/markdown/text) | `json` |
| `--strict`, `-s` | Modo strict (falha em não conformidade) | `false` |

---

## 🎯 Casos de Uso

### 1. Validação Pré-Commit

```bash
#!/bin/bash
# .git/hooks/pre-commit

./bin/validate-tree --strict
if [ $? -ne 0 ]; then
  echo "❌ Tree validation failed. Please fix structural issues."
  exit 1
fi
```

### 2. Auditoria Periódica

```bash
#!/bin/bash
# scripts/audit_tree.sh

DATE=$(date +%Y-%m-%d)
./bin/validate-tree --format markdown > ".cursor/audits/tree-validation-${DATE}.md"
```

### 3. Relatório de Conformidade

```bash
# Gerar relatório completo
./bin/validate-tree --format markdown > compliance-report.md

# Enviar para equipe
mail -s "Tree Compliance Report" team@example.com < compliance-report.md
```

---

## 🔍 Interpretação dos Resultados

### Compliance ≥ 95%

✅ **Aprovado** - Estrutura conforme. Pode prosseguir.

### Compliance 90-95%

⚠️ **Revisar** - Algumas divergências menores. Revisar arquivos extras e documentar.

### Compliance < 90%

❌ **Rejeitar** - Estrutura não conforme. Bloquear merge até correção.

---

## 🐛 Troubleshooting

### Erro: "failed to load original tree"

**Causa:** Arquivo de árvore não encontrado.

**Solução:**
```bash
# Verificar se o arquivo existe
ls -la .cursor/mcp-fulfillment-ops-ARVORE-FULL.md

# Especificar caminho correto
./bin/validate-tree --original /caminho/correto/ARVORE-FULL.md
```

### Erro: "compliance below threshold"

**Causa:** Compliance abaixo do threshold (padrão 95%).

**Solução:**
```bash
# Verificar relatório detalhado
./bin/validate-tree --format markdown > report.md
cat report.md

# Corrigir arquivos faltantes ou ajustar threshold
```

### Performance Lenta

**Causa:** Projeto muito grande ou muitos arquivos.

**Solução:**
```bash
# Excluir diretórios grandes do scan
# Editar validate_tree.go para adicionar mais ignoredDirs
```

---

## 📚 Documentação Relacionada

- `.cursor/MAPA-DIFERENCAS-mcp-fulfillment-ops.md` - Mapa completo de diferenças
- `.cursor/CHECKLIST-AUDITORIA.md` - Checklist de auditoria
- `.cursor/RELATORIO-EXECUTIVO-CONSOLIDADO.md` - Relatório executivo

---

## 🤝 Contribuindo

Para melhorar a ferramenta:

1. Adicionar novos formatos de saída
2. Melhorar detecção de blocos
3. Adicionar validações específicas
4. Otimizar performance

---

**Última Atualização:** 2025-01-27  
**Versão:** 1.0

