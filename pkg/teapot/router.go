package teapot

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mallardduck/teapot-router/internal/core"
	"github.com/mallardduck/teapot-router/pkg/dispatch"
)

// Router wraps chi.Mux and adds named routes, query multiplexing, and S3 actions
type Router struct {
	mux               *chi.Mux
	routes            *[]*core.Route
	dispatchers       map[string]*core.Dispatcher // key: "METHOD:PATTERN"
	directRoutes      map[string]*core.Route      // tracks direct-registered routes, key: "METHOD:PATTERN"
	nameIndex         map[string]*core.Route      // for URL generation
	pathPrefix        string
	namePrefix        string
	middlewares       []func(http.Handler) http.Handler
	optimizedHandlers *[]*optimizedHandler // for finalization optimization
	finalized         bool
	debugLog          bool // enable debug logging for auto-promotion
}

// RouteBuilder provides a fluent API for building routes
type RouteBuilder struct {
	router     *Router
	route      *core.Route
	dispatcher *core.Dispatcher
	isDirect   bool // true if route bypasses dispatcher (fast path)
}

// HTTPMethod represents HTTP verb options for resource routes
type HTTPMethod string

const (
	// POST is the HTTP POST method (standard REST for create)
	POST HTTPMethod = "POST"
	// PUT is the HTTP PUT method (S3-style for create/update)
	PUT HTTPMethod = "PUT"
)

// Name assigns a name to the route for URL generation
// Panics if the name is already registered with the same method or if the same name
// is used with different methods but different paths.
func (rb *RouteBuilder) Name(name string) *RouteBuilder {
	fullName := rb.router.namePrefix + name
	rb.route.Name = fullName

	// Validate: check for duplicate method+name
	for existingName, existingRoute := range rb.router.nameIndex {
		if existingName == fullName {
			// Same name found - check if it's the same method
			if existingRoute.Method == rb.route.Method {
				panic(fmt.Sprintf("teapot: duplicate route %s:%s (existing: %s, new: %s)",
					rb.route.Method, fullName, existingRoute.Pattern, rb.route.Pattern))
			}

			// Different method - ensure paths match (Laravel-style resources allow this)
			if existingRoute.Pattern != rb.route.Pattern {
				panic(fmt.Sprintf("teapot: route name %q used with different paths: %s %s vs %s %s",
					fullName, existingRoute.Method, existingRoute.Pattern, rb.route.Method, rb.route.Pattern))
			}
		}
	}

	// Register in name index
	rb.router.nameIndex[fullName] = rb.route

	return rb
}

// Action assigns an S3 action to the route (injected into request context)
func (rb *RouteBuilder) Action(action string) *RouteBuilder {
	rb.route.Action = action
	return rb
}

// Query adds a query parameter existence matcher
func (rb *RouteBuilder) Query(key string) *RouteBuilder {
	if rb.isDirect {
		panic("teapot: Cannot use .Query() with standard methods (GET, POST, etc). Use QueryGET, QueryPOST, etc. instead for query multiplexing.")
	}
	rb.route.QueryMatchers = append(rb.route.QueryMatchers, dispatch.QueryExistsMatcher{Key: key})
	rb.dispatcher.UpdateSpecificity()
	return rb
}

// QueryValue adds a query parameter value matcher
func (rb *RouteBuilder) QueryValue(key, value string) *RouteBuilder {
	if rb.isDirect {
		panic("teapot: Cannot use .QueryValue() with standard methods (GET, POST, etc). Use QueryGET, QueryPOST, etc. instead for query multiplexing.")
	}
	rb.route.QueryMatchers = append(rb.route.QueryMatchers, dispatch.QueryValueMatcher{Key: key, Value: value})
	rb.dispatcher.UpdateSpecificity()
	return rb
}

// With adds route-specific middleware
func (rb *RouteBuilder) With(middlewares ...func(http.Handler) http.Handler) *RouteBuilder {
	rb.route.Middlewares = append(rb.route.Middlewares, middlewares...)
	return rb
}

