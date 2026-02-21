package core

import (
	"net/http"
	"sort"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/mallardduck/teapot-router/pkg/dispatch"
)

// Dispatcher handles routing for a single method+pattern combination.
// It supports multiple handlers distinguished by query parameters.
//
// Routes are accumulated during registration via AddRoute. On the first
// request (or when Build is called explicitly), all routes are wrapped with
// context injection, wildcard remapping, and middleware, and a dispatch.Dispatcher
// is built to handle the actual matching. All subsequent requests delegate
// directly to that built dispatcher.
type Dispatcher struct {
	Routes []*Route

	built *dispatch.Dispatcher
	once  sync.Once
}

func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.once.Do(d.build)
	d.built.ServeHTTP(w, r)
}

// Build eagerly builds the internal dispatcher. Safe to call multiple times;
// only the first call has any effect. Called by Finalize to avoid lazy-build
// latency on the first request.
func (d *Dispatcher) Build() {
	d.once.Do(d.build)
}

func (d *Dispatcher) build() {
	d.built = dispatch.New(func(b *dispatch.Builder) {
		for _, rt := range d.Routes {
			wrapped := wrapRoute(rt)
			if len(rt.QueryMatchers) == 0 {
				b.Default(wrapped)
			} else {
				b.When(rt.QueryMatchers...).Do(wrapped)
			}
		}
	})
}

// wrapRoute captures a Route's configuration at build time and returns a
// handler that applies context injection, wildcard remapping, and middleware
// before calling the original handler.
func wrapRoute(rt *Route) http.HandlerFunc {
	name := rt.Name
	action := rt.Action
	wildcardParams := rt.WildcardParams
	middlewares := rt.Middlewares
	handler := rt.Handler

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if action != "" {
			ctx = SetAction(ctx, action)
		}
		if name != "" {
			ctx = SetRouteName(ctx, name)
		}

		if len(wildcardParams) > 0 {
			wildcardValue := chi.URLParam(r, "*")
			rctx := chi.RouteContext(ctx)
			if rctx != nil {
				for paramName := range wildcardParams {
					rctx.URLParams.Add(paramName, wildcardValue)
				}
			}
		}

		r = r.WithContext(ctx)

		h := handler
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		h.ServeHTTP(w, r)
	}
}

// AddRoute adds a new route to the dispatcher and re-sorts routes by specificity.
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

// UpdateSpecificity re-sorts all routes by their specificity
func (d *Dispatcher) UpdateSpecificity() {
	// Re-sort routes by specificity
	sort.Slice(d.Routes, func(i, j int) bool {
		return d.routeSpecificity(d.Routes[i]) > d.routeSpecificity(d.Routes[j])
	})
}

// FindBestDispatcherRoute finds the best route from a dispatcher for early context injection.
// Prefers routes without query matchers (fallback routes), otherwise uses the first (most specific).
func FindBestDispatcherRoute(dispatcher *Dispatcher) *Route {
	// Find fallback route (no query matchers)
	for _, rt := range dispatcher.Routes {
		if len(rt.QueryMatchers) == 0 {
			return rt
		}
	}

	// If no fallback, use first route (most specific)
	if len(dispatcher.Routes) > 0 {
		return dispatcher.Routes[0]
	}

	return nil
}
