package teapot

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mallardduck/teapot-router/internal/core"
)

// routeContextMiddleware is the middleware handler that injects teapot route metadata
// (Name, Action) into the request context. It implements http.Handler for cleaner integration.
type routeContextMiddleware struct {
	router *Router
	next   http.Handler
}

// ServeHTTP implements http.Handler, providing route metadata injection.
func (m *routeContextMiddleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	// Try fast path first (Chi's RouteContext available in Route() groups)
	if route := m.tryFastPath(ctx); route != nil {
		ctx = m.injectRouteMetadata(ctx, route)
		req = req.WithContext(ctx)
		m.next.ServeHTTP(w, req)
		return
	}

	// Fallback: manually match route when RouteContext unavailable
	if route := m.tryFallbackPath(req.Method, req.URL.Path); route != nil {
		ctx = m.injectRouteMetadata(ctx, route)
		req = req.WithContext(ctx)
	}

	m.next.ServeHTTP(w, req)
}

// tryFastPath attempts to find a route using Chi's RouteContext (fast path).
// Returns the route if found, nil otherwise.
func (m *routeContextMiddleware) tryFastPath(ctx context.Context) *core.Route {
	rctx := chi.RouteContext(ctx)
	if rctx == nil || rctx.RoutePattern() == "" {
		return nil
	}

	method := rctx.RouteMethod
	pattern := rctx.RoutePattern()
	key := method + ":" + pattern

	// Check if this is a dispatcher route (query-multiplexed)
	if disp, exists := m.router.dispatchers[key]; exists && len(disp.Routes) > 0 {
		return m.findBestDispatcherRoute(disp)
	}

	// Check if this is a direct route
	if route, exists := m.router.directRoutes[key]; exists {
		return route
	}

	return nil
}

// tryFallbackPath manually matches a route when Chi's RouteContext is unavailable.
// Returns the route if found, nil otherwise.
func (m *routeContextMiddleware) tryFallbackPath(method, path string) *core.Route {
	// Try exact match first (fast)
	key := method + ":" + path
	if route, exists := m.router.directRoutes[key]; exists {
		return route
	}

	// Pattern matching (slower)
	return m.router.findMatchingRoute(method, path)
}

// findBestDispatcherRoute finds the best route from a dispatcher for early context injection.
// Prefers routes without query matchers (fallback routes), otherwise uses the first (most specific).
func (m *routeContextMiddleware) findBestDispatcherRoute(disp *core.Dispatcher) *core.Route {
	// Find fallback route (no query matchers)
	for _, rt := range disp.Routes {
		if len(rt.QueryMatchers) == 0 {
			return rt
		}
	}

	// If no fallback, use first route (most specific)
	if len(disp.Routes) > 0 {
		return disp.Routes[0]
	}

	return nil
}

// injectRouteMetadata adds route metadata (Action, Name) to the context if present.
// Returns the updated context.
func (m *routeContextMiddleware) injectRouteMetadata(ctx context.Context, route *core.Route) context.Context {
	if route.Action != "" {
		ctx = core.SetAction(ctx, route.Action)
	}
	if route.Name != "" {
		ctx = core.SetRouteName(ctx, route.Name)
	}
	return ctx
}

// RouteContextMiddleware returns middleware that injects teapot route metadata (Name, Action)
// into the request context. The middleware intelligently adapts based on available context:
// - When Chi's RouteContext is available (fast path), it uses that
// - When RouteContext is not available (fallback), it manually matches routes
//
// RECOMMENDED: Use inside a Route() group for best performance:
//
//	r.Route("/", func(r *teapot.Router) {
//	    r.Use(teapot.RouteContextMiddleware(r))  // Fast path
//	    r.Use(loggingMiddleware)                 // Sees route metadata
//	    // register routes...
//	})
//
// ALTERNATIVE: Add globally (works but slightly slower due to fallback matching):
//
//	r.Use(teapot.RouteContextMiddleware(r))  // Uses fallback matching
//	r.Use(loggingMiddleware)                 // Sees route metadata
//
// For query-multiplexed routes, this provides early context injection. The Dispatcher
// will re-inject more specific metadata after query parameter matching.
func RouteContextMiddleware(r *Router) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return &routeContextMiddleware{
			router: r,
			next:   next,
		}
	}
}
