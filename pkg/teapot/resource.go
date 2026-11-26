package teapot

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jinzhu/inflection"
)

// ResourceOption configures a resource registered with [Router.Resource].
type ResourceOption func(*resourceConfig)

// resourceConfig collects the options given to Resource. It is internal: the
// public surface is the option constructors below.
type resourceConfig struct {
	path       string
	param      string
	storeVerb  HTTPMethod
	only       []string
	except     []string
	names      map[string]string
	middleware []func(http.Handler) http.Handler
}

// Only limits the resource to the named actions (for example "index", "show").
// Panics if combined with Except, or if an action name is unknown.
func Only(actions ...string) ResourceOption {
	return func(c *resourceConfig) { c.only = append(c.only, actions...) }
}

// Except removes the named actions from the resource. Panics if an action
// name is unknown.
func Except(actions ...string) ResourceOption {
	return func(c *resourceConfig) { c.except = append(c.except, actions...) }
}

// Middleware applies middleware to every route the resource registers. It is
// scoped to this resource only, not to the rest of the router.
func Middleware(mws ...func(http.Handler) http.Handler) ResourceOption {
	return func(c *resourceConfig) { c.middleware = append(c.middleware, mws...) }
}

// Path sets the URL path for the resource. The default is "/" plus the name.
func Path(p string) ResourceOption {
	return func(c *resourceConfig) { c.path = p }
}

// Param sets the name of the route parameter for single-item routes. The
// default is the singular form of the resource name.
func Param(name string) ResourceOption {
	return func(c *resourceConfig) { c.param = name }
}

// Names overrides the route name for individual actions, keyed by action name.
// For example, Names(map[string]string{"index": "list"}) names the index route
// "photos.list" instead of "photos.index".
func Names(names map[string]string) ResourceOption {
	return func(c *resourceConfig) { c.names = names }
}

// StoreVerb sets the HTTP method used to create a resource. Use [PUT] for
// S3-style creation. The default is [POST].
func StoreVerb(m HTTPMethod) ResourceOption {
	return func(c *resourceConfig) { c.storeVerb = m }
}

// Optional action interfaces. A controller implements the actions it supports;
// an action whose method is missing is not registered.
type (
	indexer interface {
		Index(http.ResponseWriter, *http.Request)
	}
	creator interface {
		Create(http.ResponseWriter, *http.Request)
	}
	storer interface {
		Store(http.ResponseWriter, *http.Request)
	}
	shower interface {
		Show(http.ResponseWriter, *http.Request)
	}
	editor interface {
		Edit(http.ResponseWriter, *http.Request)
	}
	updater interface {
		Update(http.ResponseWriter, *http.Request)
	}
	destroyer interface {
		Destroy(http.ResponseWriter, *http.Request)
	}
)

// resourceAction describes one standard action and the routes it produces.
type resourceAction struct {
	name    string
	methods []HTTPMethod
	suffix  string // appended to the collection path or item path
	item    bool   // true when the route is on the item path (/{param})
}

// resourceActions lists the standard actions in registration order.
var resourceActions = []resourceAction{
	{name: "index", methods: []HTTPMethod{HTTPMethod(http.MethodGet)}, suffix: "", item: false},
	{name: "create", methods: []HTTPMethod{HTTPMethod(http.MethodGet)}, suffix: "/create", item: false},
	{name: "store", methods: nil, suffix: "", item: false}, // method comes from StoreVerb
	{name: "show", methods: []HTTPMethod{HTTPMethod(http.MethodGet)}, suffix: "", item: true},
	{name: "edit", methods: []HTTPMethod{HTTPMethod(http.MethodGet)}, suffix: "/edit", item: true},
	{name: "update", methods: []HTTPMethod{PUT, HTTPMethod(http.MethodPatch)}, suffix: "", item: true},
	{name: "destroy", methods: []HTTPMethod{HTTPMethod(http.MethodDelete)}, suffix: "", item: true},
}

