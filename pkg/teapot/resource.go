package teapot

import "net/http"

// ResourceOption configures a resource registered with [Router.Resource].
type ResourceOption func(*resourceConfig)

// resourceConfig is the internal, resolved description of one resource.
// It is built from the controller and the options, then turned into routes.
type resourceConfig struct{}

// Only limits the resource to the named actions (for example "index", "show").
//
// STUB: not implemented yet.
func Only(actions ...string) ResourceOption {
	return func(*resourceConfig) {}
}

// Except removes the named actions from the resource.
//
// STUB: not implemented yet.
func Except(actions ...string) ResourceOption {
	return func(*resourceConfig) {}
}

// Middleware applies middleware to every route the resource registers.
//
// STUB: not implemented yet.
func Middleware(mws ...func(http.Handler) http.Handler) ResourceOption {
	return func(*resourceConfig) {}
}

// Path sets the URL path for the resource. The default is "/" plus the name.
//
// STUB: not implemented yet.
func Path(p string) ResourceOption {
	return func(*resourceConfig) {}
}

// Param sets the name of the route parameter for single-item routes. The
// default is the singular form of the resource name.
//
// STUB: not implemented yet.
func Param(name string) ResourceOption {
	return func(*resourceConfig) {}
}

// Names overrides the route name for individual actions, keyed by action name.
//
// STUB: not implemented yet.
func Names(names map[string]string) ResourceOption {
	return func(*resourceConfig) {}
}

// StoreVerb sets the HTTP method used to create a resource. Use [PUT] for
// S3-style creation. The default is [POST].
//
// STUB: not implemented yet.
func StoreVerb(m HTTPMethod) ResourceOption {
	return func(*resourceConfig) {}
}

// Resource registers the RESTful routes for controller under the given name.
//
// The controller is any value. Each action is registered only if the
// controller has the matching method: Index, Create, Store, Show, Edit,
// Update, or Destroy.
//
// STUB: not implemented yet. Registers nothing.
func (r *Router) Resource(name string, controller any, opts ...ResourceOption) {
}
