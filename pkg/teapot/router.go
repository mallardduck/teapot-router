package teapot

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mallardduck/teapot-router/internal/core"
)

// Router wraps chi.Mux and adds named routes, query multiplexing, and S3 actions
type Router struct {
	mux         *chi.Mux
	routes      []*core.Route
	dispatchers map[string]*core.Dispatcher // key: "METHOD:PATTERN"
	nameIndex   map[string]*core.Route      // for URL generation
	pathPrefix  string
	namePrefix  string
	middlewares []func(http.Handler) http.Handler
}

// RouteBuilder provides a fluent API for building routes
type RouteBuilder struct {
	router     *Router
	route      *core.Route
	dispatcher *core.Dispatcher
}

// Name assigns a name to the route for URL generation
func (rb *RouteBuilder) Name(name string) *RouteBuilder {
	fullName := rb.router.namePrefix + name
	rb.route.Name = fullName

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
	rb.route.QueryMatchers = append(rb.route.QueryMatchers, core.QueryExistsMatcher{Key: key})
	rb.dispatcher.UpdateSpecificity()
	return rb
}

// QueryValue adds a query parameter value matcher
func (rb *RouteBuilder) QueryValue(key, value string) *RouteBuilder {
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
	return &Router{
		mux:         chi.NewRouter(),
		routes:      make([]*core.Route, 0),
		dispatchers: make(map[string]*core.Dispatcher),
		nameIndex:   make(map[string]*core.Route),
	}
}

// GET registers a GET route
func (r *Router) GET(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handle("GET", pattern, handler)
}

// POST registers a POST route
func (r *Router) POST(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handle("POST", pattern, handler)
}

// PUT registers a PUT route
func (r *Router) PUT(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handle("PUT", pattern, handler)
}

// DELETE registers a DELETE route
func (r *Router) DELETE(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handle("DELETE", pattern, handler)
}

// HEAD registers a HEAD route
func (r *Router) HEAD(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handle("HEAD", pattern, handler)
}

// PATCH registers a PATCH route
func (r *Router) PATCH(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handle("PATCH", pattern, handler)
}

// OPTIONS registers an OPTIONS route
func (r *Router) OPTIONS(pattern string, handler http.HandlerFunc) *RouteBuilder {
	return r.handle("OPTIONS", pattern, handler)
}

// handle is the internal method that registers routes
func (r *Router) handle(method, pattern string, handler http.HandlerFunc) *RouteBuilder {
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

	r.routes = append(r.routes, rt)

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

// NamedGroup creates a route group with path and name prefixes
func (r *Router) NamedGroup(pattern, namePrefix string, fn func(r *Router)) {
	// Create a sub-router with prefixes
	subRouter := &Router{
		mux:         r.mux, // Share the same chi mux
		routes:      r.routes,
		dispatchers: r.dispatchers,
		nameIndex:   r.nameIndex,
		pathPrefix:  r.pathPrefix + pattern,
		namePrefix:  r.namePrefix + namePrefix + ".",
		middlewares: append([]func(http.Handler) http.Handler{}, r.middlewares...), // Copy parent middlewares
	}

	// Trim trailing dot if namePrefix is empty
	if namePrefix == "" {
		subRouter.namePrefix = r.namePrefix
	}

	fn(subRouter)
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
	Method  string
	Pattern string
	Name    string
	Action  string
}

// Routes returns information about all registered routes
func (r *Router) Routes() []RouteInfo {
	var infos []RouteInfo
	for _, rt := range r.routes {
		infos = append(infos, RouteInfo{
			Method:  rt.Method,
			Pattern: rt.Pattern,
			Name:    rt.Name,
			Action:  rt.Action,
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
