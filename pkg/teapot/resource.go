package teapot

import "net/http"

// ResourceHandlers defines handlers for RESTful resource operations
type ResourceHandlers struct {
	// Index lists all resources (GET /photos -> photos.index)
	Index http.Handler
	// Create shows the form to create a new resource (GET /photos/create -> photos.create)
	Create http.Handler
	// Store creates a new resource (POST/PUT /photos -> photos.store)
	Store http.Handler
	// Show displays a specific resource (GET /photos/{id} -> photos.show)
	Show http.Handler
	// Edit shows the form to edit a resource (GET /photos/{id}/edit -> photos.edit)
	Edit http.Handler
	// Update modifies a resource (PUT/POST /photos/{id} -> photos.update)
	Update http.Handler
	// Destroy deletes a resource (DELETE /photos/{id} -> photos.destroy)
	Destroy http.Handler
	// Head retrieves resource metadata (HEAD /photos/{id} -> photos.head)
	Head http.Handler

	// StoreMethod specifies the HTTP method for Store (default: POST for REST, use PUT for S3)
	StoreMethod HTTPMethod
	// UpdateMethod specifies the HTTP method for Update (default: PUT for REST, use POST if needed)
	UpdateMethod HTTPMethod
}

// resourceHandlerProvider is an unexported interface implemented by both ResourceHandlers
// and *ResourceHandlerBuilder, allowing r.Resource() to accept either.
type resourceHandlerProvider interface {
	buildHandlers() ResourceHandlers
}

// buildHandlers implements resourceHandlerProvider for ResourceHandlers (identity).
func (rh ResourceHandlers) buildHandlers() ResourceHandlers { return rh }

// ResourceHandlerBuilder is a fluent builder for ResourceHandlers.
// Obtain via NewResourceHandlers(). Each handler field has two setters:
//   - Field(h http.Handler)                               — for handler structs or named vars
//   - FuncField(fn func(http.ResponseWriter, *http.Request)) — for inline function literals
type ResourceHandlerBuilder struct{ h ResourceHandlers }

// NewResourceHandlers returns a new ResourceHandlerBuilder for constructing
// ResourceHandlers using a fluent API, useful when mixing handler structs and
// inline function literals.
func NewResourceHandlers() *ResourceHandlerBuilder { return &ResourceHandlerBuilder{} }

// buildHandlers implements resourceHandlerProvider.
func (b *ResourceHandlerBuilder) buildHandlers() ResourceHandlers { return b.h }

// Index sets the Index handler.
func (b *ResourceHandlerBuilder) Index(h http.Handler) *ResourceHandlerBuilder {
	b.h.Index = h
	return b
}

// FuncIndex sets the Index handler using a plain function literal.
func (b *ResourceHandlerBuilder) FuncIndex(fn func(http.ResponseWriter, *http.Request)) *ResourceHandlerBuilder {
	b.h.Index = http.HandlerFunc(fn)
	return b
}

// Create sets the Create handler.
func (b *ResourceHandlerBuilder) Create(h http.Handler) *ResourceHandlerBuilder {
	b.h.Create = h
	return b
}

// FuncCreate sets the Create handler using a plain function literal.
func (b *ResourceHandlerBuilder) FuncCreate(fn func(http.ResponseWriter, *http.Request)) *ResourceHandlerBuilder {
	b.h.Create = http.HandlerFunc(fn)
	return b
}

// Store sets the Store handler.
func (b *ResourceHandlerBuilder) Store(h http.Handler) *ResourceHandlerBuilder {
	b.h.Store = h
	return b
}

// FuncStore sets the Store handler using a plain function literal.
func (b *ResourceHandlerBuilder) FuncStore(fn func(http.ResponseWriter, *http.Request)) *ResourceHandlerBuilder {
	b.h.Store = http.HandlerFunc(fn)
	return b
}

// Show sets the Show handler.
func (b *ResourceHandlerBuilder) Show(h http.Handler) *ResourceHandlerBuilder {
	b.h.Show = h
	return b
}

// FuncShow sets the Show handler using a plain function literal.
func (b *ResourceHandlerBuilder) FuncShow(fn func(http.ResponseWriter, *http.Request)) *ResourceHandlerBuilder {
	b.h.Show = http.HandlerFunc(fn)
	return b
}

// Edit sets the Edit handler.
func (b *ResourceHandlerBuilder) Edit(h http.Handler) *ResourceHandlerBuilder {
	b.h.Edit = h
	return b
}

// FuncEdit sets the Edit handler using a plain function literal.
func (b *ResourceHandlerBuilder) FuncEdit(fn func(http.ResponseWriter, *http.Request)) *ResourceHandlerBuilder {
	b.h.Edit = http.HandlerFunc(fn)
	return b
}

