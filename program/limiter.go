package program

import (
	"net/http"
	"sync"

	"github.com/Sambruk/windermere/scimserverlite"
	"golang.org/x/time/rate"
)

// Limiter returns a middleware with token bucket rate limiting applied per tenant
func Limiter(h http.Handler, tenantGetter scimserverlite.TenantGetter, r rate.Limit, b int) http.Handler {

	limiters := make(map[string]*rate.Limiter)
	var lock sync.Mutex

	getLimiter := func(entity string) *rate.Limiter {
		lock.Lock()
		defer lock.Unlock()

		if limiter, ok := limiters[entity]; ok {
			return limiter
		}
		limiter := rate.NewLimiter(r, b)
		limiters[entity] = limiter
		return limiter
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := getLimiter(tenantGetter(r.Context()))

		if limiter.Wait(r.Context()) != nil {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		h.ServeHTTP(w, r)
	})
}