// New creates a new Router instance
func New() *Router {
	routes := make([]*core.Route, 0)
	optimizedHandlers := make([]*optimizedHandler, 0)
	return &Router{
		mux:               chi.NewRouter(),
		routes:            &routes,
		dispatchers:       make(map[string]*core.Dispatcher),
		directRoutes:      make(map[string]*core.Route),
		nameIndex:         make(map[string]*core.Route),
		optimizedHandlers: &optimizedHandlers,
		debugLog:          false, // can be enabled with router.SetDebugLog(true)
	}
}

// SetDebugLog enables or disables debug logging for route registration
func (r *Router) SetDebugLog(enabled bool) *Router {
	r.debugLog = enabled
	return r
}

// debugLog logs a message if debug logging is enabled
func (r *Router) debugLogf(format string, args ...any) {
	if r.debugLog {
		log.Printf("[teapot-debug] "+format, args...)
	}
}

// GET registers a GET route (direct, no query multiplexing)
// For query multiplexing, use QueryGET instead
func (r *Router) GET(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleDirect("GET", pattern, handler)
}

// POST registers a POST route (direct, no query multiplexing)
// For query multiplexing, use QueryPOST instead
func (r *Router) POST(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleDirect("POST", pattern, handler)
}

// PUT registers a PUT route (direct, no query multiplexing)
// For query multiplexing, use QueryPUT instead
func (r *Router) PUT(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleDirect("PUT", pattern, handler)
}

// DELETE registers a DELETE route (direct, no query multiplexing)
// For query multiplexing, use QueryDELETE instead
func (r *Router) DELETE(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleDirect("DELETE", pattern, handler)
}

// HEAD registers a HEAD route (direct, no query multiplexing)
// For query multiplexing, use QueryHEAD instead
func (r *Router) HEAD(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleDirect("HEAD", pattern, handler)
}

// PATCH registers a PATCH route (direct, no query multiplexing)
// For query multiplexing, use QueryPATCH instead
func (r *Router) PATCH(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleDirect("PATCH", pattern, handler)
}

// OPTIONS registers an OPTIONS route (direct, no query multiplexing)
// For query multiplexing, use QueryOPTIONS instead
func (r *Router) OPTIONS(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleDirect("OPTIONS", pattern, handler)
}

// QueryGET registers a GET route with query multiplexing support
func (r *Router) QueryGET(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleQuery("GET", pattern, handler)
}

// QueryPOST registers a POST route with query multiplexing support
func (r *Router) QueryPOST(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleQuery("POST", pattern, handler)
}

// QueryPUT registers a PUT route with query multiplexing support
func (r *Router) QueryPUT(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleQuery("PUT", pattern, handler)
}

// QueryDELETE registers a DELETE route with query multiplexing support
func (r *Router) QueryDELETE(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleQuery("DELETE", pattern, handler)
}

// QueryHEAD registers a HEAD route with query multiplexing support
func (r *Router) QueryHEAD(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleQuery("HEAD", pattern, handler)
}

// QueryPATCH registers a PATCH route with query multiplexing support
func (r *Router) QueryPATCH(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleQuery("PATCH", pattern, handler)
}

// QueryOPTIONS registers an OPTIONS route with query multiplexing support
func (r *Router) QueryOPTIONS(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handleQuery("OPTIONS", pattern, handler)
}

