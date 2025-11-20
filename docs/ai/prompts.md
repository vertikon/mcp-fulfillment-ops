# Prompts de IA

Este documento descreve o sistema de prompts de IA do mcp-fulfillment-ops.

## Visão Geral

O sistema de prompts permite criar, gerenciar e otimizar prompts para diferentes modelos de IA e casos de uso.

## Tipos de Prompts

### 1. System Prompts

Prompts que definem o comportamento geral do modelo:

- **Role Definition**: Define o papel do assistente
- **Constraints**: Define limitações e regras
- **Output Format**: Define formato de saída esperado

### 2. User Prompts

Prompts enviados pelo usuário:

- **Queries**: Perguntas e solicitações
- **Instructions**: Instruções específicas
- **Context**: Contexto adicional

### 3. Template Prompts

Prompts parametrizados com variáveis:

- **Variables**: `{{variable_name}}`
- **Conditionals**: `{{#if condition}}...{{/if}}`
- **Loops**: `{{#each items}}...{{/each}}`

## Estrutura de Prompt

```yaml
prompt:
  name: "mcp-generator"
  type: "system"
  template: |
    You are an expert MCP generator.
    Generate MCP projects following these guidelines:
    - Use Clean Architecture
    - Follow Go best practices
    - Include proper error handling
    
    Context: {{context}}
    Requirements: {{requirements}}
  variables:
    - name: "context"
      type: "string"
      required: true
    - name: "requirements"
      type: "string"
      required: true
```

## Gerenciamento de Prompts

### Criar Prompt

```go
promptService.CreatePrompt(ctx, promptDefinition)
```

### Usar Prompt

```go
response, err := promptService.Execute(ctx, "mcp-generator", variables)
```

### Otimizar Prompt

```go
optimizedPrompt := promptService.Optimize(ctx, promptID, examples)
```

## Best Practices

1. **Clareza**: Seja claro e específico
2. **Contexto**: Forneça contexto suficiente
3. **Exemplos**: Inclua exemplos quando possível
4. **Iteração**: Teste e refine prompts
5. **Versionamento**: Mantenha versões dos prompts

## Referências

- [Exemplos de Prompts](../examples/ai_prompts.md)
- [AI Integration](./integration.md)
- [RAG](./rag.md)
- [Knowledge Management](./knowledge_management.md)

