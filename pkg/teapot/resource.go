package teapot

import (
	"fmt"
	"net/http"
)

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
// Resource panics if StoreMethod is not POST or PUT, or if UpdateMethod is not
// PUT or POST. An empty value selects the default.
//
// Example:
//
//	r.Resource("photos", "/photos", "photo", teapot.ResourceHandlers{
//	    Index: listPhotos,       // any http.Handler, e.g. http.HandlerFunc(fn)
//	    Show:  myPhotoStruct,    // struct implementing http.Handler
//	})
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
func (r *Router) Resource(name, path, param string, h ResourceHandlers) {
	// Determine HTTP methods with defaults
	storeMethod := h.StoreMethod
	if storeMethod == "" {
		storeMethod = POST // Default: REST-style POST for creation
	}

	updateMethod := h.UpdateMethod
	if updateMethod == "" {
		updateMethod = PUT // Default: REST-style PUT for updates
	}

	// Reject anything else up front. Without this, a typo or unsupported method
	// would silently drop the route.
	if storeMethod != POST && storeMethod != PUT {
		panic(fmt.Sprintf("teapot: Resource %q: StoreMethod must be POST or PUT, got %q", name, storeMethod))
	}
	if updateMethod != PUT && updateMethod != POST {
		panic(fmt.Sprintf("teapot: Resource %q: UpdateMethod must be PUT or POST, got %q", name, updateMethod))
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
