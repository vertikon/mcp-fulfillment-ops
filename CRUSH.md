# CRUSH - mcp-fulfillment-ops Development Guide

This document contains essential information for agents working with the mcp-fulfillment-ops codebase.

## Project Overview

mcp-fulfillment-ops is a comprehensive Model Context Protocol (MCP) template and generation engine. It's a Go-based system that follows Clean Architecture principles with 14 main blocks covering everything from core execution engines to AI integration and monitoring.

The project has two main entry points:
- `cmd/main.go` - Core HTTP server with observability and execution engine
- `cmd/fulfillment-ops/main.go` - Fulfillment operations service with PostgreSQL, NATS, and Redis integration

## Development Commands

### Build & Run
```bash
# Build the main application
make build

# Run the main HTTP server with config loading
make run

# Run fulfillment-ops service directly
go run ./cmd/fulfillment-ops/main.go

# Install dependencies
make deps

# Development setup (installs golangci-lint)
make dev-setup

# Install locally
make install
```

### Testing
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration

# Run load tests (requires k6)
make test-load

# Run all checks (lint, security, test)
make check

# Prepare for release
make release
```

### Code Quality
```bash
# Run linter (requires golangci-lint)
make lint

# Format code
make fmt

# Vet code for potential issues
make vet

# Security scan (requires gosec)
make security
```

### Docker
```bash
# Build Docker image
make docker

