package teapot

import (
	"fmt"
	"net/http"

	"github.com/mallardduck/teapot-router/internal/core"
	"github.com/mallardduck/teapot-router/pkg/dispatch"
)

// Matchers provides convenient access to all built-in matcher constructors.
// A zero-value instance is passed as the second argument to the Dispatch
// callback so that the dispatch package does not need to be imported.
type Matchers struct{}

// QueryExists returns a Matcher that matches when the given query parameter is present.
func (Matchers) QueryExists(key string) dispatch.Matcher { return dispatch.QueryExists(key) }

// QueryEquals returns a Matcher that matches when the given query parameter equals value.
func (Matchers) QueryEquals(key, value string) dispatch.Matcher {
	return dispatch.QueryEquals(key, value)
}

// HeaderExists returns a Matcher that matches when the given header is present with a non-empty value.
func (Matchers) HeaderExists(key string) dispatch.Matcher { return dispatch.HeaderExists(key) }

// HeaderEquals returns a Matcher that matches when the given header equals value.
func (Matchers) HeaderEquals(key, value string) dispatch.Matcher {
	return dispatch.HeaderEquals(key, value)
}

// DispatchBuilder configures routes for a Dispatch group.
// Each route shares the same method and pattern but is distinguished by
// query parameter conditions (or other Matchers).
type DispatchBuilder struct {
	router     *Router
	method     string
	pattern    string
	chiPattern string
	routes     []*dispatchRouteConfig
}

type dispatchRouteConfig struct {
	handler     http.Handler
	matchers    []dispatch.Matcher
	name        string
	action      string
	middlewares []func(http.Handler) http.Handler
}

// DispatchRoute is the fluent builder for a single route within a Dispatch group.
type DispatchRoute struct {
	builder *DispatchBuilder
	config  *dispatchRouteConfig
}

// Dispatch registers a group of routes on the same method+pattern, distinguished
// by query parameters (or other Matchers). The callback receives a DispatchBuilder
// for route configuration and a Matchers value for creating matchers without
// importing the dispatch package directly.
//
// Example:
//
//	r.Dispatch("GET", "/{bucket}", func(d *teapot.DispatchBuilder, m teapot.Matchers) {
//	    d.Default(listV1).Name("s3.bucket.list-v1").Action("api:s3:ListObjects")
//	    d.When(m.QueryEquals("list-type", "2")).Do(listV2).Name("s3.bucket.list-v2").Action("api:s3:ListObjectsV2")
//	    d.When(m.QueryExists("location")).Do(getLocation).Name("s3.bucket.get-location")
//	})
func (r *Router) Dispatch(method, pattern string, fn func(d *DispatchBuilder, m Matchers)) {
	fullPattern := r.pathPrefix + pattern
	chiPattern, wildcardParams := core.TranslatePattern(fullPattern)
	dispatcherKey := method + ":" + chiPattern

	// Conflict checks — catch problems early rather than letting chi panic
	if _, exists := r.dispatchers[dispatcherKey]; exists {
		panic(fmt.Sprintf(
			"teapot: duplicate dispatch registration\n\n"+
				"  route:  %s %s\n"+
				"  cause:  a Dispatch() or query-multiplexed route already owns this method+pattern\n"+
				"  help:   consolidate all variants for this route into a single Dispatch() block",
			method, fullPattern))
	}
	if _, exists := r.directRoutes[dispatcherKey]; exists {
		panic(fmt.Sprintf(
			"teapot: duplicate route registration\n\n"+
				"  route:  %s %s\n"+
				"  cause:  a direct route (GET, POST, etc.) is already registered for this method+pattern\n"+
				"  help:   remove the direct route, or replace both with a single Dispatch() block",
			method, fullPattern))
	}

	db := &DispatchBuilder{
		router:     r,
		method:     method,
		pattern:    fullPattern,
		chiPattern: chiPattern,
	}

	fn(db, Matchers{})

	// Validate that every route has a handler set
	for _, cfg := range db.routes {
		if cfg.handler == nil {
			panic(fmt.Sprintf(
				"teapot: invalid dispatch route %q (%s): no handler configured\n"+
					"hint: call Do(...) for this route or provide a handler via Default(...)",
				cfg.name,
				cfg.action,
			))
		}
	}

	// Register sub-routes into the shared registry (for Routes(), URL generation,
	// and RouteContextMiddleware lookups) and collect them for the dispatcher
	coreRoutes := make([]*core.Route, 0, len(db.routes))
	for _, cfg := range db.routes {
		fullName := r.namePrefix + cfg.name

		// Validate name uniqueness (same logic as RouteBuilder.Name)
		if fullName != "" {
			for existingName, existingRoute := range r.nameIndex {
				if existingName == fullName {
					if existingRoute.Method == method {
						panic(fmt.Sprintf("teapot: duplicate route %s:%s (existing: %s, new: %s)",
							method, fullName, existingRoute.Pattern, fullPattern))
					}
					if existingRoute.Pattern != fullPattern {
						panic(fmt.Sprintf("teapot: route name %q used with different paths: %s %s vs %s %s",
							fullName, existingRoute.Method, existingRoute.Pattern, method, fullPattern))
					}
				}
			}
		}

		rt := &core.Route{
			Method:         method,
			Pattern:        fullPattern,
			ChiPattern:     chiPattern,
			Handler:        cfg.handler,
			Name:           fullName,
			Action:         cfg.action,
			QueryMatchers:  cfg.matchers,
			Middlewares:    append(append([]func(http.Handler) http.Handler{}, r.middlewares...), cfg.middlewares...),
			WildcardParams: wildcardParams,
		}

		*r.routes = append(*r.routes, rt)
		if fullName != "" {
			r.nameIndex[fullName] = rt
		}
		coreRoutes = append(coreRoutes, rt)
	}

	// Register with chi. core.Dispatcher handles dispatching, context injection,
	// wildcard remapping, and middleware via lazy build on first request (or
	// eagerly if Finalize is called).
	coreDisp := &core.Dispatcher{Routes: coreRoutes}
	r.dispatchers[dispatcherKey] = coreDisp
	r.mux.Method(method, chiPattern, coreDisp)
}

