package main

import (
	"net/http"

	"github.com/Sambruk/lakeside/ss12000v2"
	"github.com/Sambruk/windermere/ss12000v1tov2"
	"github.com/Sambruk/windermere/windermere"
)

type SS12000v2TenantMux struct {
	windermere *windermere.Windermere
	routers    map[string]http.Handler
}

func NewSS12000v2TenantMux(w *windermere.Windermere) http.Handler {
	return &SS12000v2TenantMux{
		windermere: w,
		routers:    make(map[string]http.Handler),
	}
}

func (mux *SS12000v2TenantMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tenant := r.Header.Get("X-Tenant")
	if tenant == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	router, ok := mux.routers[tenant]

	if !ok {
		router = ss12000v2.NewRouter(ss12000v1tov2.NewAdapter(mux.windermere, tenant, true))
		mux.routers[tenant] = router
	}

	router.ServeHTTP(w, r)
}
