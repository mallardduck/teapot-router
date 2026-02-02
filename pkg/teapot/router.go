package teapot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mallardduck/teapot-router/internal/core"
)

// Router wraps chi.Mux and adds named routes, query multiplexing, and S3 actions
type Router struct {
	mux               *chi.Mux
	routes            *[]*core.Route
	dispatchers       map[string]*core.Dispatcher // key: "METHOD:PATTERN"
	nameIndex         map[string]*core.Route      // for URL generation
	pathPrefix        string
	namePrefix        string
	middlewares       []func(http.Handler) http.Handler
	optimizedHandlers *[]*optimizedHandler // for finalization optimization
	finalized         bool
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

// ResourceHandlers defines handlers for RESTful resource operations
type ResourceHandlers struct {
	// Index lists all resources (GET /photos -> photos.index)
	Index http.HandlerFunc
	// Create shows the form to create a new resource (GET /photos/create -> photos.create)
	Create http.HandlerFunc
	// Store creates a new resource (POST/PUT /photos -> photos.store)
	Store http.HandlerFunc
	// Show displays a specific resource (GET /photos/{id} -> photos.show)
	Show http.HandlerFunc
	// Edit shows the form to edit a resource (GET /photos/{id}/edit -> photos.edit)
	Edit http.HandlerFunc
	// Update modifies a resource (PUT/POST /photos/{id} -> photos.update)
	Update http.HandlerFunc
	// Destroy deletes a resource (DELETE /photos/{id} -> photos.destroy)
	Destroy http.HandlerFunc
	// Head retrieves resource metadata (HEAD /photos/{id} -> photos.head)
	Head http.HandlerFunc

	// StoreMethod specifies the HTTP method for Store (default: POST for REST, use PUT for S3)
	StoreMethod HTTPMethod
	// UpdateMethod specifies the HTTP method for Update (default: PUT for REST, use POST if needed)
	UpdateMethod HTTPMethod
}

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
	rb.route.QueryMatchers = append(rb.route.QueryMatchers, core.QueryExistsMatcher{Key: key})
	rb.dispatcher.UpdateSpecificity()
	return rb
}

// QueryValue adds a query parameter value matcher
func (rb *RouteBuilder) QueryValue(key, value string) *RouteBuilder {
	if rb.isDirect {
		panic("teapot: Cannot use .QueryValue() with standard methods (GET, POST, etc). Use QueryGET, QueryPOST, etc. instead for query multiplexing.")
	}
	rb.route.QueryMatchers = append(rb.route.QueryMatchers, core.QueryValueMatcher{Key: key, Value: value})
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
		nameIndex:         make(map[string]*core.Route),
		optimizedHandlers: &optimizedHandlers,
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
// Does not support query multiplexing
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
		QueryMatchers:  make([]core.QueryMatcher, 0),
		Middlewares:    make([]func(http.Handler) http.Handler, 0),
		WildcardParams: wildcardParams,
	}

	// Copy group middlewares to route
	rt.Middlewares = append(rt.Middlewares, r.middlewares...)

	*r.routes = append(*r.routes, rt)

	// Create optimized handler (can be finalized later)
	optHandler := &optimizedHandler{
		route:  rt,
		router: r,
	}
	*r.optimizedHandlers = append(*r.optimizedHandlers, optHandler)

	// Register with Chi
	r.mux.Method(method, chiPattern, optHandler)

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
		QueryMatchers:  make([]core.QueryMatcher, 0),
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
		disp = &core.Dispatcher{Routes: make([]*core.Route, 0)}
		r.dispatchers[dispatcherKey] = disp

		// Register dispatcher with Chi immediately (using Chi pattern)
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
		nameIndex:         r.nameIndex,
		pathPrefix:        r.pathPrefix + pattern,
		namePrefix:        r.namePrefix + namePrefix + ".",
		middlewares:       append([]func(http.Handler) http.Handler{}, r.middlewares...), // Copy parent middlewares
		optimizedHandlers: r.optimizedHandlers,
	}

	// Trim trailing dot if namePrefix is empty
	if namePrefix == "" {
		subRouter.namePrefix = r.namePrefix
	}

	fn(subRouter)
}

