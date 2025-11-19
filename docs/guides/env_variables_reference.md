# Referência de Variáveis de Ambiente - MCP-HULK

Este documento lista todas as variáveis de ambiente disponíveis no MCP-HULK.

**IMPORTANTE:** Copie este conteúdo para um arquivo `.env` na raiz do projeto e ajuste os valores conforme necessário.

## Instruções de Uso

1. Crie um arquivo `.env` na raiz do projeto
2. Copie as variáveis necessárias deste documento
3. Ajuste os valores conforme seu ambiente
4. **NUNCA** commite o arquivo `.env` no Git (já está no .gitignore)

Todas as variáveis de ambiente devem usar o prefixo `HULK_`. O sistema converte automaticamente pontos (.) em underscores (_).  
Exemplo: `server.port` → `HULK_SERVER_PORT`

---

## Ambiente

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `HULK_ENV` | Define qual arquivo de ambiente será carregado (dev, staging, prod, test) | `dev` | Não |

---

## Servidor HTTP

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `HULK_SERVER_PORT` | Porta do servidor HTTP | `8080` | Não |
| `HULK_SERVER_HOST` | Host do servidor HTTP | `0.0.0.0` | Não |
| `HULK_SERVER_READ_TIMEOUT` | Timeout de leitura HTTP | `30s` | Não |
| `HULK_SERVER_WRITE_TIMEOUT` | Timeout de escrita HTTP | `30s` | Não |

---

## Banco de Dados (PostgreSQL)

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `HULK_DATABASE_URL` | URL completa de conexão (sobrescreve outras configs) | - | Não |
| `HULK_DATABASE_HOST` | Host do banco de dados | `localhost` | Não |
| `HULK_DATABASE_PORT` | Porta do banco de dados | `5432` | Não |
| `HULK_DATABASE_USER` | Usuário do banco de dados | `postgres` | Não |
| `HULK_DATABASE_PASSWORD` | Senha do banco de dados | - | **Sim (produção)** |
| `HULK_DATABASE_DATABASE` | Nome do banco de dados | `mcp_hulk` | Não |
| `HULK_DATABASE_SSL_MODE` | Modo SSL (disable, require, verify-ca, verify-full) | `disable` | Não |
| `HULK_DATABASE_MAX_CONNS` | Máximo de conexões no pool | `25` | Não |
| `HULK_DATABASE_MIN_CONNS` | Mínimo de conexões no pool | `5` | Não |

**Exemplo de URL completa:**
```
HULK_DATABASE_URL=postgres://user:password@host:5432/database?sslmode=disable
```

---

## Provedor de IA

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `HULK_AI_PROVIDER` | Provedor de IA (openai, gemini, glm) | `glm` | Não |
| `HULK_AI_MODEL` | Modelo padrão de IA | `glm-4.6-z.ai` | Não |
| `HULK_AI_API_KEY` | Chave de API do provedor de IA | - | **Sim** |
| `HULK_AI_ENDPOINT` | Endpoint da API de IA | `https://api.z.ai/v1` | Não |
| `HULK_AI_TEMPERATURE` | Temperature (criatividade) - 0.0 a 1.0 | `0.3` | Não |
| `HULK_AI_MAX_TOKENS` | Máximo de tokens por requisição | `4000` | Não |
| `HULK_AI_TIMEOUT` | Timeout de requisição de IA | `60s` | Não |

---

## Caminhos do Sistema

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `HULK_PATHS_TEMPLATES` | Caminho para diretório de templates | `./templates` | Não |
| `HULK_PATHS_OUTPUT` | Caminho para diretório de saída | `./output` | Não |
| `HULK_PATHS_DATA` | Caminho para diretório de dados | `./data` | Não |
| `HULK_PATHS_CACHE` | Caminho para diretório de cache | `./data/cache` | Não |

---

## Engine (Processamento)

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `HULK_ENGINE_WORKERS` | Número de workers (auto = NumCPU*2, ou número específico) | `auto` | Não |
| `HULK_ENGINE_QUEUE_SIZE` | Tamanho da fila de processamento | `2000` | Não |
| `HULK_ENGINE_TIMEOUT` | Timeout do engine | `20s` | Não |

---

## Cache

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `HULK_CACHE_L1_SIZE` | Tamanho do cache L1 | `5000` | Não |
| `HULK_CACHE_L2_TTL` | TTL do cache L2 | `1h` | Não |
| `HULK_CACHE_L3_PATH` | Caminho do cache L3 | `data/cache` | Não |

---

## NATS (Messaging)

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `HULK_NATS_URLS` | URLs do NATS (separadas por vírgula se múltiplas) | `nats://localhost:4222` | Não |
| `HULK_NATS_USER` | Usuário do NATS | - | Não |
| `HULK_NATS_PASS` | Senha do NATS | - | Não |

**Exemplo de múltiplas URLs:**
```
HULK_NATS_URLS=nats://server1:4222,nats://server2:4222
```

---

## Logging

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `HULK_LOGGING_LEVEL` | Nível de log (debug, info, warn, error) | `info` | Não |
| `HULK_LOGGING_FORMAT` | Formato de log (json, console) | `json` | Não |

---

