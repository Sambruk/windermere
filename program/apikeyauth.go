package program

import (
	"context"
	"net/http"
)

// Type for storing the authenticated tenant in the context
type tenantContextKey int

const (
	tenantKey tenantContextKey = iota
)

// Given an API key, try to find a tenant that matches,
// returns nil if no tenant has that key.
func lookupTenant(key string, clients map[string]string) *string {
	for client, value := range clients {
		if key == value {
			return &client
		}
	}
	return nil
}

// APIKeyAuthMiddleware provides authentication middleware for API keys.
// headerName is the HTTP header to use for the API key.
// clients is a map from tenant names to API keys.
func APIKeyAuthMiddleware(h http.Handler, headerName string, clients map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.Header[http.CanonicalHeaderKey(headerName)]
		if !ok || len(keys) < 1 {
			http.Error(w, "No API key", http.StatusForbidden)
			return
		}

		tenant := lookupTenant(keys[0], clients)

		if tenant == nil {
			http.Error(w, "Invalid API key", http.StatusForbidden)
			return
		}

		// We have an authenticated client, set the tenant name in the context
		ctx := r.Context()
		newContext := context.WithValue(ctx, tenantKey, *tenant)
		r2 := r.Clone(newContext)

		h.ServeHTTP(w, r2)
	})
}

// Gets the tenant from context if the client has authenticated with API key
// Returns nil if the client hasn't authenticated with API key
func APIKeyAuthenticatedTenantFromContext(ctx context.Context) *string {
	value := ctx.Value(tenantKey)
	if value == nil {
		return nil
	}
	tenant := value.(string)
	return &tenant
}
