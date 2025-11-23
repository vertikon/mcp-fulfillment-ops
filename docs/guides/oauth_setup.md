# OAuth Setup Guide

## Configuração de Variáveis de Ambiente

Crie um arquivo `.env` na raiz do projeto com as seguintes variáveis:

```bash
# Auth0 Configuration (TESTE - TEMPORÁRIA)
MCP_FULFILLMENT_OAUTH_AUTH0_ENABLED=true
MCP_FULFILLMENT_OAUTH_AUTH0_DOMAIN=dev-vertikon.us.auth0.com
MCP_FULFILLMENT_OAUTH_AUTH0_CLIENT_ID=iECzv5C9dFHWWbF1rqmsl1skKkTwW7xz
MCP_FULFILLMENT_OAUTH_AUTH0_CLIENT_SECRET=RTOePOhr9ykXApyaFY8TdvfFzKOQ9-d0bw-c7Qi8yZBeDO-ABtaNm1Qk4K1WSiyl
MCP_FULFILLMENT_OAUTH_AUTH0_REDIRECT_URL=http://localhost:8080/auth/callback/auth0

# JWT Configuration
MCP_FULFILLMENT_JWT_SECRET_KEY=change-me-in-production-use-strong-random-key-32-bytes-minimum
```