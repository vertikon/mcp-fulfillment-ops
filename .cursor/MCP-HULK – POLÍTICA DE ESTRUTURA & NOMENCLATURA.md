# mcp-fulfillment-ops – POLÍTICA DE ESTRUTURA & NOMENCLATURA

STATUS: **CONGELADO v1.0**

Este documento define as **regras obrigatórias** de estrutura de diretórios, nomes de arquivos e pontos de extensão do projeto **mcp-fulfillment-ops**.

A árvore oficial do projeto está em:

- `mcp-fulfillment-ops-ARVORE-FULL.md`  
- Local: `E:\vertikon\.templates\mcp-fulfillment-ops\`  

Essa árvore é a **fonte única da verdade** da estrutura de arquivos do template. :contentReference[oaicite:1]{index=1}  

Qualquer desvio dessas regras torna o template **inválido**.

---

## 1. Fonte Única da Verdade

1. A árvore definida em `mcp-fulfillment-ops-ARVORE-FULL.md` é a **estrutura oficial** do projeto.
2. **É proibido**:
   - criar arquivos/diretórios fora dos caminhos previstos;
   - renomear arquivos/diretórios sem atualizar a árvore oficial;
   - mover arquivos/diretórios para outros blocos sem atualização formal.
3. Todo novo artefato deve:
   - ser previsto na árvore;
   - ter comentário de função padronizado (quando aplicável).

---

## 2. Regras de Nomenclatura

### 2.1. Diretórios

- Diretórios seguem nomes **sem espaços** e em **inglês**, minúsculos, com underscore se necessário:
  - `internal/ai/core`
  - `internal/infrastructure/compute`
  - `templates/mcp-go-premium`
- Pastas de camada são fixas:
  - `internal/`, `pkg/`, `templates/`, `scripts/`, `config/`, `docs/`.

**É proibido** criar novas pastas de topo (root) sem alterar a árvore oficial.

### 2.2. Arquivos Go (`.go`)

- Nome em **snake_case**, sempre descrevendo o papel:
  - `mcp_http_handler.go`, `model_registry.go`, `state_snapshot.go`.
- Handlers devem deixar claro o tipo:
  - HTTP: `*_http_handler.go`
  - gRPC: `*_grpc_server.go`
  - Events: `*_events_handler.go`
- Repositórios:
  - Interfaces: `*_repository.go`
  - Implementações específicas: `postgres_*_repository.go`, `*_client.go` em `infrastructure`.

**Proibido:**
- Nomes genéricos como `utils.go`, `helpers.go` em camadas críticas.
- Ter dois arquivos com mesmo nome em contextos ambíguos (ex.: dois `knowledge_store.go` com propósito diferente).
  - Exceção só se a árvore oficial documentar claramente o contexto (ex.: `internal/ai/knowledge` vs outro pacote dedicado) – hoje, o padrão é **evitar colisão de nomes**.

### 2.3. Scripts (`.sh`)

- Todos em `scripts/**`, nunca soltos em outros diretórios.
- Prefixos por categoria:
  - `setup_*.sh`, `deploy_*.sh`, `generate_*.sh`, `validate_*.sh`, `optimize_*.sh`, `migrate_*.sh`.

---

## 3. É PROIBIDO

1. Criar arquivos/diretórios **não previstos** em `mcp-fulfillment-ops-ARVORE-FULL.md`.
2. Renomear arquivos/diretórios sem:
   - atualizar a árvore oficial;
   - atualizar `mcp-fulfillment-ops-STRUCTURE-POLICY.md` se a mudança for estrutural.
3. Escrever código Go diretamente:
   - em `cmd/` (além dos `main.go` previstos);
   - em `templates/**` fora dos placeholders declarados.
4. Criar “atalhos” como:
   - `internal/utils/`, `internal/common/`, `internal/shared/` **não previstos** na árvore.
5. Editar arquivos de template gerado diretamente no `templates/` para “um caso específico de cliente”.
   - Customizações específicas devem ser feitas via:
     - projeto gerado,
     - ou ferramenta `mcp-init/` (CLI de customização).

---

## 4. Como ESTENDER o Projeto (Processo Oficial)

Para adicionar **qualquer nova funcionalidade estrutural**:

1. **Propor alteração na árvore**:
   - Atualizar `mcp-fulfillment-ops-ARVORE-FULL.md` com:
     - novo caminho,
     - novo arquivo,
     - comentário de função.
2. **Revisar impacto**:
   - Camada (`internal`, `pkg`, `templates`, `scripts`, `config`, `docs`);
   - Bloco afetado (1 a 14).
3. **Atualizar documentação**:
   - Ajustar arquivos em `docs/` quando necessário
     (ex.: `docs/architecture/blueprint.md`, `docs/guides/development.md`).
4. **Somente após esses passos**:
   - criar o arquivo real no diretório correspondente;
   - implementar o código.

**Qualquer arquivo criado “direto no código” sem atualização da árvore é considerado inválido.**

---

## 5. Customização: Somente via Ferramentas Oficiais

Para adaptar o template ao contexto de um cliente/produto:

- Usar **`cmd/mcp-init/`**:
  - responsável por varrer a árvore,
  - aplicar regras de substituição,
  - gerar variações sem violar o padrão de estrutura.

Não é permitido:

- “hacking manual” na estrutura base do `mcp-fulfillment-ops` dentro de `E:\vertikon\.templates\mcp-fulfillment-ops\`.
- renomear pastas como `internal/ai`, `internal/mcp`, `internal/infrastructure`, `templates/base`, etc.

---

## 6. Validação Automática (Recomendado)

Fica definido que:

- `tools/template_validator.go` e `tools/code_validator.go`
- junto com scripts:
  - `scripts/validation/validate_template.sh`
  - `scripts/validation/validate_mcp.sh`
  - `scripts/validation/validate_infrastructure.sh`

devem ser estendidos para:

1. Verificar se a árvore real do filesystem **bate** com `mcp-fulfillment-ops-ARVORE-FULL.md`.
2. Bloquear PRs/commits que:
   - criem arquivos não previstos;
   - removam/renomeiem arquivos sem atualização da árvore.

---

## 7. Governança de Versão da Árvore

- Alterações neste arquivo (`mcp-fulfillment-ops-STRUCTURE-POLICY.md`) e em `mcp-fulfillment-ops-ARVORE-FULL.md`:
  - devem ser versionadas com tag clara (ex.: `ARVORE_v1`, `ARVORE_v2`);
  - exigem review estrutural (arquitetura) antes de merge.

Enquanto este documento estiver com status **CONGELADO v1.0**, nenhuma nova pasta/arquivo pode ser adicionada fora do processo descrito.

---