// Resource creates RESTful resource routes following Laravel/Rails conventions.
// This is a convenience method for scaffolding standard CRUD operations.
//
// Routes created:
//   - GET    /path              -> name.index   (Index handler)
//   - GET    /path/create       -> name.create  (Create handler)
//   - POST   /path              -> name.store   (Store handler, or PUT if StoreMethod = PUT)
//   - GET    /path/{param}      -> name.show    (Show handler)
//   - GET    /path/{param}/edit -> name.edit    (Edit handler)
//   - PUT    /path/{param}      -> name.update  (Update handler, or POST if UpdateMethod = POST)
//   - DELETE /path/{param}      -> name.destroy (Destroy handler)
//   - HEAD   /path/{param}      -> name.head    (Head handler)
//
// Example (REST-style):
//
//	r.Resource("photos", "/photos", "photo", teapot.ResourceHandlers{
//	    Index:   listPhotos,
//	    Store:   createPhoto,
//	    Show:    showPhoto,
//	    Update:  updatePhoto,
//	    Destroy: deletePhoto,
//	})
//
// Example (S3-style with PUT for creation):
//
//	r.Resource("buckets", "/buckets", "bucket", teapot.ResourceHandlers{
//	    Index:   listBuckets,
//	    Store:   createBucket,
//	    Show:    getBucket,
//	    Destroy: deleteBucket,
//	    StoreMethod: teapot.PUT,  // S3 uses PUT to create buckets
//	})
func (r *Router) Resource(name, path, param string, handlers ResourceHandlers) {
	// Determine HTTP methods with defaults
	storeMethod := handlers.StoreMethod
	if storeMethod == "" {
		storeMethod = POST // Default: REST-style POST for creation
	}

	updateMethod := handlers.UpdateMethod
	if updateMethod == "" {
		updateMethod = PUT // Default: REST-style PUT for updates
	}

	// Register routes (only if handler is provided)
	if handlers.Index != nil {
		r.GET(path, handlers.Index).Name(name + ".index")
	}

	if handlers.Create != nil {
		r.GET(path+"/create", handlers.Create).Name(name + ".create")
	}

	if handlers.Store != nil {
		switch storeMethod {
		case POST:
			r.POST(path, handlers.Store).Name(name + ".store")
		case PUT:
			r.PUT(path, handlers.Store).Name(name + ".store")
		}
	}

	if handlers.Show != nil {
		r.GET(path+"/{"+param+"}", handlers.Show).Name(name + ".show")
	}

	if handlers.Edit != nil {
		r.GET(path+"/{"+param+"}/edit", handlers.Edit).Name(name + ".edit")
	}

	if handlers.Update != nil {
		switch updateMethod {
		case PUT:
			r.PUT(path+"/{"+param+"}", handlers.Update).Name(name + ".update")
		case POST:
			r.POST(path+"/{"+param+"}", handlers.Update).Name(name + ".update")
		}
	}

	if handlers.Destroy != nil {
		r.DELETE(path+"/{"+param+"}", handlers.Destroy).Name(name + ".destroy")
	}

	if handlers.Head != nil {
		r.HEAD(path+"/{"+param+"}", handlers.Head).Name(name + ".head")
	}
}

// Use adds global middleware to the router
func (r *Router) Use(middlewares ...func(http.Handler) http.Handler) {
	// Only add to chi.Mux for truly global middleware
	// Don't add to r.middlewares as that would duplicate with route-specific
	r.mux.Use(middlewares...)
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
			case core.QueryExistsMatcher:
				queryParams = append(queryParams, QueryParam{
					Key:   m.Key,
					Value: "", // Empty means existence check
				})
			case core.QueryValueMatcher:
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

// RoutesHandler returns an HTTP handler that displays all registered routes.
// The handler responds with JSON or HTML based on the Accept header.
//
// Example:
//
//	if debug {
//	    r.GET("/.internal/routes", r.RoutesHandler()).Name("debug.routes")
//	}
func (r *Router) RoutesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		routes := r.Routes()

		// Check Accept header for JSON vs HTML
		accept := req.Header.Get("Accept")
		if strings.Contains(accept, "application/json") {
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(map[string]any{
				"count":  len(routes),
				"routes": routes,
			})
			if err != nil {
				// todo log error
				return
			}
			return
		}

		// Default to HTML
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Routes</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; margin: 2rem; }
        h1 { color: #333; }
        table { border-collapse: collapse; width: 100%%; margin-top: 1rem; }
        th, td { text-align: left; padding: 0.75rem; border-bottom: 1px solid #ddd; }
        th { background-color: #f5f5f5; font-weight: 600; }
        tr:hover { background-color: #f9f9f9; }
        .method { font-family: monospace; font-weight: 600; }
        .pattern { font-family: monospace; color: #0066cc; }
        .name { color: #666; }
        .action { color: #888; font-size: 0.9em; }
        .count { color: #666; font-size: 0.9em; }
        .get { color: #28a745; }
        .post { color: #007bff; }
        .put { color: #ffc107; }
        .delete { color: #dc3545; }
        .head { color: #6c757d; }
        .patch { color: #17a2b8; }
        .options { color: #6610f2; }
    </style>
</head>
<body>
    <h1>Registered Routes</h1>
    <p class="count">Total: %d routes</p>
    <table>
        <thead>
            <tr>
                <th>Method</th>
                <th>Pattern</th>
                <th>Name</th>
                <th>Action</th>
            </tr>
        </thead>
        <tbody>
`, len(routes))

		for _, route := range routes {
			methodClass := strings.ToLower(route.Method)
			name := route.Name
			if name == "" {
				name = "-"
			}
			action := route.Action
			if action == "" {
				action = "-"
			}

			_, _ = fmt.Fprintf(w, `            <tr>
                <td class="method %s">%s</td>
                <td class="pattern">%s</td>
                <td class="name">%s</td>
                <td class="action">%s</td>
            </tr>
`, methodClass, route.Method, route.Pattern, name, action)
		}

		_, _ = fmt.Fprintf(w, `        </tbody>
    </table>
</body>
</html>`)
	}
}

// RegisterDebugRoute is a convenience method to register a debug endpoint that shows all routes.
// This is useful for development and debugging.
//
// Example:
//
//	if debug {
//	    r.RegisterDebugRoute("/.internal/routes", "debug.routes")
//	}
func (r *Router) RegisterDebugRoute(path, name string) *RouteBuilder {
	return r.GET(path, r.RoutesHandler()).Name(name)
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
		nameIndex:         r.nameIndex,         // Shared name index
		pathPrefix:        r.pathPrefix,        // Preserve path prefix
		namePrefix:        r.namePrefix,        // Preserve name prefix
		middlewares:       r.middlewares,       // Don't duplicate - Chi handles it
		optimizedHandlers: r.optimizedHandlers, // Shared handlers
		finalized:         r.finalized,
	}
}