## Telemetria (Observabilidade)

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `HULK_TELEMETRY_TRACING_ENABLED` | Habilitar tracing | `true` | Não |
| `HULK_TELEMETRY_TRACING_EXPORTER` | Exporter de tracing (jaeger, otlp) | `jaeger` | Não |
| `HULK_TELEMETRY_TRACING_ENDPOINT` | Endpoint do tracing | `http://localhost:4318/v1/traces` | Não |
| `HULK_TELEMETRY_METRICS_ENABLED` | Habilitar métricas | `true` | Não |

---

## MCP Registry

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `HULK_MCP_REGISTRY_STORAGE_PATH` | Caminho de armazenamento do registry | `./registry` | Não |
| `HULK_MCP_REGISTRY_AUTO_SAVE` | Auto-salvar registry | `true` | Não |
| `HULK_MCP_REGISTRY_SAVE_INTERVAL` | Intervalo de salvamento automático (segundos) | `300` | Não |
| `HULK_MCP_REGISTRY_MAX_PROJECTS` | Máximo de projetos no registry | `1000` | Não |
| `HULK_MCP_REGISTRY_MAX_TEMPLATES` | Máximo de templates no registry | `100` | Não |
| `HULK_MCP_REGISTRY_ENABLE_METRICS` | Habilitar métricas do registry | `true` | Não |
| `HULK_MCP_REGISTRY_CACHE_ENABLED` | Habilitar cache do registry | `true` | Não |
| `HULK_MCP_REGISTRY_CACHE_TTL` | TTL do cache do registry (segundos) | `3600` | Não |

---

## MCP Server

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `HULK_MCP_SERVER_NAME` | Nome do servidor MCP | `mcp-fulfillment-ops` | Não |
| `HULK_MCP_SERVER_VERSION` | Versão do servidor MCP | `1.0.0` | Não |
| `HULK_MCP_SERVER_PROTOCOL` | Versão do protocolo MCP | `2024-11-05` | Não |
| `HULK_MCP_SERVER_TRANSPORT` | Transporte MCP (stdio, sse) | `stdio` | Não |
| `HULK_MCP_SERVER_PORT` | Porta do servidor MCP | `3000` | Não |
| `HULK_MCP_SERVER_HOST` | Host do servidor MCP | `localhost` | Não |
| `HULK_MCP_SERVER_MAX_WORKERS` | Máximo de workers do servidor MCP | `10` | Não |
| `HULK_MCP_SERVER_TIMEOUT` | Timeout do servidor MCP (segundos) | `30` | Não |
| `HULK_MCP_SERVER_ENABLE_AUTH` | Habilitar autenticação MCP | `false` | Não |
| `HULK_MCP_SERVER_AUTH_TOKEN` | Token de autenticação MCP | - | Não |

---

## Feature Flags

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `HULK_FEATURES_EXTERNAL_GPU` | Habilitar GPU externa (RunPod) | `false` | Não |
| `HULK_FEATURES_AUDIT_LOGGING` | Habilitar logging de auditoria detalhado | `false` | Não |
| `HULK_FEATURES_BETA_GENERATORS` | Habilitar geradores beta | `false` | Não |

---

## Exemplo de Arquivo .env Mínimo

```bash
# Ambiente
HULK_ENV=dev

# Banco de Dados
HULK_DATABASE_HOST=localhost
HULK_DATABASE_PORT=5432
HULK_DATABASE_USER=postgres
HULK_DATABASE_PASSWORD=sua_senha_aqui
HULK_DATABASE_DATABASE=mcp_hulk

# IA
HULK_AI_PROVIDER=glm
HULK_AI_API_KEY=sua_chave_api_aqui
HULK_AI_MODEL=glm-4.6-z.ai

# Logging
HULK_LOGGING_LEVEL=info
HULK_LOGGING_FORMAT=json
```

---

## Ordem de Precedência

As configurações são aplicadas na seguinte ordem (maior precedência primeiro):

1. **Variáveis de ambiente** (`HULK_*`)
2. **Arquivo de ambiente** (`environments/{env}.yaml`)
3. **features.yaml**
4. **config.yaml**
5. **Defaults** (hardcoded no código)

---

## Formato de Valores

- **Durações**: Use formato Go (`30s`, `1h`, `5m`)
- **Booleanos**: `true` ou `false` (case-insensitive)
- **Arrays**: Separados por vírgula (ex: `HULK_NATS_URLS=url1,url2`)

---

## Segurança

⚠️ **IMPORTANTE:**

1. **NUNCA** commite o arquivo `.env` no Git
2. Use um gerenciador de segredos (Vault, AWS Secrets Manager) em produção
3. Rotacione chaves regularmente
4. Em produção, use `HULK_DATABASE_SSL_MODE=require` (ou superior)

---

## Variáveis Obrigatórias em Produção

- `HULK_DATABASE_PASSWORD`
- `HULK_AI_API_KEY`
- `HULK_DATABASE_SSL_MODE=require` (ou superior)

---

## Referências

- [Guia de Configuração](../guides/configuration.md)
- [Guia de Deployment](../guides/deployment.md)
- [Blueprint BLOCO-12](../../.cursor/BLOCOS/BLOCO-12-BLUEPRINT.md)

