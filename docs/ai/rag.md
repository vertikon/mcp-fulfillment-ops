# RAG (Retrieval-Augmented Generation)

Este documento descreve o sistema RAG (Retrieval-Augmented Generation) do mcp-fulfillment-ops.

## Visão Geral

O RAG permite que modelos de IA acessem e utilizem conhecimento específico através de recuperação de documentos relevantes e geração aumentada.

## Componentes do RAG

### 1. Knowledge Base

A base de conhecimento armazena documentos e seus embeddings para recuperação eficiente.

- **Documentos**: Textos, markdown, PDFs, etc.
- **Embeddings**: Representações vetoriais dos documentos
- **Metadata**: Informações adicionais sobre os documentos

### 2. Retrieval Engine

O motor de recuperação busca documentos relevantes baseado em queries:

- **Similarity Search**: Busca por similaridade de embeddings
- **Keyword Search**: Busca por palavras-chave
- **Hybrid Search**: Combinação de ambos

### 3. Generation Engine

O motor de geração usa os documentos recuperados para gerar respostas:

- **Context Injection**: Injeta contexto dos documentos recuperados
- **Prompt Engineering**: Constrói prompts otimizados
- **Response Generation**: Gera respostas usando LLMs

## Fluxo RAG

```
Query → Retrieval Engine → Relevant Documents → Generation Engine → Response
```

## Configuração

O RAG é configurado através de `config/ai/knowledge.yaml`:

```yaml
rag:
  enabled: true
  retrieval:
    method: "similarity"  # similarity, keyword, hybrid
    top_k: 5
    threshold: 0.7
  generation:
    model: "gpt-4"
    temperature: 0.7
    max_tokens: 1000
```

## Uso

### Criar Knowledge Base

```go
knowledgeService.CreateKnowledgeBase(ctx, "my-knowledge", documents)
```

### Query RAG

```go
response, err := ragService.Query(ctx, "What is MCP?", knowledgeID)
```

## Referências

- [Knowledge Management](./knowledge_management.md)
- [Memory Management](./memory_management.md)
- [AI Integration](./integration.md)
- [Guia RAG](../guides/ai_rag.md)