// handleDirect registers a route directly with Chi (no dispatcher, best performance)
// This is used by GET, POST, PUT, DELETE, etc.
// Automatically promotes to dispatcher-based routing if multiple routes exist on same method+pattern
func (r *Router) handleDirect(method, pattern string, handler http.HandlerFunc) *RouteBuilder {
	// Apply path prefix if in a group
	fullPattern := r.pathPrefix + pattern

	// Translate our {key:.*} syntax to Chi's wildcard syntax
	chiPattern, wildcardParams := core.TranslatePattern(fullPattern)

	// Create route metadata (for introspection and URL generation)
	rt := &core.Route{
		Method:         method,
		Pattern:        fullPattern,
		ChiPattern:     chiPattern,
		Handler:        handler,
		QueryMatchers:  make([]dispatch.Matcher, 0),
		Middlewares:    make([]func(http.Handler) http.Handler, 0),
		WildcardParams: wildcardParams,
	}

	// Copy group middlewares to route
	rt.Middlewares = append(rt.Middlewares, r.middlewares...)

	*r.routes = append(*r.routes, rt)

	dispatcherKey := method + ":" + chiPattern

	// Check if dispatcher already exists (from QueryGET/etc)
	if disp, exists := r.dispatchers[dispatcherKey]; exists {
		r.debugLogf("Adding direct route to existing dispatcher: %s %s", method, fullPattern)
		// Add to existing dispatcher as fallback (no query matchers)
		disp.AddRoute(rt)
		return &RouteBuilder{router: r, route: rt, dispatcher: disp}
	}

	// Check if another direct route already exists
	if existingRoute, exists := r.directRoutes[dispatcherKey]; exists {
		r.debugLogf("Auto-promoting to dispatcher: %s %s (multiple routes on same path)", method, fullPattern)
		// Promote to dispatcher!
		disp := &core.Dispatcher{Routes: make([]*core.Route, 0)}
		disp.AddRoute(existingRoute) // Old route becomes fallback
		disp.AddRoute(rt)            // New route also fallback
		r.dispatchers[dispatcherKey] = disp
		delete(r.directRoutes, dispatcherKey)

		// Register dispatcher with Chi (overwrites previous direct registration)
		r.mux.Method(method, chiPattern, disp)

		return &RouteBuilder{router: r, route: rt, dispatcher: disp}
	}

	// First route on this pattern - register directly for best performance
	optHandler := &optimizedHandler{
		route:  rt,
		router: r,
	}
	*r.optimizedHandlers = append(*r.optimizedHandlers, optHandler)

	r.mux.Method(method, chiPattern, optHandler)
	r.directRoutes[dispatcherKey] = rt

	return &RouteBuilder{router: r, route: rt, isDirect: true}
}

// handleQuery is the internal method that registers routes with query multiplexing support
// This is used by QueryGET, QueryPOST, etc.
func (r *Router) handleQuery(method, pattern string, handler http.HandlerFunc) *RouteBuilder {
	// Apply path prefix if in a group
	fullPattern := r.pathPrefix + pattern

	// Translate our {key:.*} syntax to Chi's wildcard syntax
	chiPattern, wildcardParams := core.TranslatePattern(fullPattern)

	// Create route
	rt := &core.Route{
		Method:         method,
		Pattern:        fullPattern,
		ChiPattern:     chiPattern,
		Handler:        handler,
		QueryMatchers:  make([]dispatch.Matcher, 0),
		Middlewares:    make([]func(http.Handler) http.Handler, 0),
		WildcardParams: wildcardParams,
	}

	// Copy group middlewares to route (but not global middlewares - those are handled by chi.Mux)
	rt.Middlewares = append(rt.Middlewares, r.middlewares...)

	*r.routes = append(*r.routes, rt)

	// Get or create dispatcher for this method+pattern
	dispatcherKey := method + ":" + chiPattern
	disp, exists := r.dispatchers[dispatcherKey]
	if !exists {
		// Check if a direct route exists - if so, promote it to dispatcher
		if existingRoute, directExists := r.directRoutes[dispatcherKey]; directExists {
			r.debugLogf("Auto-promoting direct route to dispatcher: %s %s (adding query-based route)", method, fullPattern)
			disp = &core.Dispatcher{Routes: make([]*core.Route, 0)}
			disp.AddRoute(existingRoute) // Existing direct route becomes fallback
			delete(r.directRoutes, dispatcherKey)
		} else {
			disp = &core.Dispatcher{Routes: make([]*core.Route, 0)}
		}
		r.dispatchers[dispatcherKey] = disp

		// Register dispatcher with Chi (overwrites any previous direct registration)
		r.mux.Method(method, chiPattern, disp)
	}

	// Add route to dispatcher
	disp.AddRoute(rt)

	return &RouteBuilder{router: r, route: rt, dispatcher: disp}
}

// Group creates a route group with a path prefix
func (r *Router) Group(pattern string, fn func(r *Router)) {
	r.NamedGroup(pattern, "", fn)
}