# Run Docker container
make docker-run
```

### Code Generation
```bash
# Generate code (go generate)
make generate
```

## Project Structure

The project follows a strict Clean Architecture pattern:

```
├── cmd/                    # Application entry points (main.go files)
├── internal/               # Private application code
│   ├── core/              # Performance engine, config, scheduler
│   ├── ai/                # AI & knowledge management
│   ├── state/             # State management with event sourcing
│   ├── monitoring/        # Observability and health checks
│   ├── versioning/        # Version control for data/models/knowledge
│   ├── mcp/               # MCP protocol implementation & generators
│   ├── services/          # Application services layer
│   ├── domain/            # Domain entities, repositories, value objects
│   ├── application/       # Use cases and DTOs
│   ├── infrastructure/    # External integrations (databases, APIs)
│   ├── interfaces/        # Input/output adapters (HTTP, gRPC, CLI)
│   └── security/          # Authentication, authorization, encryption
├── pkg/                    # Public libraries
├── templates/              # Code generation templates
├── tools/                  # Development tools
├── config/                 # Configuration files
├── scripts/                # Automation scripts
└── docs/                   # Documentation
```

## Key Architectural Patterns

### Clean Architecture
- **Domain**: Core business logic (entities, value objects)
- **Application**: Use cases and application-specific rules
- **Infrastructure**: External concerns (databases, APIs)
- **Interfaces**: Adapters for input/output (HTTP, gRPC, CLI)

### MCP Protocol
The system implements the Model Context Protocol with:
- JSON-RPC 2.0 messaging
- Tool registration and execution
- Resource management
- Prompt templating

### Template Generation
Support for multiple technology stacks:
- **Go**: Standard Go backend services
- **TinyGo**: Go compiled to WebAssembly
- **Rust WASM**: Rust compiled to WebAssembly  
- **Web**: React/TypeScript frontend applications

## Configuration

Configuration is managed through YAML files in `config/` directory with two approaches:

1. **Main Application** (`cmd/main.go`): Uses Viper with HULK_ prefixed environment variables
2. **Fulfillment Service** (`cmd/fulfillment-ops/main.go`): Uses direct environment variables

### Primary Config
- `config/config.yaml` - Main application configuration
- Uses Viper for loading with environment variable overrides
- Environment variables use `HULK_` prefix (see config.yaml)
- For fulfillment-ops service: uses direct `getEnv()` with standard names

### Key Configuration Sections
- **Server**: HTTP server settings (port, timeouts)
- **Database**: PostgreSQL connection settings
- **AI**: LLM provider configuration (OpenAI, Gemini, GLM)
- **Engine**: Worker pool and execution settings
- **NATS**: Message broker configuration
- **Cache**: Multi-level caching settings
- **Logging**: Structured logging with Zap
- **Telemetry**: OpenTelemetry tracing and metrics

### Environment Variables
Key environment variables for fulfillment-ops service:
- `DATABASE_URL` - PostgreSQL connection string
- `NATS_URL` - NATS server URL (default: nats://localhost:4222)
- `REDIS_URL` - Redis connection string
- `CORE_INVENTORY_URL` - Core inventory service URL
- `HTTP_PORT` - Server port (default: :8080)

For main application, use `HULK_` prefixed variables as defined in config.

## Key Dependencies

### Core Runtime
- **Echo v4 & Gin**: HTTP server frameworks
- **NATS**: Message broker and streaming with JetStream
- **Viper**: Configuration management
- **Cobra**: CLI framework
- **Zap**: Structured logging
- **PostgreSQL**: Primary database (lib/pq driver)
- **Redis**: Caching and session storage

### Observability
- **OpenTelemetry**: Distributed tracing
- **Prometheus**: Metrics collection
- **Jaeger**: Trace visualization

### Auth & Security
- **JWT (golang-jwt/jwt)**: Token management
- **OAuth2 (golang.org/x/oauth2)**: OAuth provider integration
- **Crypto (golang.org/x/crypto)**: Encryption utilities

### AI/ML Integration
- Multiple LLM provider clients (OpenAI, Gemini, GLM)
- Vector database clients (Qdrant, Pinecone, Weaviate)
- Knowledge graph capabilities

### Infrastructure
- **Kubernetes**: Client-go for K8s integration
- **Badger**: Embedded key-value store
- **UUID**: Unique identifier generation

## Naming Conventions

### Files
- **Go files**: `snake_case.go` describing purpose
  - Examples: `mcp_http_handler.go`, `model_registry.go`, `state_snapshot.go`
- **Handlers**: Suffix with type (`*_http_handler.go`, `*_grpc_server.go`)
- **Repositories**: `*_repository.go` for interfaces, `postgres_*_repository.go` for implementations
- **Scripts**: Category prefixed (`setup_*.sh`, `deploy_*.sh`, `generate_*.sh`)

### Directories
- Lowercase English names with underscores if needed
- Fixed top-level directories: `internal/`, `pkg/`, `templates/`, `scripts/`, `config/`, `docs/`

### Code Style
- Follow Go standard conventions
- Use Zap for structured logging
- Error wrapping with context
- Interface-based design in domain layer

## Critical Rules (from .cursor policy)

### Forbidden
- Creating files/directories not in the official tree structure
- Renaming files/directories without updating documentation
- Generic names like `utils.go`, `helpers.go` in critical layers
- Creating unapproved top-level directories
- Editing template files directly in `templates/` for specific cases
- Deviating from the defined Clean Architecture layer separation

### Required
- All new artifacts must be documented in the official tree
- Standardized function comments for Go code
- Follow the Clean Architecture layer separation
- Use the HULK_ prefix for environment variables in main application
- Use direct environment variables for fulfillment-ops service

### Validation
- The project includes automated tree validation in CI/CD
- See `.gitlab-ci.yml` and `tools/validate_tree.go`
- Original tree reference: `.cursor/MCP-HULK-ARVORE-FULL.md`

## Testing

### Structure
- Unit tests: `*_test.go` alongside source files
- Integration tests: Organized by feature area
- Coverage reporting available via `make test-coverage`

### Key Test Patterns
- Table-driven tests for multiple scenarios
- Mock interfaces for infrastructure concerns
- Test fixtures in `testdata/` directories where needed

## Development Workflow

### Adding Features
1. Identify the appropriate layer (domain/application/infrastructure/interfaces)
2. Create domain entities and value objects first
3. Implement use cases in application layer
4. Add infrastructure implementations as needed
5. Create interface adapters (HTTP/gRPC/CLI)
6. Add tests at each layer

### Template Development
1. Create template in `templates/` with `manifest.yaml`
2. Implement generators in `internal/mcp/generators/`
3. Add validation logic in `internal/mcp/validators/`
4. Update template registry

### Configuration Changes
1. Update config structs in `internal/core/config/config.go`
2. Add defaults in `setDefaults()` method
3. Update `config/config.yaml` with new sections
4. Add validation rules where needed

### Release Pipeline
Use the automated PowerShell pipeline for releases:
```powershell
# Pipeline completo (validação, testes, build, deploy)
.\pipeline-release.ps1

# Pular testes (uso com cautela)
.\pipeline-release.ps1 -SkipTests

# Pular validação de estrutura
.\pipeline-release.ps1 -SkipValidation

# Fazer push automático para Git
.\pipeline-release.ps1 -PushToGit
```

The pipeline:
1. Validates project structure against .cursor rules
2. Runs unit and integration tests with coverage
3. Builds Windows and Linux binaries
4. Creates Docker image
5. Deploys via Docker Compose with health checks
6. Commits changes with detailed messages
7. Generates execution summaries

### Docker Compose Services
- **PostgreSQL**: `localhost:5435` (user: fulfillment, pass: fulfillment123)
- **NATS**: `localhost:4225` (monitoring: `localhost:8225`)
- **Redis**: `localhost:6381`
- **Fulfillment Ops**: `http://localhost:8082`

