# Referência de Variáveis de Ambiente - mcp-fulfillment-ops

Este documento lista todas as variáveis de ambiente disponíveis no mcp-fulfillment-ops.

**IMPORTANTE:** Copie este conteúdo para um arquivo `.env` na raiz do projeto e ajuste os valores conforme necessário.

## Instruções de Uso

1. Crie um arquivo `.env` na raiz do projeto
2. Copie as variáveis necessárias deste documento
3. Ajuste os valores conforme seu ambiente
4. **NUNCA** commite o arquivo `.env` no Git (já está no .gitignore)

Todas as variáveis de ambiente devem usar o prefixo `FULFILLMENT_`. O sistema converte automaticamente pontos (.) em underscores (_).  
Exemplo: `server.port` → `FULFILLMENT_SERVER_PORT`

---

## Ambiente

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `FULFILLMENT_ENV` | Define qual arquivo de ambiente será carregado (dev, staging, prod, test) | `dev` | Não |

---

## Servidor HTTP

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `FULFILLMENT_SERVER_PORT` | Porta do servidor HTTP | `8080` | Não |
| `FULFILLMENT_SERVER_HOST` | Host do servidor HTTP | `0.0.0.0` | Não |
| `FULFILLMENT_SERVER_READ_TIMEOUT` | Timeout de leitura HTTP | `30s` | Não |
| `FULFILLMENT_SERVER_WRITE_TIMEOUT` | Timeout de escrita HTTP | `30s` | Não |

---

## Banco de Dados (PostgreSQL)

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `FULFILLMENT_DATABASE_URL` | URL completa de conexão (sobrescreve outras configs) | - | Não |
| `FULFILLMENT_DATABASE_HOST` | Host do banco de dados | `localhost` | Não |
| `FULFILLMENT_DATABASE_PORT` | Porta do banco de dados | `5432` | Não |
| `FULFILLMENT_DATABASE_USER` | Usuário do banco de dados | `postgres` | Não |
| `FULFILLMENT_DATABASE_PASSWORD` | Senha do banco de dados | - | **Sim (produção)** |
| `FULFILLMENT_DATABASE_DATABASE` | Nome do banco de dados | `fulfillment` | Não |
| `FULFILLMENT_DATABASE_SSL_MODE` | Modo SSL (disable, require, verify-ca, verify-full) | `disable` | Não |
| `FULFILLMENT_DATABASE_MAX_CONNS` | Máximo de conexões no pool | `25` | Não |
| `FULFILLMENT_DATABASE_MIN_CONNS` | Mínimo de conexões no pool | `5` | Não |

**Exemplo de URL completa:**
```
FULFILLMENT_DATABASE_URL=postgres://user:password@host:5432/database?sslmode=disable
```

---

## Provedor de IA

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `FULFILLMENT_AI_PROVIDER` | Provedor de IA (openai, gemini, glm) | `glm` | Não |
| `FULFILLMENT_AI_MODEL` | Modelo padrão de IA | `glm-4.6-z.ai` | Não |
| `FULFILLMENT_AI_API_KEY` | Chave de API do provedor de IA | - | **Sim** |
| `FULFILLMENT_AI_ENDPOINT` | Endpoint da API de IA | `https://api.z.ai/v1` | Não |
| `FULFILLMENT_AI_TEMPERATURE` | Temperature (criatividade) - 0.0 a 1.0 | `0.3` | Não |
| `FULFILLMENT_AI_MAX_TOKENS` | Máximo de tokens por requisição | `4000` | Não |
| `FULFILLMENT_AI_TIMEOUT` | Timeout de requisição de IA | `60s` | Não |

---

## Caminhos do Sistema

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `FULFILLMENT_PATHS_TEMPLATES` | Caminho para diretório de templates | `./templates` | Não |
| `FULFILLMENT_PATHS_OUTPUT` | Caminho para diretório de saída | `./output` | Não |
| `FULFILLMENT_PATHS_DATA` | Caminho para diretório de dados | `./data` | Não |
| `FULFILLMENT_PATHS_CACHE` | Caminho para diretório de cache | `./data/cache` | Não |

---

## Engine (Processamento)

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `FULFILLMENT_ENGINE_WORKERS` | Número de workers (auto = NumCPU*2, ou número específico) | `auto` | Não |
| `FULFILLMENT_ENGINE_QUEUE_SIZE` | Tamanho da fila de processamento | `2000` | Não |
| `FULFILLMENT_ENGINE_TIMEOUT` | Timeout do engine | `20s` | Não |

---

## Cache

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `FULFILLMENT_CACHE_L1_SIZE` | Tamanho do cache L1 | `5000` | Não |
| `FULFILLMENT_CACHE_L2_TTL` | TTL do cache L2 | `1h` | Não |
| `FULFILLMENT_CACHE_L3_PATH` | Caminho do cache L3 | `data/cache` | Não |

---

## NATS (Messaging)

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `FULFILLMENT_NATS_URLS` | URLs do NATS (separadas por vírgula se múltiplas) | `nats://localhost:4222` | Não |
| `FULFILLMENT_NATS_USER` | Usuário do NATS | - | Não |
| `FULFILLMENT_NATS_PASS` | Senha do NATS | - | Não |

**Exemplo de múltiplas URLs:**
```
FULFILLMENT_NATS_URLS=nats://server1:4222,nats://server2:4222
```

---

## Logging

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `FULFILLMENT_LOGGING_LEVEL` | Nível de log (debug, info, warn, error) | `info` | Não |
| `FULFILLMENT_LOGGING_FORMAT` | Formato de log (json, console) | `json` | Não |

---

## Telemetria (Observabilidade)

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `FULFILLMENT_TELEMETRY_TRACING_ENABLED` | Habilitar tracing | `true` | Não |
| `FULFILLMENT_TELEMETRY_TRACING_EXPORTER` | Exporter de tracing (jaeger, otlp) | `jaeger` | Não |
| `FULFILLMENT_TELEMETRY_TRACING_ENDPOINT` | Endpoint do tracing | `http://localhost:4318/v1/traces` | Não |
| `FULFILLMENT_TELEMETRY_METRICS_ENABLED` | Habilitar métricas | `true` | Não |

---

## MCP Registry

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `FULFILLMENT_MCP_REGISTRY_STORAGE_PATH` | Caminho de armazenamento do registry | `./registry` | Não |
| `FULFILLMENT_MCP_REGISTRY_AUTO_SAVE` | Auto-salvar registry | `true` | Não |
| `FULFILLMENT_MCP_REGISTRY_SAVE_INTERVAL` | Intervalo de salvamento automático (segundos) | `300` | Não |
| `FULFILLMENT_MCP_REGISTRY_MAX_PROJECTS` | Máximo de projetos no registry | `1000` | Não |
| `FULFILLMENT_MCP_REGISTRY_MAX_TEMPLATES` | Máximo de templates no registry | `100` | Não |
| `FULFILLMENT_MCP_REGISTRY_ENABLE_METRICS` | Habilitar métricas do registry | `true` | Não |
| `FULFILLMENT_MCP_REGISTRY_CACHE_ENABLED` | Habilitar cache do registry | `true` | Não |
| `FULFILLMENT_MCP_REGISTRY_CACHE_TTL` | TTL do cache do registry (segundos) | `3600` | Não |

---

## MCP Server

| Variável | Descrição | Default | Obrigatório |
|----------|-----------|---------|-------------|
| `