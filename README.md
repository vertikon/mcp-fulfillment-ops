# mcp-fulfillment-ops

mcp-fulfillment-ops is a comprehensive Model Context Protocol (MCP) template and generation engine for building high-performance, scalable MCP implementations.

## 🚀 Features

- **Complete MCP Framework**: Full implementation of the Model Context Protocol
- **Template Engine**: Generate code from multiple templates (Go, TinyGo, Rust WASM, Web)
- **AI Integration**: Built-in support for OpenAI, Gemini, GLM, and other LLM providers
- **Knowledge Management**: RAG (Retrieval-Augmented Generation) with vector and graph databases
- **State Management**: Distributed state with event sourcing
- **Observability**: Full monitoring, tracing, and analytics
- **Security**: Enterprise-grade authentication, authorization, and encryption
- **Hybrid Compute**: Support for local CPU and external GPU (RunPod) compute
- **Clean Architecture**: Modular, maintainable code structure

## 📁 Project Structure

The project follows a **Clean Architecture** pattern with 14 main blocks:

```
├── cmd/                    # Application entry points
├── internal/               # Private application code
│   ├── core/              # Performance engine
│   ├── ai/                # AI & knowledge management
│   ├── state/             # State management
│   ├── monitoring/        # Observability
│   ├── versioning/        # Version control
│   ├── mcp/               # MCP protocol & generation
│   ├── services/         # Application services
│   ├── domain/            # Domain logic
│   ├── application/      # Use cases
│   ├── infrastructure/    # External integrations
│   ├── interfaces/        # Input/output adapters
│   └── security/          # Security layer
├── pkg/                    # Public libraries
├── templates/              # Code generation templates
├── tools/                  # Development tools
├── config/                 # Configuration files
├── scripts/                # Automation scripts
└── docs/                   # Documentation
```

## 🛠️ Quick Start

### Prerequisites

- Go 1.21 or higher
- Docker (optional)
- Access to supported databases and message brokers

### Installation

```bash
# Clone the repository
git clone https://github.com/vertikon/mcp-fulfillment-ops.git
cd mcp-fulfillment-ops

# Install dependencies
make deps

# Build the application
make build
```

### Running the Application

```bash
# Run with default configuration
make run

# Or using Go directly
go run ./cmd/main.go
```

## 🎯 Usage

### CLI Commands

The mcp-fulfillment-ops CLI provides comprehensive commands for managing MCPs and templates:

```bash
# Generate a new MCP
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

### Template Generation

```bash
# Generate a Go backend service
hulk generate template go --name my-service --with-grpc

# Generate a TinyGo WASM module
hulk generate template tinygo --name my-wasm

# Generate a React web frontend
hulk generate template web --name my-ui

# Generate a premium MCP with full features
hulk generate template mcp-go-premium --name enterprise-mcp
```

## 🔧 Configuration

Configuration is managed through YAML files in the `config/` directory:

- `config/core/` - Engine and core settings
- `config/ai/` - AI model and knowledge settings
- `config/infrastructure/` - Database and messaging settings
- `config/environments/` - Environment-specific configs
- `config/features.yaml` - Feature flags

### Environment Setup

```bash
# Development
export HULK_ENV=dev

# Production
export HULK_ENV=prod

# Override configuration path
export HULK_CONFIG_PATH=/path/to/config
```

## 🤖 AI Integration

mcp-fulfillment-ops integrates with multiple AI providers:

### Supported Providers

- **OpenAI**: GPT-3.5, GPT-4, GPT-4-turbo
- **Google**: Gemini Pro, Gemini Ultra
- **GLM**: GLM-4.6, GLM-3-turbo
- **Custom**: OpenAI-compatible endpoints

### Knowledge Management

Built-in RAG capabilities with:

- Vector databases (Qdrant, Pinecone, Weaviate)
- Graph databases (Neo4j, ArangoDB)
- Document stores (MongoDB, CouchDB)
- Hybrid search (vector + keyword)

### Memory Management

- **Episodic Memory**: Short-term conversation history
- **Semantic Memory**: Long-term knowledge storage
- **Working Memory**: Current session context
- **Memory Consolidation**: Automatic transfer from short to long-term

## 📊 Monitoring & Observability

Comprehensive monitoring built-in:

- **Metrics**: Prometheus-compatible metrics
- **Tracing**: OpenTelemetry/Jaeger distributed tracing
- **Logging**: Structured logging with multiple levels
- **Health Checks**: Liveness and readiness probes
- **Analytics**: Usage, performance, and cost analytics

### Dashboard

Access the monitoring dashboard at `http://localhost:3000` (Grafana)

## 🔒 Security

Enterprise-grade security features:

- **Authentication**: JWT, OAuth 2.0, SSO support
- **Authorization**: Role-based access control (RBAC)
- **Encryption**: AES-256 encryption at rest and in transit
- **Audit Logging**: Comprehensive audit trails
- **Compliance**: GDPR, HIPAA, SOX compliance support

## 🚀 Deployment

### Docker

```bash
# Build image
make docker

# Run container
docker run -p 8080:8080 -e HULK_ENV=prod mcp-fulfillment-ops:latest
```

### Kubernetes

```bash
# Deploy to Kubernetes
kubectl apply -f deployments/k8s/

# Check deployment status
kubectl get pods -l app=mcp-fulfillment-ops
```

### Serverless

```bash
# Deploy as serverless functions
hulk deploy serverless --provider aws
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run integration tests
go test -v ./tests/integration/...
```

## 📚 Documentation

- [Architecture Guide](docs/architecture/)
- [API Documentation](docs/api/)
- [User Guides](docs/guides/)
- [Examples](docs/examples/)
- [Troubleshooting](docs/guides/troubleshooting.md)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/vertikon/mcp-fulfillment-ops/issues)
- **Discussions**: [GitHub Discussions](https://github.com/vertikon/mcp-fulfillment-ops/discussions)
- **Documentation**: [Project Docs](https://docs.vertikon.com/mcp-fulfillment-ops)

## 🙏 Acknowledgments

- The **Model Context Protocol (MCP)** community
- Anthropic for the MCP specification
- All contributors and users of mcp-fulfillment-ops

---

**Built with ❤️ by the Vertikon Team**