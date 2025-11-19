# Exemplo de RAG (Retrieval-Augmented Generation)

Este documento demonstra como usar o sistema RAG do MCP-HULK.

## Cenário

Criar uma base de conhecimento sobre MCP e fazer queries usando RAG.

## Passo 1: Criar Base de Conhecimento

```go
package main

import (
    "context"
    "github.com/vertikon/mcp-fulfillment-ops/internal/ai/knowledge"
)

func main() {
    ctx := context.Background()
    
    // Criar serviço de conhecimento
    knowledgeService := knowledge.NewKnowledgeService(...)
    
    // Documentos sobre MCP
    documents := []string{
        "MCP (Model Context Protocol) é um protocolo para criar projetos de IA...",
        "O MCP-HULK implementa o protocolo MCP usando Go...",
        "A arquitetura do MCP-HULK segue Clean Architecture...",
    }
    
    // Criar base de conhecimento
    knowledgeBase, err := knowledgeService.CreateKnowledgeBase(ctx, "mcp-knowledge", documents)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Knowledge base created: %s\n", knowledgeBase.ID())
}
```

## Passo 2: Fazer Query RAG

```go
package main

import (
    "context"
    "github.com/vertikon/mcp-fulfillment-ops/internal/ai/rag"
)

func main() {
    ctx := context.Background()
    
    // Criar serviço RAG
    ragService := rag.NewRAGService(...)
    
    // Query
    query := "Como funciona o protocolo MCP?"
    
    // Executar query RAG
    response, err := ragService.Query(ctx, query, knowledgeBaseID)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Response: %s\n", response.Text)
    fmt.Printf("Sources: %v\n", response.Sources)
}
```

## Passo 3: Usar com LLM

```go
package main

import (
    "context"
    "github.com/vertikon/mcp-fulfillment-ops/internal/ai/rag"
)

func main() {
    ctx := context.Background()
    
    // Criar serviço RAG com LLM
    ragService := rag.NewRAGServiceWithLLM(...)
    
    // Query com contexto
    query := "Explique a arquitetura do MCP-HULK"
    
    // Executar query RAG
    response, err := ragService.QueryWithContext(ctx, query, knowledgeBaseID, rag.QueryOptions{
        TopK: 5,
        Temperature: 0.7,
        MaxTokens: 1000,
    })
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Response: %s\n", response.Text)
    fmt.Printf("Relevant Documents: %d\n", len(response.Sources))
}
```

## Resultado Esperado

```
Response: O MCP-HULK implementa o protocolo MCP usando Go e segue Clean Architecture...
Relevant Documents: 3
```

## Referências

- [RAG Documentation](../ai/rag.md)
- [Knowledge Management](../ai/knowledge_management.md)
- [Guia RAG](../guides/ai_rag.md)

