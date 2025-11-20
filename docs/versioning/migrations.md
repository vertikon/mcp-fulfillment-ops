# Migrations (Migrações)

Este documento descreve o sistema de migrações do mcp-fulfillment-ops para versionamento de conhecimento, modelos e dados.

## Visão Geral

O sistema de migrações permite mover versões de conhecimento, modelos e dados entre ambientes e versões.

## Tipos de Migração

### 1. Knowledge Migration

Migração de bases de conhecimento:

- **Documents**: Migração de documentos
- **Embeddings**: Migração de embeddings
- **Metadata**: Migração de metadados

### 2. Model Migration

Migração de modelos de IA:

- **Model Files**: Arquivos do modelo
- **Weights**: Pesos do modelo
- **Configurations**: Configurações do modelo

### 3. Data Migration

Migração de dados e schemas:

- **Schema Changes**: Mudanças de schema
- **Data Transformation**: Transformação de dados
- **Data Validation**: Validação de dados

## Estrutura de Migração

### Migration Object

```json
{
  "id": "string (UUID)",
  "type": "knowledge|model|data",
  "source_version": "string",
  "target_version": "string",
  "status": "pending|running|completed|failed",
  "steps": [
    {
      "id": "string",
      "name": "string",
      "type": "string",
      "status": "pending|running|completed|failed",
      "error": "string (optional)"
    }
  ],
  "created_at": "timestamp",
  "started_at": "timestamp (optional)",
  "completed_at": "timestamp (optional)",
  "error": "string (optional)"
}
```

## Execução de Migração

### Criar Migração

```go
migration := migrationEngine.CreateMigration(ctx, sourceVersion, targetVersion, steps)
```

### Executar Migração

```go
err := migrationEngine.Execute(ctx, migrationID)
```

### Rollback Migração

```go
err := migrationEngine.Rollback(ctx, migrationID)
```

## Validação

### Pré-Migração

- Validação de versões de origem e destino
- Validação de compatibilidade
- Validação de pré-requisitos

### Pós-Migração

- Validação de integridade
- Validação de consistência
- Validação de completude

## Scripts de Migração

Os scripts de migração estão em `scripts/migration/`:

- `migrate_knowledge.sh` - Migração de conhecimento
- `migrate_models.sh` - Migração de modelos
- `migrate_data.sh` - Migração de dados

## Configuração

```yaml
versioning:
  migrations:
    enabled: true
    engine: "in-memory"  # in-memory, database
    validation:
      pre_migration: true
      post_migration: true
    rollback:
      enabled: true
      max_attempts: 3
```

## Referências

- [Knowledge Versioning](./knowledge_versioning.md)
- [Model Versioning](./model_versioning.md)
- [Data Versioning](./data_versioning.md)
- [Workflow](./workflow.md)