// Update sets the Update handler.
func (b *ResourceHandlerBuilder) Update(h http.Handler) *ResourceHandlerBuilder {
	b.h.Update = h
	return b
}

// FuncUpdate sets the Update handler using a plain function literal.
func (b *ResourceHandlerBuilder) FuncUpdate(fn func(http.ResponseWriter, *http.Request)) *ResourceHandlerBuilder {
	b.h.Update = http.HandlerFunc(fn)
	return b
}

// Destroy sets the Destroy handler.
func (b *ResourceHandlerBuilder) Destroy(h http.Handler) *ResourceHandlerBuilder {
	b.h.Destroy = h
	return b
}

// FuncDestroy sets the Destroy handler using a plain function literal.
func (b *ResourceHandlerBuilder) FuncDestroy(fn func(http.ResponseWriter, *http.Request)) *ResourceHandlerBuilder {
	b.h.Destroy = http.HandlerFunc(fn)
	return b
}

// Head sets the Head handler.
func (b *ResourceHandlerBuilder) Head(h http.Handler) *ResourceHandlerBuilder {
	b.h.Head = h
	return b
}

// FuncHead sets the Head handler using a plain function literal.
func (b *ResourceHandlerBuilder) FuncHead(fn func(http.ResponseWriter, *http.Request)) *ResourceHandlerBuilder {
	b.h.Head = http.HandlerFunc(fn)
	return b
}

// StoreMethod sets the HTTP method used for the Store route.
func (b *ResourceHandlerBuilder) StoreMethod(m HTTPMethod) *ResourceHandlerBuilder {
	b.h.StoreMethod = m
	return b
}

// UpdateMethod sets the HTTP method used for the Update route.
func (b *ResourceHandlerBuilder) UpdateMethod(m HTTPMethod) *ResourceHandlerBuilder {
	b.h.UpdateMethod = m
	return b
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
// handlers may be a ResourceHandlers struct or a *ResourceHandlerBuilder from NewResourceHandlers().
//
// Example (struct literal — handler fields now accept any http.Handler):
//
//	r.Resource("photos", "/photos", "photo", teapot.ResourceHandlers{
//	    Index:   listPhotos,      // http.HandlerFunc var satisfies http.Handler
//	    Show:    myPhotoStruct,   // struct implementing http.Handler
//	})
//
// Example (builder with inline functions):
//
//	r.Resource("photos", "/photos", "photo",
//	    teapot.NewResourceHandlers().
//	        FuncIndex(func(w http.ResponseWriter, r *http.Request) { ... }).
//	        Show(myPhotoStruct),
//	)
//
// Example (S3-style with PUT for creation):
//
//	r.Resource("buckets", "/buckets", "bucket", teapot.ResourceHandlers{
//	    Index:       listBuckets,
//	    Store:       createBucket,
//	    Show:        getBucket,
//	    Destroy:     deleteBucket,
//	    StoreMethod: teapot.PUT,  // S3 uses PUT to create buckets
//	})
func (r *Router) Resource(name, path, param string, handlers resourceHandlerProvider) {
	h := handlers.buildHandlers()

	// Determine HTTP methods with defaults
	storeMethod := h.StoreMethod
	if storeMethod == "" {
		storeMethod = POST // Default: REST-style POST for creation
	}

	updateMethod := h.UpdateMethod
	if updateMethod == "" {
		updateMethod = PUT // Default: REST-style PUT for updates
	}

	// Register routes (only if handler is provided)
	if h.Index != nil {
		r.GET(path, h.Index).Name(name + ".index")
	}

	if h.Create != nil {
		r.GET(path+"/create", h.Create).Name(name + ".create")
	}

	if h.Store != nil {
		switch storeMethod {
		case POST:
			r.POST(path, h.Store).Name(name + ".store")
		case PUT:
			r.PUT(path, h.Store).Name(name + ".store")
		}
	}

	if h.Show != nil {
		r.GET(path+"/{"+param+"}", h.Show).Name(name + ".show")
	}

	if h.Edit != nil {
		r.GET(path+"/{"+param+"}/edit", h.Edit).Name(name + ".edit")
	}

	if h.Update != nil {
		switch updateMethod {
		case PUT:
			r.PUT(path+"/{"+param+"}", h.Update).Name(name + ".update")
		case POST:
			r.POST(path+"/{"+param+"}", h.Update).Name(name + ".update")
		}
	}

	if h.Destroy != nil {
		r.DELETE(path+"/{"+param+"}", h.Destroy).Name(name + ".destroy")
	}

	if h.Head != nil {
		r.HEAD(path+"/{"+param+"}", h.Head).Name(name + ".head")
	}
}
