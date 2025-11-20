# Configuração do mcp-fulfillment-ops

Esta pasta contém todos os arquivos de configuração do mcp-fulfillment-ops.

## Estrutura

```
config/
├── config.yaml              # Configuração principal
├── features.yaml            # Feature flags
├── environments/           # Configurações por ambiente
│   ├── dev.yaml
│   ├── staging.yaml
│   ├── prod.yaml
│   └── test.yaml
└── README.md               # Este arquivo
```

## Arquivo .env

Para configurar variáveis de ambiente sensíveis (senhas, chaves de API, etc.), crie um arquivo `.env` na **raiz do projeto** (não nesta pasta).

### Como Criar o Arquivo .env

1. Na raiz do projeto, crie um arquivo chamado `.env`
2. Adicione as variáveis necessárias usando o prefixo `HULK_`
3. Consulte a documentação completa em: `docs/guides/env_variables_reference.md`

### Exemplo Mínimo

```bash
# .env (na raiz do projeto)

# Ambiente
HULK_ENV=dev

# Banco de Dados
HULK_DATABASE_PASSWORD=sua_senha_aqui

# IA
HULK_AI_API_KEY=sua_chave_api_aqui
```

### Variáveis Importantes

- `HULK_ENV`: Define qual arquivo de ambiente será carregado (dev, staging, prod, test)
- `HULK_DATABASE_PASSWORD`: Senha do banco de dados (obrigatória em produção)
- `HULK_AI_API_KEY`: Chave de API do provedor de IA (obrigatória)

### Segurança

⚠️ **IMPORTANTE:** O arquivo `.env` **NÃO** deve ser commitado no Git. Ele já está no `.gitignore`.

Para ver todas as variáveis disponíveis, consulte: `docs/guides/env_variables_reference.md`

## Ordem de Precedência

As configurações são aplicadas na seguinte ordem (maior precedência primeiro):

1. Variáveis de ambiente (`HULK_*`)
2. Arquivo de ambiente (`environments/{env}.yaml`)
3. `features.yaml`
4. `config.yaml`
5. Defaults (hardcoded)

## Documentação Completa

- [Referência de Variáveis de Ambiente](../../docs/guides/env_variables_reference.md)
- [Guia de Configuração](../../docs/guides/configuration.md)
- [Blueprint BLOCO-12](../../.cursor/BLOCOS/BLOCO-12-BLUEPRINT.md)

