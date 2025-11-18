# 🌳 ÁRVORE DE ARQUIVOS E DIRETÓRIOS COMENTADA - MCP-HULK

**Data de Geração:** 2025-01-27  
**Versão:** 1.0  
**Projeto:** MCP-HULK (Model Context Protocol - Hulk)

---

## 📋 SUMÁRIO

Este documento apresenta a estrutura completa de arquivos e diretórios do projeto MCP-HULK com comentários explicativos sobre cada componente. A estrutura segue os princípios de **Clean Architecture** e está organizada em **14 blocos principais**.

---

## 🗂️ ESTRUTURA COMPLETA COMENTADA

```
mcp-hulk/                                    # Raiz do projeto MCP-HULK
│
├── 📁 cmd/                                  # BLOCO-1: Application Entry Points
│   │                                        # Contém todos os pontos de entrada da aplicação (main.go)
│   │
│   ├── 📄 main.go                           # Servidor HTTP principal - ponto de entrada padrão
│   │                                        # Inicia servidor HTTP, configura rotas, inicializa serviços
│   │
│   ├── 📁 mcp-cli/                          # CLI para operações MCP
│   │   └── 📄 main.go                       # Interface CLI para operações MCP (criar, listar, validar)
│   │
│   ├── 📁 mcp-server/                       # Servidor do protocolo MCP
│   │   └── 📄 main.go                       # Servidor que implementa o protocolo MCP (JSON-RPC 2.0)
│   │
│   ├── 📁 mcp-init/                         # Ferramenta de customização/inicialização
│   │   ├── 📄 main.go                       # CLI para inicializar e customizar projetos MCP
│   │   │                                    # Função: Ponto de entrada da CLI de customização
│   │   │                                    # Comandos: --config, --path
│   │   │
│   │   └── 📁 internal/                     # Lógica interna da ferramenta (privado)
│   │       ├── 📁 config/                   # Configurações de regras de substituição
│   │       │   └── 📄 config.go             # Função: LoadConfig, Define mapeamentos e regras de transformação
│   │       │                                # Config: Estrutura de configuração com mapeamentos e exclusões
│   │       │
│   │       ├── 📁 processor/                # Núcleo do processamento de arquivos
│   │       │   └── 📄 processor.go         # Função: NewProcessor, Process, registerHandlers
│   │       │                                # Processor: Orquestra o walk pela árvore e delega aos handlers
│   │       │                                # processDirectory, processFile: Processa diretórios e arquivos
│   │       │
│   │       └── 📁 handlers/                 # Implementações específicas para cada tipo de arquivo
│   │           ├── 📄 handler.go            # Função: Interface Handler que define o contrato
│   │           ├── 📄 go_file.go            # Função: Process - Handler para arquivos .go (foco em imports)
│   │           ├── 📄 go_mod.go             # Função: Process - Handler para go.mod (reescrita segura)
│   │           ├── 📄 yaml_file.go          # Função: Process - Handler para arquivos .yaml/.yml
│   │           ├── 📄 text_file.go          # Função: Process - Handler genérico para .md, .sh, etc.
│   │           └── 📄 directory.go          # Função: Process - Handler para renomear diretórios/arquivos
│   │
│   ├── 📁 thor/                             # CLI principal Thor
│   │   └── 📄 main.go                       # CLI principal com comandos de gerenciamento
│   │
│   ├── 📁 tools-generator/                  # Executável CLI para ferramentas de geração
│   │   └── 📄 main.go                       # Expõe tools/generators via CLI (mcp, template, config, code)
│   │
│   ├── 📁 tools-validator/                  # Executável CLI para ferramentas de validação
│   │   └── 📄 main.go                       # Expõe tools/validators via CLI (mcp, template, config, code)
│   │
│   └── 📁 tools-deployer/                   # Executável CLI para ferramentas de deploy
│       └── 📄 main.go                       # Expõe tools/deployers via CLI (kubernetes, docker, serverless)
│
├── 📁 internal/                             # Código privado da aplicação (não exportado)
│   │                                        # Segue Clean Architecture com camadas bem definidas
│   │
│   ├── 📁 core/                             # BLOCO-1: Core Platform
│   │   │                                    # Motor de performance, configuração, cache, scheduler
│   │   │
│   │   ├── 📁 cache/                        # Sistema de cache multi-nível (L1/L2/L3)
│   │   │   │                                # L1: In-memory, L2: Distributed, L3: Persistent
│   │   │   ├── 📄 multi_level_cache.go      # Interface Cache e implementação MultiLevelCache (L1/L2/L3)
│   │   │   │                                # Função: Get, Set, Delete, Clear, Stats
│   │   │   │                                # L1Cache: Cache em memória com sync.Map
│   │   │   ├── 📄 cache_warmer.go           # Warmer para aquecimento de cache
│   │   │   │                                # Função: WarmUp, WarmUpFunc
│   │   │   ├── 📄 cache_invalidation.go     # Sistema de invalidação de cache
│   │   │   │                                # Função: Invalidate, InvalidatePattern, InvalidateAll
│   │   │   │                                # KeyTracker: Rastreamento de chaves para invalidação por padrão
│   │   │   │                                # TTL invalidation: Limpeza periódica de entradas expiradas
│   │   │   └── 📄 multi_level_cache_test.go # Testes unitários do cache
│   │   │
│   │   ├── 📁 config/                       # Sistema de configuração centralizado
│   │   │   │                                # Carrega configs de YAML, ENV, defaults (ordem de precedência)
│   │   │   ├── 📄 config.go                 # Estruturas de configuração e Loader
│   │   │   │                                # Função: NewLoader, Load, loadFeatures, loadEnvironmentConfig
│   │   │   │                                # Config: Server, Database, AI, Engine, Cache, NATS, Logging, Telemetry, MCP
│   │   │   │                                # setDefaults: Define valores padrão do sistema
│   │   │   ├── 📄 validation.go             # Validador de configurações
│   │   │   │                                # Função: Validate, validateServer, validateEngine, validateCache, validateNATS, validateLogging
│   │   │   ├── 📄 environment.go           # Gerenciador de ambiente
│   │   │   │                                # Função: NewEnvironmentManager, GetEnvironment, IsDevelopment, IsProduction, IsStaging, IsTest
│   │   │   └── 📄 config_test.go            # Testes unitários de configuração
│   │   │
│   │   ├── 📁 engine/                       # Motor de execução de alta performance
│   │   │   │                                # Worker pools, circuit breakers, otimizações
│   │   │   ├── 📄 execution_engine.go       # Motor principal de execução
│   │   │   │                                # Função: NewExecutionEngine, Start, Stop, Submit, Schedule, ScheduleInterval, Stats
│   │   │   │                                # ExecutionEngine: Orquestra WorkerPool e TaskScheduler
│   │   │   ├── 📄 worker_pool.go           # Pool de workers para processamento paralelo
│   │   │   │                                # Função: NewWorkerPool, Start, Stop, Submit, Stats
│   │   │   │                                # Task: Interface para tarefas executáveis
│   │   │   │                                # WorkerPool: Gerencia workers com retry, timeout, estatísticas
│   │   │   ├── 📄 task_scheduler.go        # Agendador de tarefas
│   │   │   │                                # Função: NewTaskScheduler, Start, Stop, Schedule, ScheduleInterval, Cancel
│   │   │   │                                # TaskScheduler: Gerencia tarefas agendadas e recorrentes
│   │   │   ├── 📄 circuit_breaker.go       # Circuit breaker para resiliência
│   │   │   │                                # Função: NewCircuitBreaker, Execute, State, Stats
│   │   │   │                                # CircuitBreaker: Estados Closed/Open/HalfOpen com recuperação automática
│   │   │   ├── 📄 execution_engine_test.go  # Testes unitários do execution engine
│   │   │   ├── 📄 worker_pool_test.go      # Testes unitários do worker pool
│   │   │   ├── 📄 task_scheduler_test.go   # Testes unitários do task scheduler
│   │   │   └── 📄 circuit_breaker_test.go  # Testes unitários do circuit breaker
│   │   │
│   │   ├── 📁 metrics/                      # Métricas do sistema
│   │   │   │                                # Prometheus metrics, contadores, gauges
│   │   │   ├── 📄 performance_monitor.go   # Monitor de performance do sistema
│   │   │   │                                # Função: NewPerformanceMonitor, Start, Stop, GetMetrics, GetCPUUsage, IsHealthy
│   │   │   │                                # PerformanceMonitor: Monitora CPU, memória, goroutines, GC
│   │   │   ├── 📄 resource_tracker.go      # Rastreador de recursos do sistema
│   │   │   │                                # Função: NewResourceTracker, Start, Stop, GetStats, IsHealthy, SetLimit
│   │   │   │                                # ResourceTracker: Rastreia uso de CPU, memória, disco com limites e alertas
│   │   │   └── 📄 alerting.go               # Sistema de alertas
│   │   │       │                            # Função: NewAlertManager, Start, Stop, AddHandler, GetAlerts, GetStats
│   │   │       │                            # AlertManager: Gerencia regras de alerta e notificações
│   │   │       │                            # AlertRule: Define condições e severidade de alertas
│   │   │       │                            # LogHandler: Handler de alertas que registra em logs
│   │   │
│   │   ├── 📁 scheduler/                    # Agendador de tarefas com NATS JetStream
│   │   │   └── 📄 scheduler.go              # Agendamento de tarefas assíncronas com NATS
│   │   │       │                            # Função: NewScheduler, InitializeStreams, PublishTick, SubscribeToTicks
│   │   │       │                            # Scheduler: Gerencia streams NATS JetStream para tarefas agendadas
│   │   │       │                            # TickEvent: Evento de tick do scheduler
│   │   │
│   │   ├── 📁 state/                        # Estado do core
│   │   │   ├── 📄 store.go                  # Store de estado persistente usando BadgerDB
│   │   │   │                                # Função: NewStore, Close, Get, Set, Delete, GetJSON, SetJSON
│   │   │   │                                # Store: Armazenamento key-value persistente com TTL
│   │   │   └── 📄 distributed_store.go      # Store distribuído de estado
│   │   │       │                            # Função: NewDistributedStore, Sync, GetSnapshot, RestoreSnapshot
│   │   │       │                            # DistributedStore: Sincronização de estado entre instâncias
│   │   │
│   │   ├── 📁 transformer/                  # Transformadores de dados (GLM-4.6)
│   │   │   │                                # Arquitetura Transformer para processamento de linguagem
│   │   │   ├── 📄 transformer.go            # Implementação do Transformer
│   │   │   │                                # Função: NewTransformer, Forward, Encode, Decode
│   │   │   ├── 📄 attention.go              # Mecanismo de atenção multi-cabeça
│   │   │   │                                # Função: MultiHeadAttention, ScaledDotProductAttention
│   │   │   ├── 📄 feedforward.go            # Redes feed-forward
│   │   │   │                                # Função: FeedForward, GELU activation
│   │   │   ├── 📄 embeddings.go            # Camada de embeddings
│   │   │   │                                # Função: Embedding, TokenEmbedding, PositionEmbedding
│   │   │   ├── 📄 positional_encoding.go   # Codificação posicional
│   │   │   │                                # Função: PositionalEncoding, SinusoidalEncoding
│   │   │   ├── 📄 inference_engine.go      # Motor de inferência
│   │   │   │                                # Função: NewInferenceEngine, Generate, BeamSearch, Sample
│   │   │   ├── 📄 transformer_test.go      # Testes unitários do transformer
│   │   │   └── 📄 inference_engine_test.go # Testes unitários do inference engine
│   │   │
│   │   ├── 📁 events/                       # Eventos do core com NATS JetStream
│   │   │   └── 📄 nats_events.go            # Sistema de eventos do core usando NATS
│   │   │       │                            # Função: NewEventPublisher, PublishTaskCreated, PublishTaskCompleted
│   │   │       │                            # Função: PublishTaskFailed, PublishRuntimeHealth
│   │   │       │                            # EventPublisher: Publica eventos para NATS JetStream
│   │   │       │                            # TaskEvent: Evento de tarefa (created, completed, failed)
│   │   │       │                            # HealthEvent: Evento de saúde do runtime
│   │   │
│   │   └── 📁 crush/                        # CRUSH - Parallel Processing Optimizations
│   │       │                                # Otimizações de processamento paralelo para GLM-4.6
│   │       ├── 📄 parallel_processor.go     # Processador paralelo
│   │       │                                # Função: NewParallelProcessor, Process, ProcessBatch
│   │       │                                # WorkerPool: Pool de workers paralelos com load balancing
│   │       │                                # AutoScaler: Escalamento automático de workers
│   │       ├── 📄 batch_processor.go        # Processador em batch
│   │       │                                # Função: NewBatchProcessor, ProcessBatch, Flush
│   │       │                                # BatchProcessor: Agrupa tarefas em batches para processamento eficiente
│   │       ├── 📄 memory_optimizer.go        # Otimizador de memória
│   │       │                                # Função: OptimizeMemory, Compact, Evict
│   │       │                                # MemoryOptimizer: Reduz uso de memória através de técnicas de compactação
│   │       ├── 📄 optimizer.go              # Otimizador de performance geral
│   │       │                                # Função: Optimize, Analyze, Recommend
│   │       │                                # Optimizer: Analisa e otimiza performance geral do sistema
│   │       └── 📄 optimizer_test.go         # Testes unitários do optimizer
│   │
│   ├── 📁 domain/                           # BLOCO-4: Domain Layer (Clean Architecture)
│   │   │                                    # Entidades de domínio, value objects, interfaces de repositório
│   │   │                                    # Regras de negócio puras, sem dependências externas
│   │   │                                    # Independência total de infraestrutura
│   │   │
│   │   ├── 📁 entities/                     # Entidades de domínio
│   │   │   │                                # Objetos de negócio principais com identidade
│   │   │   ├── 📄 mcp.go                    # Entidade MCP (raiz do agregado principal)
│   │   │   │                                # Função: NewMCP, SetPath, AddFeature, AddContext
│   │   │   │                                # Regras: nome obrigatório, stack válida, features únicas
│   │   │   │                                # Invariantes: path nunca vazio, timestamps automáticos
│   │   │   │
│   │   │   ├── 📄 knowledge.go             # Entidade Knowledge Base (AI/RAG)
│   │   │   │                                # Função: NewKnowledge, AddDocument, AddEmbedding
│   │   │   │                                # Regras: nome obrigatório, documentos obrigatórios
│   │   │   │                                # Invariantes: embeddings vinculados a documentos
│   │   │   │
│   │   │   ├── 📄 project.go                # Entidade Project
│   │   │   │                                # Função: NewProject, SetStatus, Activate, Archive
│   │   │   │                                # Regras: nome obrigatório, MCP ID obrigatório
│   │   │   │                                # Invariantes: status válido, transições controladas
│   │   │   │
│   │   │   ├── 📄 template.go              # Entidade Template
│   │   │   │                                # Função: NewTemplate, SetContent, AddVariable
│   │   │   │                                # Regras: nome obrigatório, conteúdo obrigatório
│   │   │   │                                # Invariantes: variáveis sem duplicatas, versionamento
│   │   │   │
│   │   │   ├── 📄 memory.go                # Entidade Memory (extensão - AI Memory Management)
│   │   │   │                                # Função: NewMemory, SetContent, RecordAccess
│   │   │   │                                # Tipos: EpisodicMemory, SemanticMemory, WorkingMemory
│   │   │   │                                # Regras: tipo obrigatório, conteúdo obrigatório
│   │   │   │
│   │   │   ├── 📄 finetuning.go            # Entidades Fine-tuning (extensão)
│   │   │   │                                # Função: NewDataset, NewTrainingJob, NewModelVersion
│   │   │   │                                # Entidades: Dataset, TrainingJob, ModelVersion
│   │   │   │                                # Regras: validações de status, métricas, checkpoints
│   │   │   │
│   │   │   ├── 📄 mcp_test.go              # Testes unitários da entidade MCP
│   │   │   │                                # Testa: criação, validações, features, context
│   │   │   │
│   │   │   └── 📄 errors.go                # Erros de domínio customizados
│   │   │                                    # Função: NewDomainError, Error, Unwrap
│   │   │                                    # Códigos: INVALID_INPUT, NOT_FOUND, ALREADY_EXISTS
│   │   │                                    # Erros pré-definidos: ErrMCPNotFound, ErrKnowledgeNotFound
│   │   │
│   │   ├── 📁 value_objects/                # Value Objects (imutáveis)
│   │   │   │                                # Objetos imutáveis com significado e validação
│   │   │   ├── 📄 technology.go            # StackType (go-premium, tinygo, web)
│   │   │   │                                # Função: NewStackType, IsValid, ValidStackTypes
│   │   │   │                                # Validação: apenas valores permitidos
│   │   │   │
│   │   │   ├── 📄 technology_test.go       # Testes unitários do StackType
│   │   │   │
│   │   │   ├── 📄 feature.go                # Feature (Enable/Disable + configs)
│   │   │   │                                # Função: NewFeature, Enable, Disable, SetConfig
│   │   │   │                                # Regras: nome obrigatório, imutabilidade preservada
│   │   │   │                                # Métodos: Equals para comparação
│   │   │   │
│   │   │   ├── 📄 feature_test.go          # Testes unitários do Feature
│   │   │   │
│   │   │   └── 📄 validation_rule.go       # ValidationRule (extensão)
│   │   │                                    # Função: NewValidationRule, Validate
│   │   │                                    # Tipos: Required, Min, Max, Pattern, Custom
│   │   │
│   │   ├── 📁 repositories/                 # Interfaces de Repositório
│   │   │   │                                # Contratos para persistência (implementados na infra)
│   │   │   ├── 📄 mcp_repository.go         # Interface MCPRepository
│   │   │   │                                # Métodos: Save, FindByID, FindByName, List, Delete, Exists
│   │   │   │                                # Filtros: MCPFilters (Stack, HasContext, Limit, Offset)
│   │   │   │
│   │   │   ├── 📄 knowledge_repository.go  # Interface KnowledgeRepository
│   │   │   │                                # Métodos: Save, FindByID, FindByName, List, Delete, Exists
│   │   │   │                                # Filtros: KnowledgeFilters (MinVersion, Limit, Offset)
│   │   │   │
│   │   │   ├── 📄 project_repository.go    # Interface ProjectRepository
│   │   │   │                                # Métodos: Save, FindByID, FindByMCPID, List, Delete, Exists
│   │   │   │                                # Filtros: ProjectFilters (MCPID, Status, Limit, Offset)
│   │   │   │
│   │   │   └── 📄 template_repository.go    # Interface TemplateRepository
│   │   │                                    # Métodos: Save, FindByID, FindByName, List, Delete, Exists
│   │   │                                    # Filtros: TemplateFilters (Stack, Limit, Offset)
│   │   │
│   │   └── 📁 services/                     # Domain Services
│   │       │                                # Regras de negócio que não pertencem a uma entidade
│   │       │                                # Não acessam banco, não fazem IO, não dependem de infra
│   │       ├── 📄 mcp_domain_service.go     # MCPDomainService
│   │       │                                # Função: ValidateMCP, CanAddFeature, CanAttachContext
│   │       │                                # Regras: validação de MCP completo, features sem conflitos
│   │       │
│   │       ├── 📄 knowledge_domain_service.go # KnowledgeDomainService
│   │       │                                # Função: ValidateKnowledge, CanAddDocument, CanAddEmbedding
│   │       │                                # Regras: conhecimento deve ter documentos, embeddings válidos
│   │       │
│   │       ├── 📄 ai_domain_service.go      # AIDomainService
│   │       │                                # Função: ValidateKnowledgeContext, CanUseKnowledgeForInference
│   │       │                                # Regras: contexto válido para AI, conhecimento pronto para inferência
│   │       │
│   │       └── 📄 template_domain_service.go # TemplateDomainService
│   │                                        # Função: ValidateTemplate, CanAddVariable, ShouldIncrementVersion
│   │                                        # Regras: template válido, variáveis sem duplicatas, versionamento
│   │
│   ├── 📁 application/                      # BLOCO-1: Application Layer (Clean Architecture)
│   │   │                                    # Casos de uso, DTOs, orquestração de serviços
│   │   │                                    # Coordena operações entre domínio e infraestrutura
│   │   │
│   │   ├── 📁 use_cases/                    # Casos de uso (application services)
│   │   │   │                                # Orquestram operações de negócio
│   │   │   ├── 📄 create_mcp_use_case.go    # Caso de uso: Criar MCP
│   │   │   ├── 📄 generate_mcp_use_case.go  # Caso de uso: Gerar MCP
│   │   │   ├── 📄 validate_mcp_use_case.go  # Caso de uso: Validar MCP
│   │   │   └── 📄 manage_knowledge_use_case.go # Caso de uso: Gerenciar Knowledge
│   │   │
│   │   ├── 📁 dtos/                         # Data Transfer Objects
│   │   │   │                                # Objetos para transferência de dados entre camadas
│   │   │   ├── 📄 mcp_dto.go                # DTOs relacionados a MCP
│   │   │   ├── 📄 knowledge_dto.go          # DTOs relacionados a Knowledge
│   │   │   └── 📄 common_dto.go             # DTOs comuns
│   │   │
│   │   └── 📁 ports/                        # Portas (interfaces de aplicação)
│   │       │                                # Contratos para adapters externos
│   │       └── 📄 ports.go                   # Interfaces de entrada/saída
│   │
│   ├── 📁 infrastructure/                   # BLOCO-7: Infrastructure Layer
│   │   │                                    # Implementações concretas de persistência, mensageria, cloud
│   │   │                                    # Adaptadores para sistemas externos
│   │   │
│   │   ├── 📁 persistence/                  # Persistência de dados
│   │   │   │                                # Implementações de repositórios para diferentes bancos
│   │   │   │
│   │   │   ├── 📁 relational/               # Bancos relacionais (PostgreSQL)
│   │   │   │   ├── 📄 postgres_mcp_repository.go      # Repositório MCP PostgreSQL
│   │   │   │   ├── 📄 postgres_knowledge_repository.go # Repositório Knowledge PostgreSQL
│   │   │   │   ├── 📄 postgres_project_repository.go  # Repositório Project PostgreSQL
│   │   │   │   ├── 📄 postgres_template_repository.go  # Repositório Template PostgreSQL
│   │   │   │   ├── 📄 schema.go                        # Schemas SQL (mcps, knowledge, projects, templates)
│   │   │   │   └── 📄 migrations.go                   # Migrações de banco
│   │   │   │
│   │   │   ├── 📁 document/                 # Bancos NoSQL (MongoDB, CouchDB)
│   │   │   │   ├── 📄 document_client.go               # Cliente genérico de Document DB
│   │   │   │   ├── 📄 mongodb_client.go                 # Cliente MongoDB
│   │   │   │   ├── 📄 couchdb_client.go                # Cliente CouchDB
│   │   │   │   └── 📄 document_query.go                # Query builder para documentos
│   │   │   │
│   │   │   ├── 📁 cache/                    # Cache distribuído (Redis, Memcached, Hazelcast)
│   │   │   │   ├── 📄 cache_client.go                  # Cliente genérico de cache
│   │   │   │   ├── 📄 redis_cluster.go                 # Cluster Redis
│   │   │   │   ├── 📄 memcached_cluster.go            # Cluster Memcached
│   │   │   │   ├── 📄 hazelcast_cluster.go            # Cluster Hazelcast
│   │   │   │   └── 📄 cache_consistency.go            # Consistência de cache
│   │   │   │
│   │   │   ├── 📁 graph/                    # Bancos de grafos (Neo4j, ArangoDB)
│   │   │   │   ├── 📄 graph_client.go                  # Cliente genérico de Graph DB
│   │   │   │   ├── 📄 neo4j_client.go                 # Cliente Neo4j
│   │   │   │   ├── 📄 arango_client.go                # Cliente ArangoDB
│   │   │   │   └── 📄 graph_traversal.go              # Travessia e queries de grafos
│   │   │   │
│   │   │   ├── 📁 vector/                   # Bancos vetoriais (Qdrant, Pinecone, Weaviate)
│   │   │   │   ├── 📄 vector_client.go                 # Cliente genérico de Vector DB
│   │   │   │   ├── 📄 qdrant_client.go                 # Cliente Qdrant
│   │   │   │   ├── 📄 pinecone_client.go              # Cliente Pinecone
│   │   │   │   ├── 📄 weaviate_client.go               # Cliente Weaviate
│   │   │   │   └── 📄 hybrid_search.go                # Busca híbrida (vector + outros sinais)
│   │   │   │
│   │   │   └── 📁 time_series/              # Bancos time series (InfluxDB, Prometheus)
│   │   │       ├── 📄 timeseries_client.go             # Cliente genérico de Time Series DB
│   │   │       ├── 📄 influxdb_client.go               # Cliente InfluxDB
│   │   │       ├── 📄 prometheus_client.go             # Cliente Prometheus
│   │   │       └── 📄 timeseries_analytics.go         # Analytics de time series
│   │   │
│   │   ├── 📁 messaging/                    # Mensageria (NATS, RabbitMQ, Kafka, Pulsar)
│   │   │   │                                # Sistema de mensageria assíncrona e eventos
│   │   │   │
│   │   │   ├── 📄 message_broker.go         # Broker de mensagens genérico
│   │   │   ├── 📄 event_router.go           # Roteador de eventos
│   │   │   │
│   │   │   ├── 📁 pubsub/                   # Pub/Sub (NATS, RabbitMQ, Redis)
│   │   │   │   ├── 📄 pubsub_client.go      # Cliente genérico Pub/Sub
│   │   │   │   ├── 📄 nats_pubsub.go        # Pub/Sub NATS
│   │   │   │   ├── 📄 rabbitmq_cluster.go  # Cluster RabbitMQ
│   │   │   │   └── 📄 redis_pubsub.go       # Pub/Sub Redis
│   │   │   │
│   │   │   ├── 📁 streaming/                # Streaming (NATS JetStream, Kafka, Pulsar)
│   │   │   │   ├── 📄 stream_client.go      # Cliente genérico de streaming
│   │   │   │   ├── 📄 nats_jetstream.go     # NATS JetStream
│   │   │   │   ├── 📄 kafka_cluster.go      # Cluster Kafka
│   │   │   │   └── 📄 pulsar_cluster.go     # Cluster Pulsar
│   │   │   │
│   │   │   └── 📁 rpc/                      # RPC (gRPC, HTTP/2, Thrift)
│   │   │       ├── 📄 rpc_client.go         # Cliente genérico RPC
│   │   │       ├── 📄 grpc_cluster.go       # Cluster gRPC
│   │   │       ├── 📄 http2_cluster.go      # Cluster HTTP/2
│   │   │       ├── 📄 thrift_cluster.go     # Cluster Thrift
│   │   │       └── 📄 connection_pool.go   # Pool de conexões RPC
│   │   │
│   │   ├── 📁 cloud/                        # Integrações com cloud
│   │   │   │                                # Clientes para serviços cloud (Kubernetes, Docker, Serverless)
│   │   │   │
│   │   │   ├── 📁 kubernetes/               # Kubernetes
│   │   │   │   ├── 📄 k8s_client.go         # Cliente Kubernetes
│   │   │   │   ├── 📄 deployment_manager.go # Gerenciamento de deployments
│   │   │   │   ├── 📄 service_manager.go    # Gerenciamento de services
│   │   │   │   └── 📄 config_map_manager.go # Gerenciamento de ConfigMaps
│   │   │   │
│   │   │   ├── 📁 docker/                  # Docker
│   │   │   │   ├── 📄 docker_client.go      # Cliente Docker
│   │   │   │   ├── 📄 container_manager.go  # Gerenciamento de containers
│   │   │   │   ├── 📄 image_builder.go      # Builder de imagens
│   │   │   │   └── 📄 registry_manager.go  # Gerenciamento de registries
│   │   │   │
│   │   │   └── 📁 serverless/               # Serverless (AWS Lambda, Azure Functions, GCP Functions)
│   │   │       ├── 📄 faas_manager.go       # Gerenciador FaaS genérico
│   │   │       ├── 📄 function_deployer.go # Deployer de funções
│   │   │       ├── 📄 aws_lambda.go         # AWS Lambda
│   │   │       ├── 📄 azure_functions.go    # Azure Functions
│   │   │       └── 📄 google_cloud_functions.go # Google Cloud Functions
│   │   │
│   │   ├── 📁 compute/                      # Compute (CPU, GPU, Serverless, Distributed)
│   │   │   │                                # Gerenciamento de compute para IA e processamento
│   │   │   │
│   │   │   ├── 📁 cpu/                      # Compute CPU
│   │   │   │   ├── 📄 cpu_manager.go        # Gerenciador de CPU
│   │   │   │   ├── 📄 process_scheduler.go  # Agendador de processos
│   │   │   │   └── 📄 thread_pool.go        # Pool de threads
│   │   │   │
│   │   │   ├── 📁 gpu/                      # Compute GPU (CUDA, OpenCL, TensorRT)
│   │   │   │   ├── 📄 gpu_pool.go           # Pool de GPUs
│   │   │   │   ├── 📄 cuda_manager.go       # Gerenciador CUDA
│   │   │   │   ├── 📄 opencl_manager.go     # Gerenciador OpenCL
│   │   │   │   └── 📄 tensorrt_inference.go # Inferência TensorRT
│   │   │   │
│   │   │   ├── 📁 serverless/               # Compute Serverless (RunPod, Cloud Functions)
│   │   │   │   ├── 📄 runpod_client.go      # Cliente RunPod API
│   │   │   │   ├── 📄 lambda_manager.go     # Gerenciador Lambda
│   │   │   │   ├── 📄 cloud_functions.go    # Cloud Functions
│   │   │   │   ├── 📄 faas_manager.go       # Gerenciador FaaS
│   │   │   │   └── 📄 function_orchestrator.go # Orquestrador de funções
│   │   │   │
│   │   │   └── 📁 distributed/               # Compute Distribuído (Dask, Ray, Spark)
│   │   │       ├── 📄 task_distributor.go    # Distribuidor de tarefas
│   │   │       ├── 📄 dask_cluster.go        # Cluster Dask
│   │   │       ├── 📄 ray_cluster.go         # Cluster Ray
│   │   │       └── 📄 spark_cluster.go       # Cluster Spark
│   │   │
│   │   ├── 📁 llm/                          # Clientes LLM
│   │   │   │                                # Clientes para diferentes provedores de LLM
│   │   │   ├── 📄 openai_client.go          # Cliente OpenAI
│   │   │   ├── 📄 gemini_client.go          # Cliente Gemini (Google)
│   │   │   └── 📄 glm_client.go             # Cliente GLM (ChatGLM)
│   │   │
│   │   └── 📁 network/                      # Rede e comunicação
│   │       │                                # Clientes HTTP, gRPC, WebSocket, CDN, Load Balancer
│   │       │
│   │       ├── 📁 load_balancer/            # Load Balancers
│   │       │   ├── 📄 nginx_lb.go           # Load Balancer Nginx
│   │       │   ├── 📄 envoy_lb.go           # Load Balancer Envoy
│   │       │   ├── 📄 haproxy_lb.go          # Load Balancer HAProxy
│   │       │   └── 📄 health_checker.go     # Verificador de saúde
│   │       │
│   │       ├── 📁 cdn/                      # CDN (Content Delivery Network)
│   │       │   ├── 📄 cdn_client.go         # Cliente genérico CDN
│   │       │   ├── 📄 aws_cdn.go            # AWS CloudFront
│   │       │   ├── 📄 cloudflare_cdn.go      # Cloudflare CDN
│   │       │   ├── 📄 fastly_cdn.go         # Fastly CDN
│   │       │   └── 📄 cache_optimizer.go    # Otimizador de cache CDN
│   │       │
│   │       └── 📁 security/                 # Segurança de rede
│   │           ├── 📄 rate_limiter.go       # Rate limiter
│   │           ├── 📄 ddos_protection.go    # Proteção DDoS
│   │           ├── 📄 ssl_terminator.go     # SSL/TLS terminator
│   │           └── 📄 waf.go                # Web Application Firewall
│   │
│   ├── 📁 interfaces/                       # BLOCO-8: Interface Layer (Clean Architecture)
│   │   │                                    # Adaptadores de entrada/saída (HTTP, gRPC, CLI, Events)
│   │   │                                    # Conecta o mundo externo com a aplicação
│   │   │
│   │   ├── 📁 http/                         # Adaptadores HTTP (REST API)
│   │   │   │                                # Handlers HTTP usando Echo framework
│   │   │   ├── 📄 mcp_handler.go            # Handler HTTP para MCP
│   │   │   ├── 📄 knowledge_handler.go     # Handler HTTP para Knowledge
│   │   │   ├── 📄 model_handler.go          # Handler HTTP para Models
│   │   │   ├── 📄 health_handler.go         # Handler HTTP para health checks
│   │   │   ├── 📄 metrics_handler.go        # Handler HTTP para métricas (Prometheus)
│   │   │   ├── 📄 router.go                 # Configuração de rotas
│   │   │   ├── 📄 middleware.go             # Middlewares (auth, logging, cors)
│   │   │   └── 📄 server.go                 # Servidor HTTP principal
│   │   │
│   │   ├── 📁 grpc/                         # Adaptadores gRPC
│   │   │   │                                # Servidores gRPC para comunicação RPC
│   │   │   ├── 📄 mcp_server.go             # Servidor gRPC para MCP
│   │   │   ├── 📄 knowledge_server.go       # Servidor gRPC para Knowledge
│   │   │   ├── 📄 model_server.go           # Servidor gRPC para Models
│   │   │   ├── 📄 server.go                 # Servidor gRPC principal
│   │   │   └── 📄 interceptors.go          # Interceptors (auth, logging)
│   │   │
│   │   ├── 📁 cli/                          # Adaptadores CLI
│   │   │   │                                # Comandos CLI usando Cobra framework
│   │   │   ├── 📄 root.go                   # Comando raiz da CLI
│   │   │   ├── 📄 mcp_command.go            # Comandos MCP (create, list, validate)
│   │   │   ├── 📄 knowledge_command.go      # Comandos Knowledge
│   │   │   ├── 📄 model_command.go          # Comandos Model
│   │   │   ├── 📄 generate_command.go       # Comandos de geração
│   │   │   ├── 📄 validate_command.go       # Comandos de validação
│   │   │   ├── 📄 deploy_command.go         # Comandos de deploy
│   │   │   └── 📄 config_command.go         # Comandos de configuração
│   │   │
│   │   └── 📁 messaging/                    # Adaptadores de mensageria
│   │       │                                # Handlers de eventos e mensagens assíncronas
│   │       ├── 📄 event_handler.go          # Handler de eventos de domínio
│   │       ├── 📄 message_handler.go        # Handler de mensagens NATS
│   │       └── 📄 subscriber.go             # Subscritor de eventos
│   │
│   ├── 📁 mcp/                              # BLOCO-2: MCP Protocol & Generation
│   │   │                                    # Protocolo MCP, geração de projetos, validação e registry
│   │   │
│   │   ├── 📁 protocol/                     # Protocolo MCP (JSON-RPC 2.0)
│   │   │   │                                # Servidor MCP com suporte a stdio e HTTP/SSE
│   │   │   ├── 📄 server.go                 # MCPServer: stdio/HTTP, handlers, graceful shutdown
│   │   │   │                                # Função: NewMCPServer, Start, Stop, RegisterHandler, GetCapabilities
│   │   │   ├── 📄 tools.go                  # Definições de tools MCP com schemas JSON
│   │   │   │                                # Função: GetToolDefinitions, generateProjectTool, validateProjectTool
│   │   │   │                                # Função: listTemplatesTool, describeStackTool, listProjectsTool
│   │   │   │                                # Função: getProjectInfoTool, deleteProjectTool, updateProjectTool
│   │   │   ├── 📄 handlers.go               # Handlers para todas as tools MCP
│   │   │   │                                # Função: HandlerManager, GetAllHandlers
│   │   │   │                                # Função: GenerateProjectHandler, ValidateProjectHandler
│   │   │   │                                # Função: ListTemplatesHandler, DescribeStackHandler
│   │   │   │                                # Função: ListProjectsHandler, GetProjectInfoHandler
│   │   │   │                                # Função: DeleteProjectHandler, UpdateProjectHandler
│   │   │   ├── 📄 router.go                 # Roteamento de tools MCP
│   │   │   │                                # Função: NewToolRouter, Route, handleListTools, handleCallTool
│   │   │   │                                # Função: handleInitialize, handlePing, validateParams
│   │   │   ├── 📄 types.go                  # Tipos JSON-RPC 2.0
│   │   │   │                                # Função: JSONRPCRequest, JSONRPCResponse, JSONRPCError
│   │   │   │                                # Função: Tool, ToolCall, ToolResult, InitializeParams
│   │   │   ├── 📄 client.go                 # Cliente MCP (adicional)
│   │   │   ├── 📄 server_test.go            # Testes do servidor
│   │   │   ├── 📄 handlers_test.go          # Testes dos handlers
│   │   │   ├── 📄 router_test.go            # Testes do router
│   │   │   └── 📄 tools_test.go             # Testes das tools
│   │   │
│   │   ├── 📁 generators/                   # Fábrica de geração
│   │   │   │                                # Generators para diferentes stacks tecnológicos
│   │   │   ├── 📄 base_generator.go         # BaseGenerator: lógica comum de templates
│   │   │   │                                # Função: NewBaseGenerator, Generate, validateRequest
│   │   │   │                                # Função: createProjectStructure, getTemplateFiles
│   │   │   │                                # Função: processTemplate, prepareTemplateData
│   │   │   │                                # Função: createTemplateFuncMap (upper, lower, snakeCase, etc.)
│   │   │   ├── 📄 generator_factory.go      # GeneratorFactory: Strategy Pattern
│   │   │   │                                # Função: NewGeneratorFactory, RegisterGenerator, GetGenerator
│   │   │   │                                # Função: ListGenerators, GetGeneratorInfo, ValidateRequest
│   │   │   │                                # Função: Generate, GetFactoryStats, Shutdown
│   │   │   ├── 📄 go_generator.go           # GoGenerator: Gerador de stack Go
│   │   │   │                                # Função: NewGoGenerator, Generate, getTemplateFiles
│   │   │   │                                # Função: postProcessGoProject (verificação de estrutura)
│   │   │   │                                # Função: getGoVersion, getDependencies, CreateDockerfile
│   │   │   ├── 📄 web_generator.go          # WebGenerator: Gerador Web/React/Vue
│   │   │   │                                # Função: NewWebGenerator, Generate, getTemplateFiles
│   │   │   │                                # Função: postProcessWebProject (verificação de estrutura)
│   │   │   ├── 📄 tinygo_generator.go       # TinyGoGenerator: Gerador WASM/Embedded
│   │   │   │                                # Função: NewTinyGoGenerator, Generate, getTemplateFiles
│   │   │   │                                # Função: postProcessTinyGoProject (verificação de estrutura)
│   │   │   ├── 📄 rust_generator.go         # RustGenerator: Gerador Rust (adicional)
│   │   │   │                                # Função: NewRustGenerator, Validate
│   │   │   └── 📄 generator_factory_test.go # Testes da factory
│   │   │
│   │   ├── 📁 validators/                   # Controle de qualidade
│   │   │   │                                # Validators para estrutura, dependências, segurança
│   │   │   ├── 📄 validator_factory.go      # ValidatorFactory: Factory de validators
│   │   │   │                                # Função: NewValidatorFactory, GetStructureValidator
│   │   │   │                                # Função: GetDependencyValidator, GetTreeValidator
│   │   │   │                                # Função: GetSecurityValidator, GetConfigValidator
│   │   │   │                                # Função: ValidateAll
│   │   │   ├── 📄 structure_validator.go   # StructureValidator: Validação de estrutura
│   │   │   │                                # Função: NewStructureValidator, Validate, validateRule
│   │   │   │                                # Função: getDefaultStructureRules
│   │   │   ├── 📄 dependency_validator.go   # DependencyValidator: Validação de dependências
│   │   │   │                                # Função: NewDependencyValidator, Validate
│   │   │   │                                # Análise de go.mod com parsing e contagem de dependências
│   │   │   ├── 📄 base_validator.go         # BaseValidator: Validador base (adicional)
│   │   │   ├── 📄 code_validator.go         # CodeValidator: Validação de código (adicional)
│   │   │   ├── 📄 template_validator.go    # TemplateValidator: Validação de templates (adicional)
│   │   │   │                                # Função: NewTemplateValidator, ValidateTemplate
│   │   │   │                                # Função: validateManifest, validateTemplateFiles
│   │   │   │                                # Função: validatePlaceholders, ValidateAllTemplates
│   │   │   └── 📄 validator_factory_test.go # Testes da factory
│   │   │
│   │   └── 📁 registry/                     # Auto-descoberta
│   │       │                                # Registry de MCPs, templates, projetos e serviços
│   │       ├── 📄 mcp_registry.go           # MCPRegistry: Registro de MCPs e Templates
│   │       │                                # Função: NewMCPRegistry, RegisterProject, GetProjectByName
│   │       │                                # Função: ListProjects, ListTemplates, GetStackInfo
│   │       │                                # Função: RegisterService, GetRegistryStats
│   │       │                                # Função: saveToStorage (persistência JSON), loadFromStorage
│   │       ├── 📄 service_registry.go       # ServiceRegistry: Registro de serviços (adicional)
│   │       │                                # Função: NewServiceRegistry, RegisterService, GetService
│   │       │                                # Função: ListServices, UpdateServiceStatus
│   │       ├── 📄 template_registry.go      # TemplateRegistry: Registro de templates (adicional)
│   │       │                                # Função: NewTemplateRegistry, LoadTemplates
│   │       │                                # Função: GetTemplate, ListTemplates, SearchTemplates
│   │       │                                # Função: ValidateTemplate, RegisterTemplate
│   │       ├── 📄 discovery.go              # ServiceDiscovery: Descoberta de serviços (adicional)
│   │       │                                # Função: NewServiceDiscovery, DiscoverServices
│   │       │                                # Função: WatchServices, pollForChanges
│   │       └── 📄 mcp_registry_test.go      # Testes do registry
│   │
│   ├── 📁 ai/                               # BLOCO-6: AI Layer
│   │   │                                    # Integração com IA, RAG, conhecimento, memória
│   │   │                                    # Função: Cérebro cognitivo do Hulk
│   │   │                                    # Responsabilidades: LLM, RAG, Memória, Finetuning
│   │   │
│   │   ├── 📁 core/                         # AI Core (Núcleo cognitivo)
│   │   │   │                                # Função: Interface LLM, prompts, roteamento, métricas
│   │   │   │                                # Responsabilidades: Unificação, fallback, observabilidade
│   │   │   ├── 📄 llm_interface.go          # ✅ Implementado - Interface LLM unificada
│   │   │   │                                # Função: NewLLMInterface, Generate, GenerateStream, GetAvailableProviders, GetModels
│   │   │   │                                # Tipos: LLMProvider, LLMRequest, LLMResponse, LLMError
│   │   │   ├── 📄 prompt_builder.go         # ✅ Implementado - Builder de prompts
│   │   │   │                                # Função: NewPromptBuilder, Build
│   │   │   │                                # Tipos: PromptPolicy, PromptContext, Message
│   │   │   ├── 📄 router.go                 # ✅ Implementado - Router inteligente
│   │   │   │                                # Função: NewRouter, SelectProvider, SelectFallback
│   │   │   │                                # Estratégias: Cost, Latency, Quality, Balanced, Fallback
│   │   │   ├── 📄 metrics.go                # ✅ Implementado - Métricas de IA
│   │   │   │                                # Função: NewMetrics, RecordGeneration, RecordError, GetAverageLatency, GetP95Latency
│   │   │   │                                # Tipos: ProviderStats
│   │   │   ├── 📄 llm_interface_test.go     # ✅ Testes unitários
│   │   │   ├── 📄 prompt_builder_test.go    # ✅ Testes unitários
│   │   │   ├── 📄 router_test.go            # ✅ Testes unitários
│   │   │   └── 📄 metrics_test.go           # ✅ Testes unitários
│   │   │
│   │   ├── 📁 knowledge/                    # Knowledge (RAG - Vector + Graph)
│   │   │   │                                # Função: Ingestão, indexação e recuperação híbrida
│   │   │   │                                # Responsabilidades: VectorDB, GraphDB, RAG híbrido
│   │   │   ├── 📄 knowledge_store.go        # ✅ Implementado - Store de conhecimento
│   │   │   │                                # Função: NewKnowledgeStore, AddKnowledge, AddDocument, AddEmbedding, SearchDocuments
│   │   │   │                                # Tipos: KnowledgeStats, DocumentInput
│   │   │   ├── 📄 retriever.go              # ✅ Implementado - Hybrid Retriever
│   │   │   │                                # Função: NewHybridRetriever, Retrieve
│   │   │   │                                # Fusion: ReciprocalRankFusion (RRF)
│   │   │   │                                # Tipos: RetrievalResult, KnowledgeContext, FusionStrategy
│   │   │   ├── 📄 indexer.go                # ✅ Implementado - Indexador de documentos
│   │   │   │                                # Função: NewIndexer, IndexDocument, UpdateVectorIndex, Search, DeleteKnowledge
│   │   │   │                                # Tipos: VectorClient, GraphClient, Embedder
│   │   │   ├── 📄 knowledge_graph.go        # ✅ Implementado - Graph de conhecimento
│   │   │   │                                # Função: NewKnowledgeGraph, CreateEntity, CreateRelation, Traverse, Query
│   │   │   │                                # Tipos: GraphNode
│   │   │   ├── 📄 semantic_search.go        # ✅ Implementado - Busca semântica
│   │   │   │                                # Função: NewSemanticSearch, Search, SearchWithFilters, SimilaritySearch
│   │   │   ├── 📄 knowledge_store_test.go   # ✅ Testes unitários
│   │   │   ├── 📄 retriever_test.go         # ✅ Testes unitários
│   │   │   └── 📄 indexer_test.go           # ✅ Testes unitários
│   │   │
│   │   ├── 📁 memory/                       # Memory (Episodic, Semantic, Working)
│   │   │   │                                # Função: Memória viva do agente
│   │   │   │                                # Responsabilidades: Episódica, semântica, trabalho
│   │   │   ├── 📄 memory_store.go           # ✅ Implementado - Store de memória
│   │   │   │                                # Função: NewMemoryStore, SaveEpisodic, SaveSemantic, SaveWorking, GetEpisodic, GetSemantic, GetWorking
│   │   │   │                                # Tipos: MemoryRepository, CacheClient
│   │   │   ├── 📄 episodic_memory.go       # ✅ Implementado - Memória episódica
│   │   │   │                                # Função: NewEpisodicMemoryManager, Create, AddEvent, GetEvents, GetRecentEvents, Consolidate
│   │   │   ├── 📄 semantic_memory.go        # ✅ Implementado - Memória semântica
│   │   │   │                                # Função: NewSemanticMemoryManager, Create, AddConcept, AddRelated, GetByConcept, Search, ConsolidateFromEpisodic
│   │   │   ├── 📄 working_memory.go         # ✅ Implementado - Memória de trabalho
│   │   │   │                                # Função: NewWorkingMemoryManager, Create, Get, AdvanceStep, SetContext, Complete
│   │   │   ├── 📄 memory_consolidation.go   # ✅ Implementado - Consolidação de memória
│   │   │   │                                # Função: NewMemoryConsolidation, ConsolidateSession, ConsolidateAll (requer SessionRepository), ConsolidateBatch
│   │   │   │                                # Tipos: ConsolidationPolicy
│   │   │   ├── 📄 memory_retrieval.go       # ✅ Implementado - Recuperação de memória
│   │   │   │                                # Função: NewMemoryRetrieval, Retrieve, RetrieveForPrompt, RetrieveRecent, RetrieveByImportance
│   │   │   │                                # Tipos: RetrievalStrategy, RetrieveContext, MemoryContext
│   │   │   ├── 📄 memory_store_test.go      # ✅ Testes unitários
│   │   │   └── 📄 episodic_memory_test.go   # ✅ Testes unitários
│   │   │
│   │   └── 📁 finetuning/                   # Finetuning (GPU Externa - RunPod)
│   │       │                                # Função: Treinamento remoto de modelos
│   │       │                                # Responsabilidades: RunPod, datasets, versionamento
│   │       ├── 📄 engine.go                 # ✅ Implementado - Engine de finetuning
│   │       │                                # Função: NewFinetuningEngine, StartTraining, CheckStatus, CancelTraining, GetLogs, CompleteTraining, Rollback
│   │       │                                # Tipos: RunPodClient, RunPodJobConfig, RunPodJobStatus
│   │       ├── 📄 finetuning_store.go       # ✅ Implementado - Store de finetuning
│   │       │                                # Função: NewFinetuningStore, SaveJob, GetJob, ListJobs, GetActiveJobs, SaveDataset, SaveModelVersion
│   │       │                                # Tipos: FinetuningRepository, JobFilters
│   │       ├── 📄 memory_manager.go         # ✅ Implementado - Gerenciador de memória
│   │       │                                # Função: NewMemoryManager, GenerateDataset, GenerateDatasetFromMemory, SaveDatasetToFile, ParseDatasetFile
│   │       │                                # Tipos: MemorySource, TrainingExample
│   │       ├── 📄 versioning.go             # ✅ Implementado - Versionamento
│   │       │                                # Função: NewVersioning, CreateVersion, ActivateVersion, Rollback, CompareVersions
│   │       │                                # Tipos: VersionComparison
│   │       ├── 📄 finetuning_prompt_builder.go # ✅ Implementado - Builder de prompts
│   │       │                                # Função: NewFinetuningPromptBuilder, BuildTrainingPrompt, BuildCompletionPrompt, BuildInstructionPrompt
│   │       └── 📄 finetuning_store_test.go  # ✅ Testes unitários
│   │
│   ├── 📁 state/                            # BLOCO-3: STATE MANAGEMENT
│   │   │                                    # Gerenciamento de Estado Distribuído
│   │   │                                    # Função: Estado vivo, linha do tempo imutável, cache acelerado
│   │   │                                    # Responsabilidades: Store distribuído, Event Sourcing, Cache multi-nível
│   │   │
│   │   ├── 📁 store/                        # Estado Distribuído Vivo
│   │   │   │                                # Função: Gerenciamento de estado versionado e distribuído
│   │   │   │                                # Responsabilidades: get/set versionado, CAS, locks, snapshots, sync
│   │   │   │
│   │   │   ├── 📄 distributed_store.go      # ✅ Implementado
│   │   │   │                                # Interface: DistributedStore
│   │   │   │                                # Implementação: InMemoryDistributedStore
│   │   │   │                                # Funções: NewInMemoryDistributedStore, Get, Set, Delete,
│   │   │   │                                #         CompareAndSet, AcquireLock, ReleaseLock, Snapshot,
│   │   │   │                                #         Restore, SyncFrom, NotifyUpdate, Health, Stats, GetAllKeys
│   │   │   │                                # Tipos: VersionedState, StoreConfig, StoreHealth, StoreStats
│   │   │   │
│   │   │   ├── 📄 state_sync.go            # ✅ Implementado
│   │   │   │                                # Interface: StateSync
│   │   │   │                                # Implementação: StateSyncImpl
│   │   │   │                                # Funções: NewStateSync, SyncWithPeer, BroadcastUpdate,
│   │   │   │                                #         SubscribeToUpdates, GetSyncStatus
│   │   │   │                                # Tipos: SyncConfig, SyncStatus, SyncProgress
│   │   │   │
│   │   │   ├── 📄 conflict_resolver.go     # ✅ Implementado
│   │   │   │                                # Interface: ConflictResolver
│   │   │   │                                # Implementação: ConflictResolverImpl
│   │   │   │                                # Funções: NewConflictResolver, Resolve, GetStrategy, SetStrategy,
│   │   │   │                                #         GetConflictStats
│   │   │   │                                # Estratégias: LastWriteWins, FirstWriteWins, VectorClock,
│   │   │   │                                #            CRDTLastWriterWins, CRDTMerge
│   │   │   │                                # Tipos: Conflict, ConflictStats, ConflictResolverConfig
│   │   │   │
│   │   │   ├── 📄 state_snapshot.go        # ✅ Implementado (corrigido)
│   │   │   │                                # Interface: SnapshotManager
│   │   │   │                                # Implementação: SnapshotManagerImpl
│   │   │   │                                # Funções: NewSnapshotManager, CreateSnapshot, RestoreSnapshot,
│   │   │   │                                #         DeleteSnapshot, ListSnapshots, GetSnapshotInfo,
│   │   │   │                                #         IncrementalSnapshot, ScheduleAutoSnapshot, GetSnapshotStats
│   │   │   │                                # Tipos: SnapshotInfo, SnapshotData, SnapshotConfig, SnapshotStats
│   │   │   │                                # CORREÇÃO: captureFullState implementado completamente
│   │   │   │
│   │   │   ├── 📄 distributed_store_test.go # ✅ Testes unitários
│   │   │   ├── 📄 state_sync_test.go       # ✅ Testes unitários
│   │   │   ├── 📄 conflict_resolver_test.go # ✅ Testes unitários
│   │   │   └── 📄 state_snapshot_test.go   # ✅ Testes unitários
│   │   │
│   │   ├── 📁 events/                      # Linha do Tempo Imutável (Event Sourcing)
│   │   │   │                                # Função: Armazenamento e processamento de eventos imutáveis
│   │   │   │                                # Responsabilidades: event store, replay, projeções, versionamento
│   │   │   │
│   │   │   ├── 📄 event_store.go           # ✅ Implementado
│   │   │   │                                # Interface: EventStore
│   │   │   │                                # Implementação: InMemoryEventStore
│   │   │   │                                # Funções: NewInMemoryEventStore, SaveEvent, SaveEvents,
│   │   │   │                                #         GetEvents, GetAllEvents, GetEventsByType,
│   │   │   │                                #         GetEventsByTimeRange, StreamEvents, StreamAllEvents,
│   │   │   │                                #         GetAggregateInfo, GetEventStats, GetStoreInfo,
│   │   │   │                                #         CreateSnapshot, GetSnapshot, Health, CompactEvents, PruneEvents
│   │   │   │                                # Tipos: Event, EventType, AggregateInfo, EventStoreStats,
│   │   │   │                                #        Snapshot, EventStoreConfig
│   │   │   │
│   │   │   ├── 📄 event_projection.go      # ✅ Implementado
│   │   │   │                                # Interface: EventProjection
│   │   │   │                                # Implementação: EventProjectionImpl
│   │   │   │                                # Funções: NewEventProjection, CreateProjection, UpdateProjection,
│   │   │   │                                #         DeleteProjection, GetProjection, ListProjections,
│   │   │   │                                #         ProcessEvent, ProcessEvents, RebuildProjection,
│   │   │   │                                #         RebuildAllProjections, GetProjectionState, ResetProjection,
│   │   │   │                                #         GetProjectionStats, GetProjectionMetrics
│   │   │   │                                # Tipos: Projection, ProjectionType, ProjectionHandler,
│   │   │   │                                #        ProjectionState, ProjectionStats, ProjectionMetrics
│   │   │   │
│   │   │   ├── 📄 event_replay.go         # ✅ Implementado
│   │   │   │                                # Interface: EventReplay
│   │   │   │                                # Implementação: EventReplayImpl
│   │   │   │                                # Funções: NewEventReplay, ReplayEvents, ReplayAllEvents,
│   │   │   │                                #         ReplayEventsByType, ReplayFromSnapshot, ReplayToState,
│   │   │   │                                #         GetReplayStats
│   │   │   │                                # Estratégias: Sequential, Parallel, Batch
│   │   │   │                                # Tipos: ReplayConfig, ReplayProgress, ReplayHandler, ReplayStats
│   │   │   │
│   │   │   ├── 📄 event_versioning.go      # ✅ Implementado
│   │   │   │                                # Interface: EventVersioning
│   │   │   │                                # Implementação: EventVersioningImpl
│   │   │   │                                # Funções: NewEventVersioning, GetVersion, IncrementVersion,
│   │   │   │                                #         ValidateVersion, GetVersionHistory, AddVersionHistory,
│   │   │   │                                #         ResolveVersionConflict, GetVersionConflicts,
│   │   │   │                                #         GetVersioningStats
│   │   │   │                                # Tipos: VersionInfo, VersionHistoryEntry, VersionConflict,
│   │   │   │                                #        VersioningConfig, VersioningStats
│   │   │   │
│   │   │   ├── 📄 event_store_test.go      # ✅ Testes unitários
│   │   │   ├── 📄 event_projection_test.go  # ✅ Testes unitários
│   │   │   ├── 📄 event_replay_test.go     # ✅ Testes unitários
│   │   │   └── 📄 event_versioning_test.go # ✅ Testes unitários
│   │   │
│   │   └── 📁 cache/                        # Camada de Aceleração
│   │       │                                # Função: Cache multi-nível com coerência
│   │       │                                # Responsabilidades: L1/L2/L3, coerência, invalidação, distribuição
│   │       │
│   │       ├── 📄 state_cache.go            # ✅ Implementado
│   │       │                                # Interface: StateCache
│   │       │                                # Implementação: StateCacheImpl
│   │       │                                # Funções: NewStateCache, Get, Set, Delete, Clear,
│   │       │                                #         GetFromLevel, SetToLevel, GetStats, GetLevelStats, Health
│   │       │                                # Níveis: L1 (local), L2 (cluster), L3 (distribuído)
│   │       │                                # Eviction: LRU, LFU, FIFO
│   │       │                                # Tipos: CacheEntry, CacheConfig, CacheStats, LevelStats, CacheHealth
│   │       │
│   │       ├── 📄 cache_coherency.go        # ✅ Implementado
│   │       │                                # Interface: CoherencyManager
│   │       │                                # Implementação: CoherencyManagerImpl
│   │       │                                # Funções: NewCoherencyManager, Invalidate, InvalidatePattern,
│   │       │                                #         InvalidateAll, Update, GetCoherencyStatus,
│   │       │                                #         GetInvalidationStats, OnStoreUpdate, OnEventUpdate,
│   │       │                                #         StartBackgroundInvalidator, StopBackgroundInvalidator
│   │       │                                # Estratégias: WriteThrough, WriteBack, WriteAround, Invalidate, Update
│   │       │                                # Tipos: CoherencyConfig, InvalidationEvent, CoherencyStatus,
│   │       │                                #        InvalidationStats
│   │       │
│   │       ├── 📄 cache_distribution.go     # ✅ Implementado
│   │       │                                # Interface: CacheDistribution
│   │       │                                # Implementação: CacheDistributionImpl
│   │       │                                # Funções: NewCacheDistribution, PublishInvalidation, PublishUpdate,
│   │       │                                #         PublishClear, Subscribe, Unsubscribe, GetDistributionStats
│   │       │                                # Estratégias: PubSub, Gossip, Broadcast
│   │       │                                # Tipos: DistributionConfig, DistributionMessage,
│   │       │                                #        DistributionHandler, DistributionStats
│   │       │
│   │       ├── 📄 state_cache_test.go       # ✅ Testes unitários
│   │       ├── 📄 cache_coherency_test.go   # ✅ Testes unitários
│   │       └── 📄 cache_distribution_test.go # ✅ Testes unitários
│   │
│   ├── 📁 monitoring/                       # BLOCO-3: Monitoring Service
│   │   │                                    # Observabilidade, métricas, logs, tracing
│   │   │
│   │   ├── 📁 observability/                # Observabilidade geral
│   │   │   │                                # OpenTelemetry, tracing, métricas
│   │   │   ├── 📄 tracer.go                 # Tracer OpenTelemetry
│   │   │   ├── 📄 metrics.go                # Métricas Prometheus
│   │   │   └── 📄 exporter.go               # Exportador de observabilidade
│   │   │
│   │   ├── 📁 health/                       # Health checks
│   │   │   │                                # Verificação de saúde do sistema
│   │   │   ├── 📄 health_checker.go         # Verificador de saúde
│   │   │   ├── 📄 liveness.go              # Liveness probe
│   │   │   └── 📄 readiness.go             # Readiness probe
│   │   │
│   │   └── 📁 analytics/                    # Analytics
│   │       │                                # Análise de dados e métricas de negócio
│   │       ├── 📄 analytics.go              # Analytics engine
│   │       └── 📄 collector.go              # Coletor de analytics
│   │
│   ├── 📁 versioning/                       # BLOCO-5: VERSIONING & MIGRATION
│   │   │                                    # Versionamento avançado: conhecimento, modelos, dados
│   │   │                                    # Função: Controle de versões, migrações e evolução histórica
│   │   │                                    # Responsabilidades: Reprodutibilidade, auditoria, rollback, migração
│   │   │
│   │   ├── 📁 knowledge/                    # Versionamento de conhecimento
│   │   │   │                                # Versões de bases RAG, documentos, embeddings, grafos
│   │   │   ├── 📄 knowledge_versioning.go   # ✅ Interface KnowledgeVersioning e InMemoryKnowledgeVersioning
│   │   │   │                                # Função: CreateVersion, GetVersion, ListVersions, AddDocument,
│   │   │   │                                #         GetDocument, ListDocuments, DeleteVersion,
│   │   │   │                                #         GetLatestVersion, TagVersion
│   │   │   │                                # Tipos: KnowledgeVersion, KnowledgeDocument
│   │   │   │
│   │   │   ├── 📄 version_comparator.go     # ✅ Interface VersionComparator e InMemoryVersionComparator
│   │   │   │                                # Função: CompareVersions, CompareSemantic, CompareStructural,
│   │   │   │                                #         GetDiffSummary
│   │   │   │                                # Tipos: VersionDiff, DocumentChange
│   │   │   │
│   │   │   ├── 📄 rollback_manager.go       # ✅ Interface RollbackManager e InMemoryRollbackManager
│   │   │   │                                # Função: RollbackToVersion, GetRollbackOperation,
│   │   │   │                                #         ListRollbackOperations, ValidateRollback,
│   │   │   │                                #         CancelRollback
│   │   │   │                                # Tipos: RollbackOperation, RollbackStatus
│   │   │   │
│   │   │   ├── 📄 migration_engine.go       # ✅ Interface MigrationEngine e InMemoryMigrationEngine
│   │   │   │                                # Função: MigrateKnowledge, MigrateEmbeddings, MigrateGraph,
│   │   │   │                                #         GetMigration, ListMigrations, ValidateMigration,
│   │   │   │                                #         RollbackMigration, ValidateIntegrity
│   │   │   │                                # Tipos: Migration, MigrationStep, MigrationType, MigrationStatus
│   │   │   │
│   │   │   ├── 📄 knowledge_versioning_test.go # ✅ Testes unitários
│   │   │   └── 📄 version_comparator_test.go   # ✅ Testes unitários
│   │   │
│   │   ├── 📁 models/                       # Versionamento de modelos
│   │   │   │                                # Versões de modelos de IA, registro, deploy, A/B testing
│   │   │   ├── 📄 model_registry.go         # ✅ Interface ModelRegistry e InMemoryModelRegistry
│   │   │   │                                # Função: RegisterModel, GetModel, ListModels, UpdateModel,
│   │   │   │                                #         DeleteModel, RegisterVersion, GetVersion,
│   │   │   │                                #         ListVersions, GetLatestVersion, CalculateFingerprint
│   │   │   │                                # Tipos: Model, ModelVersion, ModelVersionStatus
│   │   │   │
│   │   │   ├── 📄 model_versioning.go       # ✅ Interface ModelVersioning e InMemoryModelVersioning
│   │   │   │                                # Função: CreateVersion, PromoteVersion, DeprecateVersion,
│   │   │   │                                #         GetVersionHistory, CompareVersions, GetVersionLifecycle
│   │   │   │                                # Estratégias: Semantic, Incremental, Timestamp
│   │   │   │                                # Tipos: VersioningStrategy, VersionComparison, VersionLifecycle
│   │   │   │
│   │   │   ├── 📄 ab_testing.go             # ✅ Interface ABTesting e InMemoryABTesting
│   │   │   │                                # Função: CreateTest, GetTest, StartTest, StopTest,
│   │   │   │                                #         RecordRequest, GetMetrics, EvaluateTest,
│   │   │   │                                #         SelectVersion, ListTests
│   │   │   │                                # Tipos: ABTest, TrafficSplit, ABTestMetrics,
│   │   │   │                                #        PromotionCriteria, TestEvaluation, ABTestStatus
│   │   │   │
│   │   │   ├── 📄 model_deployment.go       # ✅ Interface ModelDeployment e InMemoryModelDeployment
│   │   │   │                                # Função: CreateDeployment, GetDeployment, StartDeployment,
│   │   │   │                                #         StopDeployment, RollbackDeployment, GetDeploymentMetrics,
│   │   │   │                                #         CheckHealth, ListDeployments, GetActiveDeployment
│   │   │   │                                # Estratégias: Canary, BlueGreen, Rolling, AllAtOnce
│   │   │   │                                # Tipos: Deployment, DeploymentTarget, HealthCheckConfig,
│   │   │   │                                #        RollbackPolicy, DeploymentMetrics, DeploymentStrategy
│   │   │   │
│   │   │   ├── 📄 model_registry_test.go    # ✅ Testes unitários
│   │   │   └── 📄 ab_testing_test.go        # ✅ Testes unitários
│   │   │
│   │   └── 📁 data/                         # Versionamento de dados
│   │       │                                # Versões de dados, schemas, linhagem, qualidade
│   │       ├── 📄 data_versioning.go        # ✅ Interface DataVersioning e InMemoryDataVersioning
│   │       │                                # Função: CreateVersion, GetVersion, ListVersions,
│   │       │                                #         GetLatestVersion, CreateSnapshot, GetSnapshot,
│   │       │                                #         ListSnapshots, TagVersion, DeleteVersion
│   │       │                                # Tipos: DataVersion, DataSnapshot, SnapshotType
│   │       │
│   │       ├── 📄 schema_migration.go       # ✅ Interface SchemaMigrationEngine e InMemorySchemaMigrationEngine
│   │       │                                # Função: CreateMigration, GetMigration, ListMigrations,
│   │       │                                #         ExecuteMigration, RollbackMigration, ValidateMigration
│   │       │                                # Tipos: SchemaMigration, MigrationStep, StepType, MigrationStatus
│   │       │
│   │       ├── 📄 data_lineage.go           # ✅ Interface DataLineageTracker e InMemoryDataLineageTracker
│   │       │                                # Função: RecordLineage, GetLineage, TraceUpstream,
│   │       │                                #         TraceDownstream, AddTransformation
│   │       │                                # Tipos: DataLineage, LineageNode, Transformation,
│   │       │                                #        NodeType, TransformationType
│   │       │
│   │       ├── 📄 data_quality.go          # ✅ Interface DataQuality e InMemoryDataQuality
│   │       │                                # Função: RunCheck, GetCheck, ListChecks, ValidateVersion,
│   │       │                                #         GetQualityScore
│   │       │                                # Tipos: QualityCheck, CheckType, CheckStatus, QualityResult,
│   │       │                                #        QualityIssue, ValidationResult, IssueSeverity
│   │       │
│   │       └── 📄 data_versioning_test.go  # ✅ Testes unitários
│   │
│   ├── 📁 services/                         # BLOCO-3: Application Services
│   │   │                                    # Serviços de aplicação que orquestram casos de uso
│   │   │
│   │   ├── 📄 mcp_service.go                # Serviço de aplicação MCP
│   │   ├── 📄 knowledge_service.go          # Serviço de aplicação Knowledge
│   │   ├── 📄 versioning_service.go         # Serviço de aplicação Versioning
│   │   └── 📄 monitoring_service.go        # Serviço de aplicação Monitoring
│   │
│   └── 📁 security/                         # BLOCO-9: Security Layer (Defense in Depth)
│       │                                    # Sistema imunológico do MCP-HULK
│       │                                    # Cross-Cutting Concern: Auth, RBAC, Encryption
│       │
│       ├── 📁 auth/                         # Autenticação e Autorização
│       │   │                                # Barreira 1: Identidade (Auth, JWT, OAuth)
│       │   ├── 📄 auth_manager.go          # ✅ AuthManager: Login, Register, ValidateToken, Logout
│       │   │                                # Função: Authenticate, Register, ValidateToken, HasPermission, Logout
│       │   │                                # Integração: TokenManager, SessionManager, RBACManager
│       │   ├── 📄 auth_manager_test.go     # ✅ Testes unitários
│       │   │
│       │   ├── 📄 token_manager.go         # ✅ TokenManager: JWT tokens (HS256/RS256)
│       │   │                                # Função: Generate, Validate, Refresh, Revoke
│       │   │                                # Suporte: HS256, RS256, Claims customizados, Revocation list
│       │   ├── 📄 token_manager_test.go    # ✅ Testes unitários
│       │   │
│       │   ├── 📄 session_manager.go       # ✅ SessionManager: Gestão de sessões
│       │   │                                # Função: Create, Get, GetByUserID, Validate, Refresh, Invalidate, InvalidateAll
│       │   │                                # Features: Limite de sessões simultâneas, Expiração automática
│       │   ├── 📄 session_manager_test.go  # ✅ Testes unitários
│       │   ├── 📄 in_memory_session_store.go # ✅ InMemorySessionStore para testes
│       │   │
│       │   ├── 📄 oauth_provider.go        # ✅ OAuthProvider: OAuth2/OIDC
│       │   │                                # Providers: Google, GitHub, Azure AD, Auth0
│       │   │                                # Função: GetAuthURL, ExchangeCode, GetUserInfo
│       │   ├── 📄 oauth_manager_test.go     # ✅ Testes unitários
│       │   ├── 📄 oauth_provider_google_test.go   # ✅ Testes Google OAuth
│       │   ├── 📄 oauth_provider_github_test.go   # ✅ Testes GitHub OAuth
│       │   ├── 📄 oauth_provider_azuread_test.go  # ✅ Testes Azure AD OAuth
│       │   ├── 📄 oauth_provider_auth0_test.go    # ✅ Testes Auth0 OAuth
│       │   └── 📄 oauth_auth0_example.go   # ✅ Exemplo Auth0
│       │
│       ├── 📁 encryption/                   # Criptografia e Gestão de Chaves
│       │   │                                # Barreira 3: Proteção de Dados
│       │   ├── 📄 encryption_manager.go     # ✅ EncryptionManager: AES-256-GCM, RSA, bcrypt, Argon2
│       │   │                                # Função: Encrypt, Decrypt, EncryptWithKey, DecryptWithKey
│       │   │                                # Função: HashPassword, VerifyPassword, HashArgon2, Sign, Verify
│       │   ├── 📄 encryption_manager_test.go # ✅ Testes unitários
│       │   │
│       │   ├── 📄 key_manager.go            # ✅ KeyManager: Gestão e rotação de chaves
│       │   │                                # Função: GetEncryptionKey, GetKeyVersion, RotateKey
│       │   │                                # Função: GetRSAPrivateKey, GetRSAPublicKey
│       │   │                                # Função: LoadKeyFromEnv, LoadKeyFromFile (✅ Implementado)
│       │   │                                # Features: Rotação automática, Thread-safe, Export PEM
│       │   │
│       │   ├── 📄 certificate_manager.go    # ✅ CertificateManager: Certificados TLS
│       │   │                                # Função: GetTLSCertificate, GenerateSelfSignedCert
│       │   │                                # Função: LoadCertificateFromFile, RotateCertificate, GetCertificateExpiry
│       │   │                                # Features: Rotação automática, Parsing X.509
│       │   │
│       │   └── 📄 secure_storage.go         # ✅ SecureStorage: Armazenamento seguro de segredos
│       │       │                                # Função: Store, Retrieve, Delete, Exists, List
│       │       │                                # Features: Encrypt-before-write, Decrypt-on-read
│       │       │                                # Backend: Abstrato (permite Redis/DB), InMemoryBackend para testes
│       │
│       ├── 📁 rbac/                         # RBAC e Policies
│       │   │                                # Barreira 2: Autorização (RBAC, Policies)
│       │   ├── 📄 rbac_manager.go           # ✅ RBACManager: Role-Based Access Control
│       │   │                                # Função: HasPermission, AssignRole, RevokeRole, GetUserRoles
│       │   │                                # Função: CreateRole, GetRole, ListRoles
│       │   │                                # Integração: RoleManager, PermissionChecker, PolicyEnforcer
│       │   ├── 📄 rbac_manager_test.go      # ✅ Testes unitários
│       │   │
│       │   ├── 📄 role_manager.go           # ✅ RoleManager: CRUD de Roles
│       │   │                                # Função: CreateRole, UpdateRole, DeleteRole, GetRole, ListRoles, Sync
│       │   │                                # Features: RoleStore abstrato, InMemoryRoleStore para testes
│       │   │
│       │   ├── 📄 permission_checker.go     # ✅ PermissionChecker: Verificação granular
│       │   │                                # Função: HasPermission, RegisterOverride, ListOverrides
│       │   │                                # Features: Pattern matching, Overrides, Condições customizadas
│       │   │
│       │   ├── 📄 policy_enforcer.go        # ✅ PolicyEnforcer: Políticas complexas
│       │   │                                # Função: Register, Remove, Evaluate, List, Clear
│       │   │                                # Features: Priorização, Condições (Role, Tenant, Attribute, TimeWindow)
│       │   │                                # Policies: "Somente admin pode deletar MCP", "Tenants isolados", etc.
│       │   │
│       │   ├── 📄 matcher.go                # ✅ Pattern matching para recursos/ações
│       │   └── 📄 effects.go                # ✅ PolicyEffect (Allow/Deny)
│       │
│       └── 📁 config/                       # Configuração de Segurança
│           │                                # Carregamento de configs (YAML, ENV)
│           ├── 📄 loader.go                 # ✅ Loader de configuração
│           │                                # Função: Load, resolveEnvVars, resolveEnvVar
│           │                                # Features: Suporte YAML, Variáveis de ambiente, Placeholders
│           ├── 📄 loader_test.go            # ✅ Testes unitários
│           ├── 📄 types.go                 # ✅ Tipos de configuração
│           └── 📄 integration.go           # ✅ Integração com outros blocos
│
├── 📁 pkg/                                  # BLOCO-1: Public Libraries
│   │                                        # Bibliotecas públicas reutilizáveis (exportadas)
│   │                                        # Podem ser usadas por outros projetos
│   │
│   ├── 📁 logger/                           # Logger estruturado (Zap)
│   │   │                                    # Sistema de logging com trace_id e span_id
│   │   ├── 📄 logger.go                     # Logger principal
│   │   │                                    # Função: Init, WithContext, Info, Debug, Warn, Error, Fatal, Sync
│   │   │                                    # Integração com OpenTelemetry para trace_id e span_id
│   │   ├── 📄 fields.go                     # Helpers para campos de log
│   │   │                                    # Função: String, Int, ErrorField, Any
│   │   └── 📄 levels.go                     # Níveis de log
│   │       │                                # Função: SetLevel
│   │       │                                # LogLevel: LevelDebug, LevelInfo, LevelWarn, LevelError
│   │
│   ├── 📁 mcp/                              # Utilitários MCP
│   │   └── 📄 mcp.go                        # Utilitários públicos do protocolo MCP
│   │       │                                # Tipos e utilitários para o protocolo MCP
│   │
│   ├── 📁 knowledge/                        # Utilitários de conhecimento
│   │   ├── 📄 knowledge.go                  # Utilitários públicos de conhecimento
│   │   │                                    # Interface e tipos principais para knowledge base
│   │   └── 📄 store.go                      # Armazenamento de conhecimento
│   │       │                                # Armazenamento de documentos e embeddings
│   │
│   ├── 📁 glm/                              # Cliente GLM
│   │   │                                    # Cliente para modelos GLM (ChatGLM, GLM-4.6)
│   │   ├── 📄 client.go                     # Cliente GLM
│   │   │                                    # Função: NewClient, Chat, Generate, Embed
│   │   └── 📄 glm.go                        # Tipos e estruturas GLM
│   │       │                                # Tipos para requisições e respostas GLM
│   │
│   ├── 📁 httpserver/                       # Servidor HTTP utilitário
│   │   │                                    # Utilitários para servidor HTTP com Echo
│   │   ├── 📄 server.go                     # Servidor HTTP principal
│   │   │                                    # Função: NewServer, Start, Stop, RegisterRoute, GetEcho
│   │   │                                    # Server: Servidor HTTP com Echo, métricas Prometheus, health checks
│   │   │                                    # Middlewares: OpenTelemetry, logging, metrics, CORS, recovery
│   │   └── 📄 server_test.go                # Testes unitários do servidor HTTP
│   │
│   ├── 📁 validator/                        # Validador público
│   │   └── 📄 validator.go                  # Validador genérico
│   │       │                                # Funções de validação reutilizáveis
│   │
│   ├── 📁 optimizer/                        # Otimizador público
│   │   └── 📄 optimizer.go                  # Otimizador genérico
│   │       │                                # Funções de otimização reutilizáveis
│   │
│   └── 📁 profiler/                         # Profiler público
│       └── 📄 profiler.go                   # Profiler de performance
│           │                                # Funções de profiling reutilizáveis
│
├── 📁 templates/                            # BLOCO-10: Templates
│   │                                        # Templates para geração de código MCP
│   │                                        # Suporta múltiplas stacks (Go, TinyGo, WASM, Web)
│   │
│   ├── 📁 base/                             # Template base
│   │   │                                    # Template base comum a todos
│   │   ├── 📄 structure.yaml.tmpl           # Estrutura base
│   │   ├── 📄 README.md.tmpl                # README base
│   │   └── 📄 manifest.yaml                # Manifesto do template
│   │
│   ├── 📁 go/                               # Template Go
│   │   │                                    # Template para projetos Go
│   │   ├── 📄 main.go.tmpl                  # main.go template
│   │   ├── 📄 handler.go.tmpl               # handler.go template
│   │   ├── 📄 service.go.tmpl               # service.go template
│   │   └── 📄 manifest.yaml                 # Manifesto do template Go
│   │
│   ├── 📁 tinygo/                           # Template TinyGo
│   │   │                                    # Template para projetos TinyGo (WebAssembly)
│   │   ├── 📄 main.go.tmpl                  # main.go template TinyGo
│   │   └── 📄 manifest.yaml                 # Manifesto do template TinyGo
│   │
│   ├── 📁 wasm/                             # Template WebAssembly
│   │   │                                    # Template para projetos WASM (Rust/Go)
│   │   ├── 📄 main.rs.tmpl                  # main.rs template (Rust)
│   │   └── 📄 manifest.yaml                 # Manifesto do template WASM
│   │
│   ├── 📁 base/                             # BLOCO-10: Template Clean Architecture Base
│   │   │                                    # Template genérico para qualquer stack
│   │   │                                    # Estrutura canônica mínima do Hulk
│   │   ├── 📄 manifest.yaml                 # Metadados do template base
│   │   ├── 📄 README.md.tmpl                # Documentação do template base
│   │   ├── 📄 CHANGELOG.md.tmpl             # Histórico de mudanças
│   │   └── 📄 structure.yaml.tmpl           # Estrutura de diretórios Clean Architecture
│   │
│   ├── 📁 go/                               # BLOCO-10: Template Go Premium
│   │   │                                    # Template Go com Clean Architecture avançada
│   │   │                                    # Echo, Zap, Viper, Docker multi-stage
│   │   ├── 📄 manifest.yaml                 # Metadados do template Go
│   │   ├── 📄 README.md.tmpl                # Documentação do template Go
│   │   ├── 📄 CHANGELOG.md.tmpl             # Histórico de mudanças
│   │   ├── 📄 go.mod.tmpl                   # go.mod template com placeholders
│   │   ├── 📄 Dockerfile.tmpl                # Dockerfile multi-stage
│   │   ├── 📄 docker-compose.yaml.tmpl      # Docker Compose para desenvolvimento
│   │   ├── 📁 cmd/server/
│   │   │   └── 📄 main.go.tmpl              # Ponto de entrada HTTP com Echo
│   │   └── 📁 internal/
│   │       ├── 📁 config/
│   │       │   └── 📄 config.go.tmpl        # Configuração centralizada (Viper)
│   │       ├── 📁 domain/
│   │       │   └── 📄 entities.go.tmpl        # Entidades de domínio
│   │       ├── 📁 application/
│   │       │   └── 📄 usecases.tmpl         # Casos de uso
│   │       ├── 📁 infrastructure/
│   │       │   └── 📄 repositories.tmpl       # Repositórios
│   │       └── 📁 interfaces/
│   │           └── 📄 handlers.tmpl           # Handlers HTTP
│   │
│   ├── 📁 tinygo/                           # BLOCO-10: Template TinyGo WASM
│   │   │                                    # Template para módulos WASM (edge/browser/IoT)
│   │   │                                    # Funções exportadas WASM
│   │   ├── 📄 manifest.yaml                 # Metadados do template TinyGo
│   │   ├── 📄 README.md.tmpl                # Documentação do template TinyGo
│   │   ├── 📄 CHANGELOG.md.tmpl             # Histórico de mudanças
│   │   ├── 📄 go.mod.tmpl                   # go.mod template
│   │   ├── 📄 main.go.tmpl                  # Funções WASM exportadas (SetMetric, GetMetric)
│   │   ├── 📁 cmd/__NAME__/
│   │   │   └── 📄 main.go                   # Runner de testes locais (placeholder __NAME__)
│   │   └── 📁 wasm/
│   │       └── 📄 exports.go.tmpl           # Utilitários de memória/echo WASM
│   │
│   ├── 📁 web/                              # BLOCO-10: Template Web React/Vite
│   │   │                                    # Template frontend moderno com React + TypeScript
│   │   │                                    # Dashboard completo de monitoramento
│   │   ├── 📄 manifest.yaml                 # Metadados do template Web
│   │   ├── 📄 README.md.tmpl                # Documentação do template Web
│   │   ├── 📄 CHANGELOG.md.tmpl             # Histórico de mudanças
│   │   ├── 📄 IMPLEMENTACAO.md               # Documentação de implementação do dashboard
│   │   ├── 📄 package.json.tmpl              # Dependências npm
│   │   ├── 📄 vite.config.ts.tmpl            # Configuração Vite
│   │   ├── 📄 index.html.tmpl                # HTML base
│   │   ├── 📄 tailwind.config.js            # Configuração Tailwind CSS
│   │   ├── 📄 tsconfig.json                 # Configuração TypeScript
│   │   ├── 📄 postcss.config.js             # Configuração PostCSS
│   │   ├── 📁 public/
│   │   │   └── 📄 manifest.json.tmpl         # Manifest PWA
│   │   └── 📁 src/
│   │       ├── 📄 main.tsx.tmpl              # Entry point React
│   │       ├── 📄 App.tsx.tmpl               # Componente principal
│   │       ├── 📄 index.css                  # Estilos globais
│   │       ├── 📁 components/
│   │       │   ├── 📁 charts/               # Componentes de gráficos
│   │       │   │   ├── 📄 LineChart.tsx
│   │       │   │   └── 📄 CacheHitChart.tsx
│   │       │   ├── 📁 layouts/              # Componentes de layout
│   │       │   │   └── 📄 Header.tsx
│   │       │   ├── 📁 sections/             # Seções do dashboard
│   │       │   │   ├── 📄 MetricsSection.tsx
│   │       │   │   ├── 📄 ComponentStatusSection.tsx
│   │       │   │   ├── 📄 AlertsSection.tsx
│   │       │   │   ├── 📄 ComponentTabs.tsx
│   │       │   │   ├── 📄 PerformanceCharts.tsx
│   │       │   │   └── 📄 QuickControls.tsx
│   │       │   └── 📁 ui/                   # Componentes UI reutilizáveis
│   │       │       ├── 📄 MetricCard.tsx
│   │       │       └── 📄 ComponentStatusCard.tsx
│   │       ├── 📁 hooks/                    # Custom hooks
│   │       │   ├── 📄 useMetrics.ts
│   │       │   └── 📄 useChartData.ts
│   │       └── 📁 types/                    # Definições TypeScript
│   │           └── 📄 index.ts
│   │
│   ├── 📁 wasm/                             # BLOCO-10: Template Rust WASM
│   │   │                                    # Template Rust com wasm-bindgen
│   │   │                                    # Alta performance para browser
│   │   ├── 📄 manifest.yaml                 # Metadados do template WASM
│   │   ├── 📄 README.md.tmpl                # Documentação do template WASM
│   │   ├── 📄 CHANGELOG.md.tmpl             # Histórico de mudanças
│   │   ├── 📄 Cargo.toml.tmpl               # Cargo.toml com placeholders
│   │   ├── 📄 build.sh                      # Script de build wasm-pack
│   │   └── 📁 src/
│   │       └── 📄 lib.rs.tmpl               # Funções WASM exportadas (update_metric, ping)
│   │
│   ├── 📁 mcp-go-premium/                  # BLOCO-10: Template MCP Go Premium
│   │   │                                    # Template completo com todas funcionalidades
│   │   │                                    # Integra: AI, State, Monitoring, Infra, Interfaces
│   │   ├── 📄 manifest.yaml                 # Metadados do template MCP Premium
│   │   ├── 📄 README.md.tmpl                # Documentação do template MCP Premium
│   │   ├── 📄 CHANGELOG.md.tmpl             # Histórico de mudanças
│   │   ├── 📄 go.mod.tmpl                   # go.mod template
│   │   ├── 📄 Makefile                      # Makefile com comandos úteis
│   │   ├── 📁 configs/
│   │   │   └── 📄 dev.yaml.tmpl             # Configuração desenvolvimento
│   │   ├── 📁 cmd/
│   │   │   └── 📄 main.go.tmpl              # Ponto de entrada com integrações completas
│   │   └── 📁 internal/
│   │       ├── 📁 ai/                       # Integração Bloco-6 (AI)
│   │       │   ├── 📁 agents/
│   │       │   │   └── 📄 agent.go.tmpl    # Agentes de IA
│   │       │   ├── 📁 core/
│   │       │   │   └── 📄 orchestrator.go.tmpl # Orquestrador de IA
│   │       │   └── 📁 rag/
│   │       │       └── 📄 ingestion.go.tmpl  # Ingestão RAG
│   │       ├── 📁 core/                     # Core engine e cache
│   │       │   ├── 📁 cache/
│   │       │   │   └── 📄 cache.go.tmpl     # Sistema de cache
│   │       │   └── 📁 engine/
│   │       │       └── 📄 engine.go.tmpl    # Motor de execução
│   │       ├── 📁 infrastructure/            # Integração Bloco-7 (Infra)
│   │       │   └── 📁 http/
│   │       │       └── 📄 server.go.tmpl    # Servidor HTTP
│   │       ├── 📁 interfaces/               # Integração Bloco-8 (Interfaces)
│   │       │   └── 📁 http/
│   │       │       └── 📄 handlers.go.tmpl  # Handlers HTTP
│   │       ├── 📁 monitoring/                # Integração Bloco-4 (Monitoring)
│   │       │   └── 📄 telemetry.go.tmpl     # Telemetria OpenTelemetry
│   │       └── 📁 state/                    # Integração Bloco-3 (State)
│   │           └── 📄 store.go.tmpl         # Store de estado
│   │
│   ├── 📁 k8s/                              # BLOCO-10: Templates Kubernetes
│   │   │                                    # Manifests Kubernetes completos para deploy
│   │   │                                    # Integração Bloco-7 (Infra)
│   │   ├── 📄 manifest.yaml                 # Metadados dos templates K8s
│   │   ├── 📄 Chart.yaml.tmpl               # Helm Chart
│   │   ├── 📄 values.yaml.tmpl              # Valores do Helm Chart
│   │   ├── 📄 deployment.yaml.tmpl          # Deployment Kubernetes
│   │   ├── 📄 service.yaml.tmpl             # Service Kubernetes
│   │   ├── 📄 ingress.yaml.tmpl             # Ingress Kubernetes
│   │   ├── 📄 configmap.yaml.tmpl           # ConfigMap Kubernetes
│   │   ├── 📄 secret.yaml.tmpl              # Secret Kubernetes
│   │   └── 📄 hpa.yaml.tmpl                 # Horizontal Pod Autoscaler
│   │
│   ├── 📁 docker-compose/                   # BLOCO-10: Templates Docker Compose
│   │   │                                    # Docker Compose para diferentes ambientes
│   │   │                                    # Integração Bloco-7 (Infra)
│   │   ├── 📄 manifest.yaml                 # Metadados dos templates Docker Compose
│   │   ├── 📄 docker-compose.yaml.tmpl      # Docker Compose base
│   │   ├── 📄 docker-compose.dev.yaml.tmpl  # Docker Compose desenvolvimento
│   │   └── 📄 docker-compose.prod.yaml.tmpl # Docker Compose produção
│   │
│   └── 📁 ci-cd/                            # BLOCO-10: Templates CI/CD
│       │                                    # Templates para pipelines CI/CD
│       │                                    # Integração Bloco-7 (Infra)
│       ├── 📄 manifest.yaml                 # Metadados dos templates CI/CD
│       ├── 📄 azure-pipelines.yml.tmpl      # Azure Pipelines template
│       └── 📄 Jenkinsfile.tmpl              # Jenkinsfile template
│
├── 📁 tools/                                # BLOCO-11: Tools & Utilities
│   │                                        # Ferramentas de desenvolvimento e operação
│   │                                        # Geradores, validadores, deployers, analisadores
│   │
│   ├── 📁 generators/                       # Geradores
│   │   │                                    # Ferramentas para gerar código, configs, MCPs
│   │   ├── 📄 mcp_generator.go              # Gerador de MCPs
│   │   ├── 📄 template_generator.go         # Gerador de templates
│   │   ├── 📄 config_generator.go          # Gerador de configurações
│   │   └── 📄 code_generator.go            # Gerador de código
│   │
│   ├── 📁 validators/                       # Validadores
│   │   │                                    # Ferramentas para validar código, configs, MCPs
│   │   ├── 📄 mcp_validator.go             # Validador de MCPs
│   │   ├── 📄 template_validator.go         # Validador de templates
│   │   ├── 📄 config_validator.go          # Validador de configurações
│   │   └── 📄 code_validator.go            # Validador de código
│   │
│   ├── 📁 deployers/                        # Deployers
│   │   │                                    # Ferramentas para deploy em diferentes plataformas
│   │   ├── 📄 kubernetes_deployer.go       # Deployer Kubernetes
│   │   ├── 📄 docker_deployer.go           # Deployer Docker
│   │   ├── 📄 serverless_deployer.go       # Deployer Serverless
│   │   └── 📄 hybrid_deployer.go           # Deployer Híbrido
│   │
│   ├── 📁 analyzers/                        # Analisadores
│   │   │                                    # Ferramentas para análise de código e performance
│   │   ├── 📄 dependency_analyzer.go       # Analisador de dependências
│   │   ├── 📄 performance_analyzer.go      # Analisador de performance
│   │   ├── 📄 quality_analyzer.go           # Analisador de qualidade
│   │   └── 📄 security_analyzer.go         # Analisador de segurança
│   │
│   └── 📁 converters/                       # Conversores
│       │                                    # Ferramentas para converter entre formatos
│       ├── 📄 openapi_generator.go         # Gerador OpenAPI
│       ├── 📄 asyncapi_generator.go        # Gerador AsyncAPI
│       ├── 📄 schema_converter.js          # Conversor de schemas (JS)
│       └── 📄 nats_schema_generator.js     # Gerador de schemas NATS (JS)
│
├── 📁 scripts/                              # BLOCO-13: Scripts & Automation
│   │                                        # Scripts de automação para operação do sistema
│   │                                        # Orquestram ferramentas Go do Bloco-11
│   │
│   ├── 📁 setup/                            # Scripts de setup
│   │   │                                    # Provisionamento de infraestrutura e serviços
│   │   ├── 📄 setup_infrastructure.sh      # Setup de infraestrutura (DBs, Cache, Messaging)
│   │   ├── 📄 setup_ai_stack.sh            # Setup da stack de IA (LLMs, VectorDB, GraphDB)
│   │   ├── 📄 setup_monitoring.sh          # Setup de monitoramento (Prometheus, OTLP, Jaeger)
│   │   ├── 📄 setup_security.sh            # Setup de segurança (Auth, RBAC, KMS)
│   │   ├── 📄 setup_state_management.sh    # Setup de gerenciamento de estado
│   │   └── 📄 setup_versioning.sh           # Setup de versionamento
│   │
│   ├── 📁 deployment/                       # Scripts de deployment
│   │   │                                    # Deploy para diferentes plataformas
│   │   ├── 📄 deploy_kubernetes.sh         # Deploy para Kubernetes
│   │   ├── 📄 deploy_docker.sh             # Deploy Docker
│   │   ├── 📄 deploy_serverless.sh         # Deploy Serverless
│   │   ├── 📄 deploy_hybrid.sh             # Deploy Híbrido
│   │   └── 📄 rollback.sh                  # Rollback de deploy
│   │
│   ├── 📁 generation/                       # Scripts de geração
│   │   │                                    # Geração de MCPs, templates, configs, docs
│   │   ├── 📄 generate_mcp.sh              # Gerar projeto MCP
│   │   ├── 📄 generate_template.sh          # Gerar projeto de template
│   │   ├── 📄 generate_config.sh            # Gerar arquivos de configuração
│   │   ├── 📄 generate_docs.sh              # Gerar documentação
│   │   ├── 📄 generate_openapi.sh           # Gerar especificação OpenAPI
│   │   └── 📄 generate_asyncapi.sh           # Gerar especificação AsyncAPI
│   │
│   ├── 📁 validation/                       # Scripts de validação
│   │   │                                    # Validação de MCPs, templates, configs, infra
│   │   ├── 📄 validate_mcp.sh              # Validar projeto MCP
│   │   ├── 📄 validate_template.sh          # Validar template
│   │   ├── 📄 validate_config.sh           # Validar configuração
│   │   ├── 📄 validate_infrastructure.sh   # Validar infraestrutura
│   │   └── 📄 validate_security.sh          # Validar segurança
│   │
│   ├── 📁 optimization/                     # Scripts de otimização
│   │   │                                    # Otimização de performance, cache, DB, rede, IA
│   │   ├── 📄 optimize_performance.sh      # Otimizar performance geral
│   │   ├── 📄 optimize_cache.sh            # Otimizar cache
│   │   ├── 📄 optimize_database.sh         # Otimizar banco de dados
│   │   ├── 📄 optimize_network.sh          # Otimizar rede
│   │   └── 📄 optimize_ai_inference.sh     # Otimizar inferência de IA
│   │
│   ├── 📁 features/                         # Scripts de feature flags
│   │   │                                    # Controle de feature flags usando yq
│   │   ├── 📄 enable_feature.sh             # Habilitar feature flag
│   │   ├── 📄 disable_feature.sh           # Desabilitar feature flag
│   │   └── 📄 list_features.sh             # Listar feature flags
│   │
│   ├── 📁 migration/                        # Scripts de migração
│   │   │                                    # Migração de conhecimento, modelos, dados
│   │   ├── 📄 migrate_knowledge.sh          # Migrar conhecimento entre ambientes
│   │   ├── 📄 migrate_models.sh            # Migrar modelos entre ambientes
│   │   └── 📄 migrate_data.sh              # Migrar dados entre ambientes
│   │
│   └── 📁 maintenance/                      # Scripts de manutenção
│       │                                    # Backup, cleanup, health-check, updates
│       ├── 📄 backup.sh                    # Backup de dados
│       ├── 📄 cleanup.sh                   # Limpeza de recursos
│       ├── 📄 health_check.sh              # Health check do sistema
│       └── 📄 update_dependencies.sh       # Atualização de dependências
│
├── 📁 config/                               # BLOCO-12: Configuration
│   │                                        # Arquivos de configuração centralizados (YAML)
│   │                                        # Ordem de precedência: ENV > env.yaml > features.yaml > config.yaml > defaults
│   │
│   ├── 📄 config.yaml                       # Configuração principal do sistema
│   │                                        # Configurações gerais: server, engine, logging, telemetry
│   │
│   ├── 📄 features.yaml                     # Feature flags
│   │                                        # Flags de funcionalidades (external_gpu, audit_logging, etc.)
│   │
│   ├── 📁 environments/                     # Configurações por ambiente
│   │   │                                    # Configurações específicas de cada ambiente
│   │   ├── 📄 dev.yaml                      # Configuração de desenvolvimento
│   │   ├── 📄 staging.yaml                  # Configuração de staging
│   │   ├── 📄 prod.yaml                     # Configuração de produção
│   │   └── 📄 test.yaml                     # Configuração de testes
│   │
│   ├── 📁 core/                             # Configurações do core
│   │   │                                    # Configurações do motor de execução
│   │   ├── 📄 engine.yaml                   # Configuração do engine
│   │   ├── 📄 engine_cache.yaml             # Configuração de cache do engine
│   │   ├── 📄 metrics.yaml                  # Configuração de métricas
│   │   └── 📄 runtime_security.yaml        # Configuração de segurança em runtime
│   │
│   ├── 📁 mcp/                              # Configurações MCP
│   │   │                                    # Configurações do protocolo MCP
│   │   ├── 📄 protocol.yaml                 # Configuração do protocolo
│   │   ├── 📄 registry.yaml                 # Configuração do registry
│   │   └── 📄 tools.yaml                    # Configuração de tools MCP
│   │
│   ├── 📁 ai/                               # Configurações de IA
│   │   │                                    # Configurações de modelos, conhecimento, memória
│   │   ├── 📄 models.yaml                   # Configuração de modelos de IA
│   │   ├── 📄 knowledge.yaml                # Configuração de conhecimento
│   │   ├── 📄 memory.yaml                   # Configuração de memória
│   │   └── 📄 learning.yaml                 # Configuração de aprendizado
│   │
│   ├── 📁 infrastructure/                   # Configurações de infraestrutura
│   │   │                                    # Configurações de cloud, compute, messaging, storage
│   │   ├── 📄 cloud.yaml                    # Configuração de cloud
│   │   ├── 📄 compute.yaml                  # Configuração de compute
│   │   ├── 📄 messaging.yaml                # Configuração de mensageria (NATS)
│   │   ├── 📄 network.yaml                  # Configuração de rede
│   │   └── 📄 storage.yaml                  # Configuração de armazenamento
│   │
│   ├── 📁 security/                         # Configurações de segurança
│   │   │                                    # Configurações de auth, RBAC, encryption
│   │   ├── 📄 auth.yaml                     # Configuração de autenticação
│   │   ├── 📄 rbac.yaml                     # Configuração de RBAC
│   │   ├── 📄 encryption.yaml               # Configuração de criptografia
│   │   └── 📄 compliance.yaml               # Configuração de compliance
│   │
│   ├── 📁 state/                            # Configurações de estado
│   │   │                                    # Configurações de event sourcing, cache, store
│   │   ├── 📄 store.yaml                    # Configuração do store de estado
│   │   ├── 📄 events.yaml                   # Configuração de eventos
│   │   └── 📄 state_cache.yaml              # Configuração de cache de estado
│   │
│   ├── 📁 monitoring/                      # Configurações de monitoramento
│   │   │                                    # Configurações de observability, alerting, analytics
│   │   ├── 📄 observability.yaml            # Configuração de observabilidade
│   │   ├── 📄 alerting.yaml                 # Configuração de alertas
│   │   ├── 📄 analytics.yaml                 # Configuração de analytics
│   │   └── 📄 health.yaml                   # Configuração de health checks
│   │
│   ├── 📁 versioning/                       # Configurações de versionamento
│   │   │                                    # Configurações de versionamento de conhecimento, modelos, dados
│   │   ├── 📄 knowledge.yaml                # Configuração de versionamento de conhecimento
│   │   ├── 📄 models.yaml                   # Configuração de versionamento de modelos
│   │   └── 📄 data.yaml                     # Configuração de versionamento de dados
│   │
│   ├── 📁 templates/                        # Configurações de templates
│   │   │                                    # Configurações de templates disponíveis
│   │   ├── 📄 base.yaml                     # Configuração do template base
│   │   ├── 📄 go.yaml                       # Configuração do template Go
│   │   ├── 📄 tinygo.yaml                   # Configuração do template TinyGo
│   │   ├── 📄 wasm.yaml                     # Configuração do template WASM
│   │   └── 📄 web.yaml                      # Configuração do template Web
│   │
│   └── 📄 README.md                         # Documentação de configuração
│
├── 📁 docs/                                 # BLOCO-14: Documentation Layer
│   │                                        # Documentação completa do sistema
│   │                                        # Fonte de verdade conceitual do ecossistema Hulk
│   │
│   ├── 📁 architecture/                     # Documentação de arquitetura
│   │   │                                    # Arquitetura geral, Clean Architecture, fluxos
│   │   ├── 📄 blueprint.md                  # Blueprint geral (Blocos 1-13)
│   │   ├── 📄 clean_architecture.md         # Clean Architecture Hulk
│   │   ├── 📄 mcp_flow.md                   # Fluxo do protocolo MCP
│   │   ├── 📄 compute_architecture.md       # Arquitetura de compute
│   │   ├── 📄 hybrid_compute.md             # Compute híbrido (CPU local + GPU externa)
│   │   ├── 📄 performance.md                # Performance e otimizações
│   │   ├── 📄 scalability.md                # Escalabilidade
│   │   ├── 📄 reliability.md                # Confiabilidade
│   │   └── 📄 security.md                   # Segurança
│   │
│   ├── 📁 mcp/                              # Documentação MCP
│   │   │                                    # Protocolo, tools, handlers, registry, schema
│   │   ├── 📄 protocol.md                   # Protocolo MCP (JSON-RPC 2.0)
│   │   ├── 📄 tools.md                      # Tools MCP disponíveis
│   │   ├── 📄 handlers.md                    # Handlers MCP
│   │   ├── 📄 registry.md                   # Registry de MCPs
│   │   ├── 📄 schema.md                     # Schema do protocolo MCP
│   │   └── 📄 lifecycle.md                  # Ciclo de vida de MCPs
│   │
│   ├── 📁 ai/                               # Documentação de IA
│   │   │                                    # RAG, memória, fine-tuning, prompts
│   │   ├── 📄 rag.md                        # Retrieval-Augmented Generation
│   │   ├── 📄 memory_management.md          # Gerenciamento de memória
│   │   ├── 📄 knowledge_management.md        # Gerenciamento de conhecimento
│   │   ├── 📄 finetuning_runpod.md          # Fine-tuning com RunPod
│   │   ├── 📄 learning.md                    # Aprendizado de máquina
│   │   ├── 📄 prompts.md                    # Sistema de prompts
│   │   ├── 📄 integration.md                # Integração de IA
│   │   └── 📄 specialists.md                # Especialistas de IA
│   │
│   ├── 📁 state/                            # Documentação de estado
│   │   │                                    # Event sourcing, projections, conflict resolution, caching
│   │   ├── 📄 distributed_state.md          # Estado distribuído
│   │   ├── 📄 event_sourcing.md            # Event sourcing
│   │   ├── 📄 projections.md                # Projeções (projections)
│   │   ├── 📄 conflict_resolution.md        # Resolução de conflitos
│   │   ├── 📄 caching.md                    # Cache de estado
│   │   └── 📄 state_sync.md                 # Sincronização de estado
│   │
│   ├── 📁 monitoring/                       # Documentação de monitoramento
│   │   │                                    # Logs, métricas, tracing, dashboards, alerting
│   │   ├── 📄 observability.md              # Observabilidade geral
│   │   ├── 📄 logs.md                       # Sistema de logs
│   │   ├── 📄 metrics.md                    # Métricas (Prometheus)
│   │   ├── 📄 tracing.md                    # Tracing (OpenTelemetry, Jaeger)
│   │   ├── 📄 dashboards.md                 # Dashboards
│   │   ├── 📄 alerting.md                   # Sistema de alertas
│   │   ├── 📄 analytics.md                  # Analytics
│   │   └── 📄 health_check.md               # Health checks
│   │
│   ├── 📁 versioning/                       # Documentação de versionamento
│   │   │                                    # Versionamento de conhecimento, modelos, dados, migrações
│   │   ├── 📄 knowledge_versioning.md       # Versionamento de conhecimento
│   │   ├── 📄 model_versioning.md           # Versionamento de modelos
│   │   ├── 📄 data_versioning.md            # Versionamento de dados
│   │   ├── 📄 migrations.md                 # Migrações
│   │   ├── 📄 workflow.md                   # Workflow de versionamento
│   │   └── 📄 compute_asset_versioning.md   # Versionamento de assets de compute
│   │
│   ├── 📁 api/                              # Documentação de API
│   │   │                                    # OpenAPI, AsyncAPI, gRPC
│   │   ├── 📄 openapi.md                    # Documentação OpenAPI (HTTP REST)
│   │   ├── 📄 openapi.yaml                  # Especificação OpenAPI (YAML)
│   │   ├── 📄 asyncapi.md                   # Documentação AsyncAPI (Eventos)
│   │   ├── 📄 asyncapi.yaml                 # Especificação AsyncAPI (YAML)
│   │   └── 📄 grpc.md                       # Documentação gRPC
│   │
│   ├── 📁 guides/                           # Guias de uso
│   │   │                                    # Guias práticos para desenvolvedores e operadores
│   │   ├── 📄 getting_started.md            # Guia de início rápido
│   │   ├── 📄 development.md                # Guia de desenvolvimento
│   │   ├── 📄 deployment.md                 # Guia de deployment
│   │   ├── 📄 cli.md                        # Guia da CLI
│   │   ├── 📄 configuration.md               # Guia de configuração
│   │   ├── 📄 ai_rag.md                     # Guia de RAG
│   │   ├── 📄 fine_tuning_cycle.md          # Ciclo de fine-tuning
│   │   ├── 📄 using_external_gpu.md         # Usando GPU externa (RunPod)
│   │   ├── 📄 troubleshooting.md            # Troubleshooting
│   │   ├── 📄 oauth_setup.md                # Setup de OAuth
│   │   ├── 📄 env_variables_reference.md    # Referência de variáveis de ambiente
│   │   └── 📄 workload_cost_control.md      # Controle de custos de workload
│   │
│   ├── 📁 examples/                         # Exemplos práticos
│   │   │                                    # Exemplos de código e uso
│   │   ├── 📄 mcp_example.md                # Exemplo de projeto MCP
│   │   ├── 📄 rag_example.md                # Exemplo de RAG
│   │   ├── 📄 ai_prompts.md                 # Exemplos de prompts de IA
│   │   ├── 📄 template_example.md           # Exemplo de template
│   │   ├── 📄 finetune_runpod_example.md    # Exemplo de fine-tuning RunPod
│   │   ├── 📄 order_flow.md                 # Exemplo de fluxo de pedidos
│   │   └── 📄 inventory_schema.json          # Schema de exemplo (JSON)
│   │
│   ├── 📁 validation/                      # Documentação de validação
│   │   │                                    # Critérios, relatórios, dados brutos
│   │   ├── 📄 criteria.md                   # Critérios de validação
│   │   ├── 📄 reports.md                    # Relatórios de validação
│   │   ├── 📄 raw.md                        # Dados brutos de validação
│   │   ├── 📁 reports/                      # Relatórios de validação (JSON)
│   │   └── 📁 raw/                          # Dados brutos de validação (JSON)
│   │
│   └── 📁 compute/                          # Documentação de compute
│       │                                    # Compute híbrido, RunPod, scheduling
│       ├── 📄 runpod_overview.md            # Visão geral do RunPod
│       ├── 📄 runpod_api.md                 # API do RunPod
│       ├── 📄 runpod_jobs.md                # Jobs no RunPod
│       ├── 📄 scheduling.md                 # Agendamento de compute
│       └── 📄 compute_security.md           # Segurança de compute
│
├── 📁 .crush/                               # Sistema CRUSH (Parallel Processing)
│   │                                        # Sistema de processamento paralelo otimizado
│   │
│   ├── 📄 init                              # Arquivo de inicialização (vazio - intencional)
│   ├── 📄 crush.db                          # Banco de dados do CRUSH
│   ├── 📁 commands/                         # Comandos do CRUSH
│   └── 📁 logs/                             # Logs do CRUSH
│       └── 📄 crush.log                     # Log do sistema CRUSH
│
├── 📁 .cursor/                              # Configurações e documentação do Cursor
│   │                                        # Documentação de blocos, blueprints, auditorias
│   │
│   ├── 📁 BLOCOS/                           # Blueprints e auditorias dos 14 blocos
│   │   │                                    # Documentação oficial de cada bloco
│   │   ├── 📄 BLOCO-1-BLUEPRINT.md          # Blueprint Bloco-1 (Core Platform)
│   │   ├── 📄 BLOCO-2-BLUEPRINT.md          # Blueprint Bloco-2 (MCP Protocol)
│   │   ├── 📄 BLOCO-5-BLUEPRINT.md          # Blueprint Bloco-5 (Versioning & Migration)
│   │   ├── 📄 BLOCO-5-BLUEPRINT-GLM-4.6.md # Blueprint executivo Bloco-5
│   │   ├── 📄 BLOCO-5-AUDITORIA-CONFORMIDADE-BLUEPRINT-IMPLEMENTACAO.md
│   │   │                                    # Auditoria de conformidade Bloco-5 (100% conforme)
│   │   ├── 📄 BLOCO-13-BLUEPRINT.md         # Blueprint Bloco-13 (Scripts & Automation)
│   │   ├── 📄 BLOCO-13-BLUEPRINT-GLM-4.6.md # Blueprint executivo Bloco-13
│   │   ├── 📄 BLOCO-13-AUDITORIA-CONFORMIDADE-BLUEPRINT-IMPLEMENTACAO.md
│   │   │                                    # Auditoria de conformidade Bloco-13
│   │   ├── 📄 BLOCO-14-AUDITORIA-CONFORMIDADE-BLUEPRINT-IMPLEMENTACAO.md
│   │   │                                    # Auditoria de conformidade Bloco-14
│   │   └── 📄 ...                           # Outros blueprints e auditorias
│   │
│   ├── 📄 MCP-HULK-ARVORE-FULL.md          # Árvore oficial completa do projeto
│   ├── 📄 MCP-HULK-INTEGRACOES.md          # Documentação de integrações entre blocos
│   ├── 📄 ANALISE-ARQUIVOS-VAZIOS.md        # Análise de arquivos vazios
│   └── 📄 ARVORE-ARQUIVOS-DIRETORIOS-COMENTADA.md # Este arquivo
│
├── 📄 go.mod                                # Módulo Go e dependências
│                                            # Define módulo: github.com/vertikon/mcp-hulk
│
├── 📄 go.sum                                # Checksums das dependências Go
│                                            # Garante integridade das dependências
│
├── 📄 Makefile                              # Makefile com comandos de build, test, lint
│                                            # Comandos: make build, make test, make lint, etc.
│
├── 📄 README.md                             # README principal do projeto
│                                            # Visão geral, features, estrutura, quick start
│
├── 📄 README-BLOCO-1.md                     # README específico do Bloco-1
│                                            # Documentação detalhada do Core Platform
│
├── 📄 CRUSH.md                              # Guia de desenvolvimento CRUSH
│                                            # Informações essenciais para agentes trabalhando no código
│
└── 📄 coverage                              # Arquivo de cobertura de testes
                                            # Gerado por: make test-coverage
```

