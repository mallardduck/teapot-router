package teapot

import (
	"net/http"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/mallardduck/teapot-router/internal/core"
)

// optimizedHandler is a smart handler that can be optimized after finalization
// Before Finalize(): uses runtime checks (slow)
// After Finalize(): uses pre-computed fast path (fast)
type optimizedHandler struct {
	route     *core.Route
	fastPath  http.Handler
	finalized atomic.Bool
	router    *Router
}

func (oh *optimizedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if oh.finalized.Load() {
		// Fast path: use pre-computed handler
		oh.fastPath.ServeHTTP(w, r)
	} else {
		// Slow path: runtime checks (for fluent API before finalization)
		oh.slowPath(w, r)
	}
}

// slowPath handles requests before finalization (supports fluent API)
func (oh *optimizedHandler) slowPath(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Runtime checks for Name/Action (set via fluent API)
	if oh.route.Action != "" {
		ctx = core.SetAction(ctx, oh.route.Action)
	}
	if oh.route.Name != "" {
		ctx = core.SetRouteName(ctx, oh.route.Name)
	}

	// Handle wildcard parameter remapping
	if len(oh.route.WildcardParams) > 0 {
		wildcardValue := chi.URLParam(r, "*")
		rctx := chi.RouteContext(ctx)
		if rctx != nil {
			for paramName := range oh.route.WildcardParams {
				rctx.URLParams.Add(paramName, wildcardValue)
			}
		}
	}

	r = r.WithContext(ctx)

	// Apply route-specific middleware at runtime
	handler := http.Handler(oh.route.Handler)
	for i := len(oh.route.Middlewares) - 1; i >= 0; i-- {
		handler = oh.route.Middlewares[i](handler)
	}

	handler.ServeHTTP(w, r)
}

// createOptimizedHandler generates the fastest possible handler for a route
// based on its actual configuration (called during Finalize)
func (r *Router) createOptimizedHandler(rt *core.Route) http.Handler {
	// Analyze what the route actually uses
	hasAction := rt.Action != ""
	hasName := rt.Name != ""
	hasWildcards := len(rt.WildcardParams) > 0
	hasMiddleware := len(rt.Middlewares) > 0

	// Fast path 1: Completely minimal route (matches Chi performance)
	if !hasAction && !hasName && !hasWildcards && !hasMiddleware {
		return rt.Handler
	}

	// Pre-apply middleware (do this once, not per request)
	finalHandler := rt.Handler
	if hasMiddleware {
		handler := http.Handler(rt.Handler)
		for i := len(rt.Middlewares) - 1; i >= 0; i-- {
			handler = rt.Middlewares[i](handler)
		}
		finalHandler = handler.(http.HandlerFunc)
	}

	// Fast path 2: No context injection needed
	if !hasAction && !hasName && !hasWildcards {
		return finalHandler
	}

	// Optimized path: Pre-compute what needs to be done
	action := rt.Action
	name := rt.Name
	wildcardParams := make(map[string]bool, len(rt.WildcardParams))
	for k, v := range rt.WildcardParams {
		wildcardParams[k] = v
	}

	// Return optimized handler with pre-computed values
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		// Pre-computed action/name injection (no runtime checks)
		if action != "" {
			ctx = core.SetAction(ctx, action)
		}
		if name != "" {
			ctx = core.SetRouteName(ctx, name)
		}

		// Pre-computed wildcard handling
		if len(wildcardParams) > 0 {
			wildcardValue := chi.URLParam(req, "*")
			rctx := chi.RouteContext(ctx)
			if rctx != nil {
				for paramName := range wildcardParams {
					rctx.URLParams.Add(paramName, wildcardValue)
				}
			}
		}

		finalHandler.ServeHTTP(w, req.WithContext(ctx))
	})
}