// Resource registers the RESTful routes for controller under the given name.
//
// The controller is any value. Each action is registered only if the
// controller has the matching method: Index, Create, Store, Show, Edit,
// Update, or Destroy.
//
// Routes created (with the defaults):
//   - GET    /photos             -> photos.index
//   - GET    /photos/create      -> photos.create
//   - POST   /photos             -> photos.store   (StoreVerb changes the method)
//   - GET    /photos/{photo}     -> photos.show
//   - GET    /photos/{photo}/edit -> photos.edit
//   - PUT    /photos/{photo}     -> photos.update
//   - PATCH  /photos/{photo}     -> photos.update
//   - DELETE /photos/{photo}     -> photos.destroy
//
// Options refine the defaults. Resource panics on invalid input.
//
// Example:
//
//	r.Resource("photos", &PhotoController{db: db},
//	    teapot.Only("index", "show"),
//	    teapot.Middleware(authMiddleware),
//	)
//
// Example (S3-style creation with PUT):
//
//	r.Resource("buckets", &BucketController{}, teapot.StoreVerb(teapot.PUT))
func (r *Router) Resource(name string, controller any, opts ...ResourceOption) {
	cfg := resourceConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	path := cfg.path
	if path == "" {
		path = "/" + name
	}
	if !strings.HasPrefix(path, "/") {
		panic(fmt.Sprintf("teapot: Resource %q: Path must start with \"/\", got %q", name, path))
	}

	param := cfg.param
	if param == "" {
		param = inflection.Singular(name)
	}

	storeVerb := cfg.storeVerb
	if storeVerb == "" {
		storeVerb = POST
	}
	if storeVerb != POST && storeVerb != PUT {
		panic(fmt.Sprintf("teapot: Resource %q: StoreVerb must be POST or PUT, got %q", name, storeVerb))
	}

	if len(cfg.only) > 0 && len(cfg.except) > 0 {
		panic(fmt.Sprintf("teapot: Resource %q: Only and Except cannot be combined", name))
	}
	for _, a := range append(append([]string{}, cfg.only...), cfg.except...) {
		if !isResourceAction(a) {
			panic(fmt.Sprintf("teapot: Resource %q: unknown action %q", name, a))
		}
	}
	for a := range cfg.names {
		if !isResourceAction(a) {
			panic(fmt.Sprintf("teapot: Resource %q: Names refers to unknown action %q", name, a))
		}
	}

	if controller == nil {
		panic(fmt.Sprintf("teapot: Resource %q: controller must not be nil", name))
	}

	handlers := resourceHandlers(controller)

	r.NamedGroup(path, name, func(g *Router) {
		g.Use(cfg.middleware...)

		for _, action := range resourceActions {
			if !actionEnabled(action.name, cfg) {
				continue
			}
			h, ok := handlers[action.name]
			if !ok {
				continue
			}

			methods := action.methods
			if action.name == "store" {
				methods = []HTTPMethod{storeVerb}
			}

			routeName := action.name
			if n, ok := cfg.names[action.name]; ok {
				routeName = n
			}

			pattern := action.suffix
			if action.item {
				pattern = "/{" + param + "}" + action.suffix
			}

			for _, m := range methods {
				g.Handle(string(m), pattern, h).Name(routeName)
			}
		}
	})
}

// isResourceAction reports whether name is one of the standard actions.
func isResourceAction(name string) bool {
	for _, a := range resourceActions {
		if a.name == name {
			return true
		}
	}
	return false
}

// actionEnabled applies Only and Except to a single action.
func actionEnabled(name string, cfg resourceConfig) bool {
	if len(cfg.only) > 0 {
		return contains(cfg.only, name)
	}
	return !contains(cfg.except, name)
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// resourceHandlers resolves the actions a controller implements into handlers,
// keyed by action name.
func resourceHandlers(controller any) map[string]http.Handler {
	h := make(map[string]http.Handler)
	if c, ok := controller.(indexer); ok {
		h["index"] = http.HandlerFunc(c.Index)
	}
	if c, ok := controller.(creator); ok {
		h["create"] = http.HandlerFunc(c.Create)
	}
	if c, ok := controller.(storer); ok {
		h["store"] = http.HandlerFunc(c.Store)
	}
	if c, ok := controller.(shower); ok {
		h["show"] = http.HandlerFunc(c.Show)
	}
	if c, ok := controller.(editor); ok {
		h["edit"] = http.HandlerFunc(c.Edit)
	}
	if c, ok := controller.(updater); ok {
		h["update"] = http.HandlerFunc(c.Update)
	}
	if c, ok := controller.(destroyer); ok {
		h["destroy"] = http.HandlerFunc(c.Destroy)
	}
	return h
}