---

## 🔷 LEGENDA DE SÍMBOLOS

- 📁 = Diretório
- 📄 = Arquivo
- ✅ = Implementado e completo
- ⚠️ = Parcialmente implementado
- ❌ = Não implementado
- 🔷 = Seção importante
- 📋 = Documentação

---

## 🔷 ESTATÍSTICAS DO PROJETO

### Por Tipo de Arquivo

- **Go Files (.go)**: ~300+ arquivos
- **Shell Scripts (.sh)**: 37 arquivos
- **YAML Configs (.yaml)**: ~30 arquivos
- **Markdown Docs (.md)**: ~60 arquivos
- **Templates (.tmpl)**: ~50 arquivos
- **JSON Schemas (.json)**: ~5 arquivos

### Por Diretório Principal

- **cmd/**: 8 executáveis principais
- **internal/**: ~300 arquivos Go (código privado)
- **pkg/**: ~15 arquivos Go (bibliotecas públicas)
- **scripts/**: 37 scripts de automação
- **templates/**: ~50 templates de geração
- **tools/**: ~20 ferramentas Go
- **config/**: ~30 arquivos de configuração
- **docs/**: ~60 arquivos de documentação

---

## 🔷 MAPEAMENTO DE BLOCOS PARA DIRETÓRIOS

| Bloco | Diretórios Principais | Descrição |
|-------|----------------------|-----------|
| **BLOCO-1** | `cmd/`, `internal/core/`, `internal/domain/`, `internal/application/`, `pkg/` | Core Platform |
| **BLOCO-2** | `internal/mcp/` | MCP Protocol |
| **BLOCO-3** | `internal/state/`, `internal/monitoring/`, `internal/services/` | State Management, Monitoring |
| **BLOCO-5** | `internal/versioning/` | Versioning & Migration |
| **BLOCO-6** | `internal/ai/` | AI Layer |
| **BLOCO-7** | `internal/infrastructure/` | Infrastructure Layer |
| **BLOCO-8** | `internal/interfaces/` | Interface Layer |
| **BLOCO-9** | `internal/security/` | Security Layer |
| **BLOCO-10** | `templates/` | Templates |
| **BLOCO-11** | `tools/` | Tools & Utilities |
| **BLOCO-12** | `config/` | Configuration |
| **BLOCO-13** | `scripts/` | Scripts & Automation |
| **BLOCO-14** | `docs/` | Documentation Layer |

---

## 🔷 REGRAS DE ESTRUTURA

### Diretórios Fixos (Não Podem Ser Criados Novos)

- `cmd/` - Application entry points
- `internal/` - Private application code
- `pkg/` - Public libraries
- `templates/` - Code generation templates
- `tools/` - Development tools
- `config/` - Configuration files
- `scripts/` - Automation scripts
- `docs/` - Documentation

### Nomenclatura

- **Diretórios**: lowercase, underscore se necessário (`internal/ai/core`)
- **Arquivos Go**: snake_case (`mcp_http_handler.go`)
- **Handlers**: sufixo com tipo (`*_http_handler.go`, `*_grpc_server.go`)
- **Repositórios**: `*_repository.go` (interfaces), `postgres_*_repository.go` (implementações)
- **Scripts**: categoria prefixada (`setup_*.sh`, `deploy_*.sh`)

---

## 🔷 DEPENDÊNCIAS PRINCIPAIS

### Core Runtime
- Echo v4 (HTTP server)
- NATS (Message broker)
- Viper (Configuration)
- Cobra (CLI)
- Zap (Logging)

### Observability
- OpenTelemetry (Tracing)
- Prometheus (Metrics)
- Jaeger (Trace visualization)

### Data & Storage
- Badger (Embedded KV store)
- PostgreSQL (Relational DB)
- Redis, MongoDB, Neo4j (Various clients)

### AI/ML
- Multiple LLM providers (OpenAI, Gemini, GLM)
- Vector databases (Qdrant, Pinecone, Weaviate)

---

## 🔷 NOTAS IMPORTANTES

1. **Clean Architecture**: O projeto segue rigorosamente Clean Architecture com separação de camadas
2. **14 Blocos**: Arquitetura dividida em 14 blocos funcionais bem definidos
3. **Fonte Única da Verdade**: A árvore oficial está em `.cursor/MCP-HULK-ARVORE-FULL.md`
4. **Política de Estrutura**: Regras rígidas de nomenclatura e organização em `.cursor/MCP-HULK – POLÍTICA DE ESTRUTURA & NOMENCLATURA.md`
5. **Documentação Completa**: Todos os 14 blocos têm blueprints e auditorias de conformidade

---

**Fim da Árvore Comentada**

**Última Atualização:** 2025-01-27  
**Versão:** 1.0