// MiddlewareGroup creates a route group with middleware but no path or name prefix.
// This is useful for applying middleware to a logical set of routes without changing their paths.
//
// Example:
//
//	r.MiddlewareGroup(func(r *teapot.Router) {
//	    r.GET("/admin", adminHandler).Name("admin")
//	    r.GET("/dashboard", dashHandler).Name("dashboard")
//	}, authMiddleware, loggingMiddleware)
func (r *Router) MiddlewareGroup(fn func(r *Router), middlewares ...func(http.Handler) http.Handler) {
	// Create a sub-router with same path/name prefixes but additional middlewares
	subRouter := &Router{
		mux:               r.mux, // Share the same chi mux
		directRoutes:      r.directRoutes,
		debugLog:          r.debugLog,
		routes:            r.routes,
		dispatchers:       r.dispatchers,
		nameIndex:         r.nameIndex,
		pathPrefix:        r.pathPrefix,                                                                          // No change
		namePrefix:        r.namePrefix,                                                                          // No change
		middlewares:       append(append([]func(http.Handler) http.Handler{}, r.middlewares...), middlewares...), // Parent + new
		optimizedHandlers: r.optimizedHandlers,
		finalized:         r.finalized,
	}

	fn(subRouter)
}

// NamedGroup creates a route group with path and name prefixes
func (r *Router) NamedGroup(pattern, namePrefix string, fn func(r *Router)) {
	// Create a sub-router with prefixes
	subRouter := &Router{
		mux:               r.mux, // Share the same chi mux
		routes:            r.routes,
		dispatchers:       r.dispatchers,
		directRoutes:      r.directRoutes,
		nameIndex:         r.nameIndex,
		pathPrefix:        r.pathPrefix + pattern,
		namePrefix:        r.namePrefix + namePrefix + ".",
		middlewares:       append([]func(http.Handler) http.Handler{}, r.middlewares...), // Copy parent middlewares
		optimizedHandlers: r.optimizedHandlers,
		debugLog:          r.debugLog,
	}

	// Trim trailing dot if namePrefix is empty
	if namePrefix == "" {
		subRouter.namePrefix = r.namePrefix
	}

	fn(subRouter)
}

// Route creates a routing group with Chi's Route() semantics. This is the RECOMMENDED way
// to structure your app when using RouteContextMiddleware, as middleware added inside
// Route() blocks has access to Chi's RouteContext (fast path).
//
// Example:
//
//	r.Route("/", func(r *teapot.Router) {
//	    r.Use(teapot.RouteContextMiddleware(r))  // Fast path - RouteContext available
//	    r.Use(loggingMiddleware)                 // Has access to route metadata
//	    // register all routes...
//	})
//
// This wraps Chi's Route() method and provides a teapot Router to the function,
// making it easy to use teapot's route registration methods while benefiting from
// Chi's routing context.
func (r *Router) Route(pattern string, fn func(r *Router)) {
	r.mux.Route(pattern, func(chiRouter chi.Router) {
		// Create a teapot Router that uses the Chi sub-router
		chiMux, ok := chiRouter.(*chi.Mux)
		if !ok {
			panic("teapot: Route() expected *chi.Mux from chi.Route()")
		}

		subRouter := &Router{
			mux:               chiMux,
			routes:            r.routes,            // Shared
			dispatchers:       r.dispatchers,       // Shared
			directRoutes:      r.directRoutes,      // Shared
			nameIndex:         r.nameIndex,         // Shared
			pathPrefix:        r.pathPrefix,        // No change (pattern is handled by Chi)
			namePrefix:        r.namePrefix,        // No change
			middlewares:       r.middlewares,       // Shared
			optimizedHandlers: r.optimizedHandlers, // Shared
			finalized:         r.finalized,
			debugLog:          r.debugLog,
		}

		fn(subRouter)
	})
}

// Use adds global middleware to the router
func (r *Router) Use(middlewares ...func(http.Handler) http.Handler) {
	// Only add to chi.Mux for truly global middleware
	// Don't add to r.middlewares as that would duplicate with route-specific
	r.mux.Use(middlewares...)
}

