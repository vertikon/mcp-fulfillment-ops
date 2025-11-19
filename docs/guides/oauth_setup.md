# OAuth Setup Guide

## Configuração de Variáveis de Ambiente

Crie um arquivo `.env` na raiz do projeto com as seguintes variáveis:

```bash
# Auth0 Configuration (TESTE - TEMPORÁRIA)
MCP_HULK_OAUTH_AUTH0_ENABLED=true
MCP_HULK_OAUTH_AUTH0_DOMAIN=dev-vertikon.us.auth0.com
MCP_HULK_OAUTH_AUTH0_CLIENT_ID=iECzv5C9dFHWWbF1rqmsl1skKkTwW7xz
MCP_HULK_OAUTH_AUTH0_CLIENT_SECRET=RTOePOhr9ykXApyaFY8TdvfFzKOQ9-d0bw-c7Qi8yZBeDO-ABtaNm1Qk4K1WSiyl
MCP_HULK_OAUTH_AUTH0_REDIRECT_URL=http://localhost:8080/auth/callback/auth0

# JWT Configuration
MCP_HULK_JWT_SECRET_KEY=change-me-in-production-use-strong-random-key-32-bytes-minimum
```

**IMPORTANTE:** 
- Nunca commite o arquivo `.env` no Git. Ele contém credenciais sensíveis.
- A chave `MCP_HULK_OAUTH_AUTH0_CLIENT_SECRET` acima é **TEMPORÁRIA PARA TESTES**. 
- **Trocar por chave de produção antes de deploy em produção.**

**Arquivo de exemplo:** Veja `docs/guides/oauth_setup_example.env` para template completo.

## Configuração Auth0

1. Acesse o [Auth0 Dashboard](https://manage.auth0.com/)
2. Vá em Applications → Settings
3. Configure:
   - **Allowed Callback URLs**: `http://localhost:8080/auth/callback/auth0`
   - **Allowed Logout URLs**: `http://localhost:8080`
   - **Allowed Web Origins**: `http://localhost:8080`

## Uso do Auth0 Provider

```go
import "github.com/vertikon/mcp-fulfillment-ops/internal/security/auth"

// Criar configuração Auth0
auth0Config := auth.OAuthProviderConfig{
    Domain:       "dev-vertikon.us.auth0.com",
    ClientID:     "iECzv5C9dFHWWbF1rqmsl1skKkTwW7xz",
    ClientSecret: os.Getenv("MCP_HULK_OAUTH_AUTH0_CLIENT_SECRET"), // Ou use diretamente: "RTOePOhr9ykXApyaFY8TdvfFzKOQ9-d0bw-c7Qi8yZBeDO-ABtaNm1Qk4K1WSiyl"
    RedirectURL:  "http://localhost:8080/auth/callback/auth0",
    Scopes:       []string{"openid", "profile", "email"},
}

// Criar provider
auth0Provider := auth.NewAuth0Provider(auth0Config)

// Registrar no OAuth Manager
oauthManager := auth.NewOAuthManager()
oauthManager.RegisterProvider(auth.OAuthProviderAuth0, auth0Provider)

// Obter URL de autorização
authURL, err := oauthManager.GetAuthURL(ctx, auth.OAuthProviderAuth0, "random-state")
if err != nil {
    log.Fatal(err)
}

// Redirecionar usuário para authURL
// Após autorização, Auth0 redireciona para callback com código

// Processar callback
userInfo, err := oauthManager.HandleCallback(ctx, auth.OAuthProviderAuth0, code, state)
if err != nil {
    log.Fatal(err)
}

// userInfo contém: ID, Email, Name, Picture
```

## Testando o Fluxo OAuth

1. Inicie o servidor
2. Acesse `/auth/login/auth0` para iniciar o fluxo
3. Faça login no Auth0
4. Será redirecionado para `/auth/callback/auth0` com o código
5. O sistema troca o código por tokens e obtém informações do usuário