## Performance Considerations

### Execution Engine
- Worker pools with configurable sizes
- Circuit breakers for resilience
- Multi-level caching (L1/L2/L3)
- NATS JetStream for reliable messaging

### Memory Management
- Custom memory optimizer in AI layer
- Episodic, semantic, and working memory types
- Memory consolidation processes

### Caching Strategy
- L1: In-memory cache for hot data
- L2: Distributed cache (Redis)  
- L3: Persistent cache (Badger)

## Security Features

### Authentication & Authorization
- JWT token management
- OAuth 2.0 provider integrations
- Role-based access control (RBAC)
- Session management

### Encryption
- AES-256 encryption at rest and in transit
- Certificate management
- Secure key storage

### Compliance
- GDPR compliance features
- Audit logging
- Data retention policies

## Observability Stack

### Metrics
- Prometheus-compatible metrics
- Custom performance metrics
- Resource utilization tracking

### Tracing
- OpenTelemetry distributed tracing
- Jaeger integration
- Request correlation across services

### Logging
- Structured JSON logging with Zap
- Multiple log levels
- Context-aware logging

## Deployment Patterns

### Container Support
- Dockerfile for containerization
- Multi-stage builds for optimization
- Health check endpoints

### Kubernetes
- Deployment manifests in standard locations
- ConfigMap and Secret management
- Horizontal pod autoscaling support

### Serverless
- Support for AWS Lambda, Azure Functions
- Function orchestration capabilities

## CLI Usage

### Multiple Entry Points

The project provides multiple CLI entry points:

```bash
# Main HTTP server (starts core services with observability)
go run ./cmd/main.go

# Fulfillment operations service (domain-specific logic)
go run ./cmd/fulfillment-ops/main.go

# MCP Server implementation
go run ./cmd/mcp-server/main.go

# MCP CLI for template generation
go run ./cmd/mcp-cli/main.go

# MCP initialization tool
go run ./cmd/mcp-init/main.go

# Development tools
go run ./cmd/tools-generator/main.go
go run ./cmd/tools-validator/main.go
go run ./cmd/tools-deployer/main.go
go run ./cmd/thor/main.go
```

### Quick Commands
```bash
# Build all binaries
make build

# Run the main application
make run

# Test all components
make test

# Full pipeline: clean, lint, security, test, build
make release
```

## Troubleshooting

### Common Issues
- **NATS connection failed**: Check NATS server is running on localhost:4222
- **PostgreSQL connection failed**: Verify DATABASE_URL or individual DB env vars
- **Redis connection failed**: Service continues without cache, check REDIS_URL
- **Configuration not found**: Ensure `config/config.yaml` exists or set environment variables
- **Build failures**: Run `make deps` to ensure dependencies are current
- **Lint failures**: Install golangci-lint via `make dev-setup`

### Service Dependencies
Before running fulfillment-ops service, ensure:
1. PostgreSQL is accessible (connection string via DATABASE_URL)
2. NATS server is running (default: localhost:4222)
3. Redis is available (optional, for caching)
4. Core inventory service is running if needed

### Debug Mode
Set environment variables for debug output:
- Main app: `HULK_LOGGING_LEVEL=debug`
- Fulfillment ops: Configure logger settings or modify logger initialization

### Health Checks
Both services provide health endpoints for monitoring deployment status.

### Development Workflow
1. Use `make deps` to update Go modules
2. Use `make test` to run unit tests
3. Use `make test-integration` for integration tests
4. Use `make check` for full validation before commits
5. Use `.\pipeline-release.ps1` for complete release automation

### Pipeline Automation
The PowerShell pipeline script provides:
- **Structure validation**: Checks against .cursor tree rules
- **Automated testing**: Unit and integration tests with coverage
- **Multi-platform builds**: Windows and Linux binaries
- **Docker deployment**: Complete stack with health checks
- **Git operations**: Smart commits with detailed messages
- **Error handling**: Comprehensive error reporting and recovery

### Service Dependencies
Before running fulfillment-ops service, ensure:
1. PostgreSQL is accessible (connection string via DATABASE_URL)
2. NATS server is running (default: localhost:4222)
3. Redis is available (optional, for caching)
4. Core inventory service is running if needed

### Performance Monitoring
- Application metrics available at `/metrics` endpoint
- Health check at `/health` endpoint
- OpenTelemetry tracing if configured
- Docker Compose includes health checks for all services