// findMatchingRoute manually matches a request against registered routes
// This is used as a fallback when Chi's RouteContext isn't available (e.g., in global middleware)
func (r *Router) findMatchingRoute(method, path string) *core.Route {
	// Check all direct routes
	for key, route := range r.directRoutes {
		if strings.HasPrefix(key, method+":") {
			pattern := strings.TrimPrefix(key, method+":")
			if r.matchPattern(pattern, path) {
				return route
			}
		}
	}

	// Check dispatcher routes (return fallback route for query-multiplexed)
	for key, disp := range r.dispatchers {
		if strings.HasPrefix(key, method+":") {
			pattern := strings.TrimPrefix(key, method+":")
			if r.matchPattern(pattern, path) {
				// Return fallback route (no query matchers)
				for _, rt := range disp.Routes {
					if len(rt.QueryMatchers) == 0 {
						return rt
					}
				}
				// If no fallback, return first route
				if len(disp.Routes) > 0 {
					return disp.Routes[0]
				}
			}
		}
	}

	return nil
}

// matchPattern checks if a Chi pattern matches a path
// Supports Chi patterns like /users/{id}, /users/{id}/posts, /{bucket}/{key}/*
func (r *Router) matchPattern(pattern, path string) bool {
	// Exact match
	if pattern == path {
		return true
	}

	// No parameters - must be exact match
	if !strings.Contains(pattern, "{") && !strings.HasSuffix(pattern, "/*") {
		return false
	}

	// Split into segments
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	// Handle wildcard at the end (Chi's /* pattern)
	if len(patternParts) > 0 && patternParts[len(patternParts)-1] == "*" {
		// Path must have at least as many parts as pattern (minus the wildcard)
		if len(pathParts) < len(patternParts)-1 {
			return false
		}
		// Match up to the wildcard
		patternParts = patternParts[:len(patternParts)-1]
		pathParts = pathParts[:len(patternParts)]
	}

	// Different number of segments
	if len(patternParts) != len(pathParts) {
		return false
	}

	// Match each segment
	for i := 0; i < len(patternParts); i++ {
		patternPart := patternParts[i]
		pathPart := pathParts[i]

		// Parameter segment (matches anything)
		if strings.HasPrefix(patternPart, "{") && strings.HasSuffix(patternPart, "}") {
			continue
		}

		// Literal segment (must match exactly)
		if patternPart != pathPart {
			return false
		}
	}

	return true
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// URL generates a URL for a named route with the given parameters
func (r *Router) URL(name string, params ...string) (string, error) {
	rt, exists := r.nameIndex[name]
	if !exists {
		return "", fmt.Errorf("route %q not found", name)
	}

	if len(params)%2 != 0 {
		return "", fmt.Errorf("params must be key-value pairs")
	}

	// Build parameter map
	paramMap := make(map[string]string)
	for i := 0; i < len(params); i += 2 {
		paramMap[params[i]] = params[i+1]
	}

	// Replace parameters in pattern
	url := rt.Pattern
	for key, value := range paramMap {
		// Handle both {key} and {key:.*} formats
		url = strings.ReplaceAll(url, "{"+key+"}", value)
		url = strings.ReplaceAll(url, "{"+key+":.*}", value)
	}

	// Check if any parameters remain unreplaced
	if strings.Contains(url, "{") {
		return "", fmt.Errorf("missing parameters for route %q", name)
	}

	return url, nil
}

// MustURL is like URL but panics on error
func (r *Router) MustURL(name string, params ...string) string {
	url, err := r.URL(name, params...)
	if err != nil {
		panic(err)
	}
	return url
}

// RouteInfo contains information about a registered route
type RouteInfo struct {
	Method      string
	Pattern     string
	Name        string
	Action      string
	QueryParams []QueryParam
}

// QueryParam represents a query parameter matcher for a route
type QueryParam struct {
	Key   string
	Value string // Empty string means "any value" (existence check only)
}

// Routes returns information about all registered routes
func (r *Router) Routes() []RouteInfo {
	var infos []RouteInfo
	for _, rt := range *r.routes {
		// Extract query parameter information from QueryMatchers
		var queryParams []QueryParam
		for _, matcher := range rt.QueryMatchers {
			switch m := matcher.(type) {
			case dispatch.QueryExistsMatcher:
				queryParams = append(queryParams, QueryParam{
					Key:   m.Key,
					Value: "", // Empty means existence check
				})
			case dispatch.QueryValueMatcher:
				queryParams = append(queryParams, QueryParam{
					Key:   m.Key,
					Value: m.Value,
				})
			}
		}

		infos = append(infos, RouteInfo{
			Method:      rt.Method,
			Pattern:     rt.Pattern,
			Name:        rt.Name,
			Action:      rt.Action,
			QueryParams: queryParams,
		})
	}
	return infos
}

// GetAction retrieves the S3 action from the request context
func GetAction(r *http.Request) string {
	return core.GetAction(r.Context())
}

// GetRouteName retrieves the route name from the request context
func GetRouteName(r *http.Request) string {
	return core.GetRouteName(r.Context())
}

// URLParam retrieves a URL parameter value (delegates to chi)
func URLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// URLParams returns all URL parameters from a request's context as a map.
// This provides a convenient way to get all parameters at once.
func URLParams(r *http.Request) map[string]string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return nil
	}
	params := make(map[string]string, len(rctx.URLParams.Keys))
	for i, key := range rctx.URLParams.Keys {
		if i < len(rctx.URLParams.Values) {
			params[key] = rctx.URLParams.Values[i]
		}
	}
	return params
}