// Default sets the fallback handler for the dispatch group.
// It matches when no other route's matchers match. Returns a DispatchRoute
// for chaining Name, Action, and With.
func (db *DispatchBuilder) Default(h http.Handler) *DispatchRoute {
	cfg := &dispatchRouteConfig{handler: h}
	db.routes = append(db.routes, cfg)
	return &DispatchRoute{builder: db, config: cfg}
}

// FuncDefault sets the fallback handler using a plain function literal.
func (db *DispatchBuilder) FuncDefault(h func(http.ResponseWriter, *http.Request)) *DispatchRoute {
	return db.Default(http.HandlerFunc(h))
}

// When starts a conditional route with the given matchers (AND semantics).
// Call Do on the returned DispatchRoute to set the handler.
func (db *DispatchBuilder) When(matchers ...dispatch.Matcher) *DispatchRoute {
	cfg := &dispatchRouteConfig{matchers: matchers}
	db.routes = append(db.routes, cfg)
	return &DispatchRoute{builder: db, config: cfg}
}

// Do sets the handler for this conditional route. Required after When.
func (dr *DispatchRoute) Do(h http.Handler) *DispatchRoute {
	dr.config.handler = h
	return dr
}

// FuncDo sets the handler for this conditional route using a plain function literal.
func (dr *DispatchRoute) FuncDo(h func(http.ResponseWriter, *http.Request)) *DispatchRoute {
	return dr.Do(http.HandlerFunc(h))
}

// Name assigns a name to this route for URL generation and route listing.
func (dr *DispatchRoute) Name(name string) *DispatchRoute {
	dr.config.name = name
	return dr
}

// Action assigns an action identifier to this route (injected into request context).
func (dr *DispatchRoute) Action(action string) *DispatchRoute {
	dr.config.action = action
	return dr
}

// With adds route-specific middleware to this route.
func (dr *DispatchRoute) With(middlewares ...func(http.Handler) http.Handler) *DispatchRoute {
	dr.config.middlewares = append(dr.config.middlewares, middlewares...)
	return dr
}
