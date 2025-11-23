package auth

// Example: How to initialize Auth0 provider with the provided credentials
//
// func ExampleAuth0Setup() {
// 	// Create Auth0 provider configuration
// 	auth0Config := OAuthProviderConfig{
// 		Domain:       "dev-vertikon.us.auth0.com",
// 		ClientID:     "iECzv5C9dFHWWbF1rqmsl1skKkTwW7xz",
// 		ClientSecret: "RTOePOhr9ykXApyaFY8TdvfFzKOQ9-d0bw-c7Qi8yZBeDO-ABtaNm1Qk4K1WSiyl", // TEMPORÁRIA - Trocar em produção
// 		RedirectURL:  "http://localhost:8080/auth/callback/auth0",
// 		Scopes:       []string{"openid", "profile", "email"},
// 	}
//
// 	// Create Auth0 provider
// 	auth0Provider := NewAuth0Provider(auth0Config)
//
// 	// Register with OAuth Manager
// 	oauthManager := NewOAuthManager()
// 	oauthManager.RegisterProvider(OAuthProviderAuth0, auth0Provider)
//
// 	// Usage:
// 	// 1. Get authorization URL
// 	// authURL, _ := oauthManager.GetAuthURL(ctx, OAuthProviderAuth0, "random-state-string")
// 	//
// 	// 2. After user authorizes, handle callback
// 	// userInfo, _ := oauthManager.HandleCallback(ctx, OAuthProviderAuth0, code, state)
// }
