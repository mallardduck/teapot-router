// Package teapot is an expressive HTTP routing layer built on top of chi,
// inspired by Laravel's router.
//
// # Core Features
//
//   - Named routes with reverse URL generation ([Router.URL], [Router.MustURL])
//   - Query-parameter multiplexing — same path, different handlers based on query params
//   - S3 action context injection for S3-compatible APIs
//   - Fluent, Laravel-inspired route registration API
//   - Full compatibility with chi middleware
//
// # Basic Usage
//
//	r := teapot.New()
//
//	r.GET("/users", listUsers).Name("users.index")
//	r.GET("/users/{id}", showUser).Name("users.show")
//
//	http.ListenAndServe(":8080", r)
//
// # Named Routes and URL Generation
//
// Assign a name to any route with .Name(), then generate its URL path later
// using [Router.URL] or [Router.MustURL]. Parameters are passed as alternating
// key-value pairs that replace placeholders in the pattern:
//
//	r.GET("/users/{id}", showUser).Name("users.show")
//
//	// With error handling:
//	path, err := r.URL("users.show", "id", "42")
//	// path == "/users/42"
//
//	// Panic on error (suited to handler code):
//	path := r.MustURL("users.show", "id", "42")
//
// Wildcard segments ({key:.*}) are substituted the same way:
//
//	r.GET("/{bucket}/{key:.*}", getObject).Name("object.get")
//	path := r.MustURL("object.get", "bucket", "photos", "key", "2024/vacation.jpg")
//	// path == "/photos/2024/vacation.jpg"
//
// To produce absolute URLs, combine with the urlbuilder package:
//
//	urls := urlbuilder.New("s3.example.com")
//	fullURL := urls.BuildURL(r, path)
//	// https://s3.example.com/photos/2024/vacation.jpg
//
// # Query-Based Routing
//
// Route the same path to different handlers depending on which query parameters
// are present — essential for S3-style APIs:
//
//	r.GET("/{bucket}", listObjects).Name("bucket.list").Action("s3:ListBucket")
//	r.GET("/{bucket}", getBucketAcl).Name("bucket.acl").Action("s3:GetBucketAcl").Query("acl")
//
// Use [Router.Dispatch] to group many query variants on one path explicitly:
//
//	r.Dispatch("GET", "/{bucket}", func(d *teapot.DispatchBuilder, m teapot.Matchers) {
//	    d.Default(listObjects).Name("bucket.list").Action("s3:ListBucket")
//	    d.When(m.QueryExists("acl")).Do(getBucketAcl).Name("bucket.acl").Action("s3:GetBucketAcl")
//	})
//
// # Route Groups
//
// Group routes under a shared path prefix, with an optional name prefix:
//
//	r.NamedGroup("/{bucket}", "bucket", func(r *teapot.Router) {
//	    r.GET("", listObjects).Name("list")      // registered as "bucket.list"
//	    r.NamedGroup("/{key:.*}", "object", func(r *teapot.Router) {
//	        r.GET("", getObject).Name("get")     // registered as "bucket.object.get"
//	    })
//	})
//
// # S3 Action Context
//
// Tag routes with an S3 action string that is injected into the request context
// and retrievable inside handlers:
//
//	r.GET("/{bucket}/{key:.*}", getObject).
//	    Name("object.get").
//	    Action("s3:GetObject")
//
//	func getObject(w http.ResponseWriter, r *http.Request) {
//	    action := teapot.GetAction(r)    // "s3:GetObject"
//	    name   := teapot.GetRouteName(r) // "object.get"
//	    bucket := teapot.URLParam(r, "bucket")
//	    key    := teapot.URLParam(r, "key")
//	}
//
// For complete documentation see https://github.com/mallardduck/teapot-router
package teapot
