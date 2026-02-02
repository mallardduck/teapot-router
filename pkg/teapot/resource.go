package teapot

import "net/http"

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