// Finalize optimizes all direct routes for maximum performance
// Call this after registering all routes and before serving requests
// This pre-computes handlers based on actual route configuration
func (r *Router) Finalize() {
	if r.finalized {
		return // Already finalized
	}

	// Optimize all direct handlers
	for _, oh := range *r.optimizedHandlers {
		oh.fastPath = r.createOptimizedHandler(oh.route)
		oh.finalized.Store(true)
	}

	// Eagerly build all dispatchers so the lazy-build doesn't land on the
	// first request
	for _, disp := range r.dispatchers {
		disp.Build()
	}

	r.finalized = true
}

// IsFinalized returns whether the router has been finalized
func (r *Router) IsFinalized() bool {
	return r.finalized
}

// Chi returns the underlying chi.Mux for advanced use cases.
// This provides an escape hatch for Chi-specific features like:
//   - Custom error handlers (NotFound, MethodNotAllowed)
//   - Direct access to Chi's routing tree
//   - Advanced middleware patterns
func (r *Router) Chi() chi.Router {
	return r.mux
}

// With returns a new Router that applies additional middleware to subsequent routes.
// The new router shares the same route registry and uses Chi's With() for middleware isolation.
// This is useful for applying middleware to a subset of routes without affecting others.
//
// Example:
//
//	r.With(authMiddleware).GET("/admin", handler).Name("admin")
//	r.With(authMiddleware, loggingMiddleware).GET("/api", handler).Name("api")
//
// Note: Chi's With() returns chi.Router interface. We assert back to *chi.Mux.
func (r *Router) With(middlewares ...func(http.Handler) http.Handler) *Router {
	chiRouter := r.mux.With(middlewares...)

	// Chi's With() should return *chi.Mux when called on *chi.Mux
	mux, ok := chiRouter.(*chi.Mux)
	if !ok {
		panic("teapot: Router.With() expected *chi.Mux from chi.With(), got different type")
	}

	return &Router{
		mux:               mux,
		routes:            r.routes,            // Shared registry
		dispatchers:       r.dispatchers,       // Shared dispatchers
		directRoutes:      r.directRoutes,      // Shared direct routes tracker
		nameIndex:         r.nameIndex,         // Shared name index
		pathPrefix:        r.pathPrefix,        // Preserve path prefix
		namePrefix:        r.namePrefix,        // Preserve name prefix
		middlewares:       r.middlewares,       // Don't duplicate - Chi handles it
		optimizedHandlers: r.optimizedHandlers, // Shared handlers
		finalized:         r.finalized,
		debugLog:          r.debugLog, // Preserve debug logging
	}
}
