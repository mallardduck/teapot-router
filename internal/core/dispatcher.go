package core

import (
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
)

// Dispatcher handles routing for a single method+pattern combination
// It supports multiple handlers distinguished by query parameters
type Dispatcher struct {
	Routes []*Route
}

func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Fast path: single route with no query matchers (common case)
	// This avoids the overhead of iteration and query matching for simple routes
	if len(d.Routes) == 1 && len(d.Routes[0].QueryMatchers) == 0 {
		d.executeRoute(w, r, d.Routes[0])
		return
	}

	// Full path: multiple routes or query multiplexing
	for _, rt := range d.Routes {
		if d.matchesQuery(r, rt) {
			d.executeRoute(w, r, rt)
			return
		}
	}

	// No route matched
	http.NotFound(w, r)
}

func (d *Dispatcher) matchesQuery(r *http.Request, rt *Route) bool {
	// If route has no query matchers, it matches any query string
	if len(rt.QueryMatchers) == 0 {
		return true
	}

	// All query matchers must match
	for _, matcher := range rt.QueryMatchers {
		if !matcher.Matches(r) {
			return false
		}
	}

	return true
}

func (d *Dispatcher) executeRoute(w http.ResponseWriter, r *http.Request, rt *Route) {
	// Inject context values
	ctx := r.Context()
	if rt.Action != "" {
		ctx = SetAction(ctx, rt.Action)
	}
	if rt.Name != "" {
		ctx = SetRouteName(ctx, rt.Name)
	}

	// Handle wildcard parameter remapping
	// If route has wildcard params (e.g., {key:.*}), copy from chi's "*" param
	if len(rt.WildcardParams) > 0 {
		wildcardValue := chi.URLParam(r, "*")
		// Add wildcard parameters to chi's route context
		rctx := chi.RouteContext(ctx)
		if rctx != nil {
			for paramName := range rt.WildcardParams {
				rctx.URLParams.Add(paramName, wildcardValue)
			}
		}
	}

	r = r.WithContext(ctx)

	// Apply route-specific middleware
	handler := http.Handler(rt.Handler)
	for i := len(rt.Middlewares) - 1; i >= 0; i-- {
		handler = rt.Middlewares[i](handler)
	}

	handler.ServeHTTP(w, r)
}

func (d *Dispatcher) AddRoute(rt *Route) {
	d.Routes = append(d.Routes, rt)

	// Sort by specificity (most specific first)
	sort.Slice(d.Routes, func(i, j int) bool {
		return d.routeSpecificity(d.Routes[i]) > d.routeSpecificity(d.Routes[j])
	})
}

func (d *Dispatcher) routeSpecificity(rt *Route) int {
	specificity := 0
	for _, matcher := range rt.QueryMatchers {
		specificity += matcher.Specificity()
	}
	return specificity
}

func (d *Dispatcher) UpdateSpecificity() {
	// Re-sort routes by specificity
	sort.Slice(d.Routes, func(i, j int) bool {
		return d.routeSpecificity(d.Routes[i]) > d.routeSpecificity(d.Routes[j])
	})
}
