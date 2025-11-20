# CRUSH - mcp-fulfillment-ops Development Guide

This document contains essential information for agents working with the mcp-fulfillment-ops codebase.

## Project Overview

mcp-fulfillment-ops is a comprehensive Model Context Protocol (MCP) template and generation engine. It's a Go-based system that follows Clean Architecture principles with 14 main blocks covering everything from core execution engines to AI integration and monitoring.

## Development Commands

### Build & Run
```bash
# Build the application
make build

# Run with default configuration
make run

# Install dependencies
make deps

# Build for production
make build-prod

# Development setup (installs golangci-lint)
make dev-setup
```

### Testing
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run all checks (lint, vet, test)
make check
```

### Code Quality
```bash
# Run linter (requires golangci-lint)
make lint

# Format code
make fmt

# Vet code for potential issues
make vet
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

# Install locally
make install
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

Configuration is managed through YAML files in `config/` directory:

### Primary Config
- `config/config.yaml` - Main application configuration
- Uses Viper for loading with environment variable overrides
- Prefix: `MCP_HULK_` for environment variables

### Key Configuration Sections
- **Server**: HTTP server settings (port, timeouts)
- **Engine**: Worker pool and execution settings
- **NATS**: Message broker configuration
- **Logging**: Structured logging with Zap
- **Telemetry**: OpenTelemetry tracing and metrics
- **MCP**: Protocol-specific settings

### Default Values
The system provides sensible defaults in `internal/core/config/config.go`:
- Server port: 8080
- Workers: "auto" (NumCPU * 2)
- Queue size: 2000
- Cache L1 size: 5000 entries

## Key Dependencies

### Core Runtime
- **Echo v4**: HTTP server framework
- **NATS**: Message broker and streaming
- **Viper**: Configuration management
- **Cobra**: CLI framework
- **Zap**: Structured logging

### Observability
- **OpenTelemetry**: Distributed tracing
- **Prometheus**: Metrics collection
- **Jaeger**: Trace visualization

### Data & Storage
- **Badger**: Embedded key-value store
- **PostgreSQL**: Primary relational database (via internal repos)
- **Various clients**: Redis, MongoDB, Neo4j, etc.

### AI/ML Integration
- Multiple LLM provider clients (OpenAI, Gemini, GLM)
- Vector database clients (Qdrant, Pinecone, Weaviate)
- Knowledge graph capabilities

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

### Required
- All new artifacts must be documented in the official tree
- Standardized function comments for Go code
- Follow the Clean Architecture layer separation

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

The CLI is accessible via the `hulk` command:

```bash
# Generate new MCP project
hulk generate --template go --name my-mcp

# List available templates
hulk template list

# Validate MCP structure
hulk validate mcp ./my-mcp

# Start monitoring
hulk monitor start

# View application status
hulk status
```

## Troubleshooting

### Common Issues
- **NATS connection failed**: Check NATS server is running on localhost:4222
- **Configuration not found**: Ensure `config/config.yaml` exists or set environment variables
- **Build failures**: Run `make deps` to ensure dependencies are current
- **Lint failures**: Install golangci-lint via `make dev-setup`

### Debug Mode
Set `MCP_HULK_LOGGING_LEVEL=debug` for verbose logging output.

### Health Checks
The system provides health check endpoints for monitoring deployment status.