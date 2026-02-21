package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/internal/testutil"
	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// Helper to make test requests
func request(t *testing.T, router http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	assert.Equal(t, want, got)
}

// Test: Basic HTTP methods
func TestBasicHTTPMethods(t *testing.T) {
	r := teapot.New()

	r.Func().GET("/get", testutil.StringResponseWriterBuilder("GET"))
	r.Func().POST("/post", testutil.StringResponseWriterBuilder("POST"))
	r.Func().PUT("/put", testutil.StringResponseWriterBuilder("PUT"))
	r.Func().DELETE("/delete", testutil.StringResponseWriterBuilder("DELETE"))
	r.Func().HEAD("/head", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	assert.Equal(t, "GET", request(t, r, "GET", "/get").Body.String())
	assert.Equal(t, "POST", request(t, r, "POST", "/post").Body.String())
	assert.Equal(t, "PUT", request(t, r, "PUT", "/put").Body.String())
	assert.Equal(t, "DELETE", request(t, r, "DELETE", "/delete").Body.String())
	assert.Equal(t, 200, request(t, r, "HEAD", "/head").Code)
}

// Test: URL parameters
func TestURLParameters(t *testing.T) {
	r := teapot.New()

	r.Func().GET("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := teapot.URLParam(r, "id")
		_, _ = w.Write([]byte("User: " + id))
	})

	w := request(t, r, "GET", "/users/123")
	assertEqual(t, w.Body.String(), "User: 123")
}

// Test: Wildcard parameters
func TestWildcardParameters(t *testing.T) {
	r := teapot.New()

	r.Func().GET("/files/{path:.*}", func(w http.ResponseWriter, r *http.Request) {
		path := teapot.URLParam(r, "path")
		_, _ = w.Write([]byte("Path: " + path))
	})

	w := request(t, r, "GET", "/files/documents/report.pdf")
	assertEqual(t, w.Body.String(), "Path: documents/report.pdf")
}

// Test: Named routes and URL generation
func TestNamedRoutes(t *testing.T) {
	r := teapot.New()

	r.Func().GET("/users/{id}", testutil.NoopResponse).Name("users.show")

	r.Func().GET("/posts/{postId}/comments/{commentId}", testutil.NoopResponse).
		Name("posts.comments.show")

	// Test URL generation
	url := r.MustURL("users.show", "id", "42")
	assertEqual(t, url, "/users/42")

	url = r.MustURL("posts.comments.show", "postId", "10", "commentId", "99")
	assertEqual(t, url, "/posts/10/comments/99")

	// Test URL() with error handling
	_, err := r.URL("nonexistent")
	assert.Error(t, err, "expected error for nonexistent route")
}

// Test: S3 Action and Route Name context injection
func TestActionAndNameContext(t *testing.T) {
	r := teapot.New()

	var capturedAction, capturedName string
	r.Func().GET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		capturedAction = teapot.GetAction(r)
		capturedName = teapot.GetRouteName(r)
		_, _ = w.Write([]byte("OK"))
	}).Name("bucket.list").Action("s3:ListBucket")

	request(t, r, "GET", "/bucket")
	assertEqual(t, capturedAction, "s3:ListBucket")
	assertEqual(t, capturedName, "bucket.list")
}

// Test: Query parameter existence matching
func TestQueryParameterMatching(t *testing.T) {
	r := teapot.New()

	r.Func().QueryGET("/bucket", testutil.StringResponseWriterBuilder("LIST")).Name("bucket.list")

	r.Func().QueryGET("/bucket", testutil.StringResponseWriterBuilder("ACL")).Name("bucket.acl").Query("acl")

	r.Func().QueryGET("/bucket", testutil.StringResponseWriterBuilder("VERSIONING")).Name("bucket.versioning").Query("versioning")

	// No query params = first route (base route)
	assertEqual(t, request(t, r, "GET", "/bucket").Body.String(), "LIST")

	// ?acl = second route
	assertEqual(t, request(t, r, "GET", "/bucket?acl").Body.String(), "ACL")

	// ?versioning = third route
	assertEqual(t, request(t, r, "GET", "/bucket?versioning").Body.String(), "VERSIONING")
}

// Test: Query parameter value matching
func TestQueryValueMatching(t *testing.T) {
	r := teapot.New()

	r.Func().QueryGET("/search", testutil.StringResponseWriterBuilder("FULL")).QueryValue("type", "full")

	r.Func().QueryGET("/search", testutil.StringResponseWriterBuilder("PARTIAL")).QueryValue("type", "partial")

	r.Func().QueryGET("/search", testutil.StringResponseWriterBuilder("DEFAULT"))

	assertEqual(t, request(t, r, "GET", "/search?type=full").Body.String(), "FULL")
	assertEqual(t, request(t, r, "GET", "/search?type=partial").Body.String(), "PARTIAL")
	assertEqual(t, request(t, r, "GET", "/search").Body.String(), "DEFAULT")
	assertEqual(t, request(t, r, "GET", "/search?type=other").Body.String(), "DEFAULT")
}

// Test: Multiple query parameter matching (priority)
func TestMultipleQueryParameterPriority(t *testing.T) {
	r := teapot.New()

	// More specific (2 params) should match first
	r.Func().QueryGET("/object", testutil.StringResponseWriterBuilder("TWO_PARAMS")).Query("partNumber").Query("uploadId")

	// Less specific (1 param)
	r.Func().QueryGET("/object", testutil.StringResponseWriterBuilder("UPLOAD_ID")).Query("uploadId")

	// No query params
	r.Func().QueryGET("/object", testutil.StringResponseWriterBuilder("BASE"))

	// Two params = most specific route
	assertEqual(t, request(t, r, "GET", "/object?uploadId=123&partNumber=1").Body.String(), "TWO_PARAMS")

	// One param = less specific route
	assertEqual(t, request(t, r, "GET", "/object?uploadId=123").Body.String(), "UPLOAD_ID")

	// No params = base route
	assertEqual(t, request(t, r, "GET", "/object").Body.String(), "BASE")
}

// Test: Route groups with path prefix
func TestRouteGroups(t *testing.T) {
	r := teapot.New()

	r.Group("/api", func(r *teapot.Router) {
		r.Func().GET("/users", testutil.StringResponseWriterBuilder("USERS"))
		r.Func().GET("/posts", testutil.StringResponseWriterBuilder("POSTS"))
	})

	assertEqual(t, request(t, r, "GET", "/api/users").Body.String(), "USERS")
	assertEqual(t, request(t, r, "GET", "/api/posts").Body.String(), "POSTS")
}

// Test: Named groups (path + name prefix)
func TestNamedGroups(t *testing.T) {
	r := teapot.New()

	r.NamedGroup("/{bucket}", "bucket", func(r *teapot.Router) {
		r.Func().QueryGET("", testutil.StringResponseWriterBuilder("LIST")).Name("list")

		r.Func().QueryGET("", testutil.StringResponseWriterBuilder("ACL")).Name("acl").Query("acl")

		r.NamedGroup("/{key:.*}", "object", func(r *teapot.Router) {
			r.Func().GET("", testutil.StringResponseWriterBuilder("GET_OBJECT")).Name("get")
		})
	})

	// Test routes work
	assertEqual(t, request(t, r, "GET", "/mybucket").Body.String(), "LIST")
	assertEqual(t, request(t, r, "GET", "/mybucket?acl").Body.String(), "ACL")
	assertEqual(t, request(t, r, "GET", "/mybucket/path/to/file.txt").Body.String(), "GET_OBJECT")

	// Test name composition
	url := r.MustURL("bucket.list", "bucket", "test")
	assertEqual(t, url, "/test")

	url = r.MustURL("bucket.object.get", "bucket", "test", "key", "file.txt")
	assertEqual(t, url, "/test/file.txt")
}

// Test: Route-specific middleware
func TestRouteMiddleware(t *testing.T) {
	r := teapot.New()

	middlewareCalled := false
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}

	r.Func().GET("/protected", testutil.StringResponseWriterBuilder("OK")).With(mw)

	r.Func().GET("/public", testutil.StringResponseWriterBuilder("OK"))

	// Protected route should trigger middleware
	middlewareCalled = false
	request(t, r, "GET", "/protected")
	assert.True(t, middlewareCalled, "middleware was not called for protected route")

	// Public route should not trigger middleware
	middlewareCalled = false
	request(t, r, "GET", "/public")
	assert.False(t, middlewareCalled, "middleware was called for public route")
}

// Test: Global middleware via Use()
func TestGlobalMiddleware(t *testing.T) {
	r := teapot.New()

	callCount := 0
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			next.ServeHTTP(w, r)
		})
	})

	r.Func().GET("/a", testutil.NoopResponse)
	r.Func().GET("/b", testutil.NoopResponse)

	request(t, r, "GET", "/a")
	request(t, r, "GET", "/b")

	assertEqual(t, callCount, 2)
}

// Test: S3 bucket operations scenario
func TestS3BucketOperations(t *testing.T) {
	r := teapot.New()

	r.NamedGroup("/{bucket}", "bucket", func(r *teapot.Router) {
		r.Func().PUT("", testutil.StringResponseWriterBuilder("CREATE")).Name("create").Action("s3:CreateBucket")

		r.Func().DELETE("", testutil.StringResponseWriterBuilder("DELETE")).Name("delete").Action("s3:DeleteBucket")

		r.Func().QueryGET("", testutil.StringResponseWriterBuilder("LIST")).Name("list").Action("s3:ListBucket")

		r.Func().QueryGET("", testutil.StringResponseWriterBuilder("ACL")).Name("acl.get").Action("s3:GetBucketAcl").Query("acl")
	})

	assertEqual(t, request(t, r, "PUT", "/mybucket").Body.String(), "CREATE")
	assertEqual(t, request(t, r, "DELETE", "/mybucket").Body.String(), "DELETE")
	assertEqual(t, request(t, r, "GET", "/mybucket").Body.String(), "LIST")
	assertEqual(t, request(t, r, "GET", "/mybucket?acl").Body.String(), "ACL")
}

// Test: S3 object operations scenario
func TestS3ObjectOperations(t *testing.T) {
	r := teapot.New()

	r.NamedGroup("/{bucket}/{key:.*}", "object", func(r *teapot.Router) {
		var capturedBucket, capturedKey string

		r.Func().QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			capturedBucket = teapot.URLParam(r, "bucket")
			capturedKey = teapot.URLParam(r, "key")
			_, _ = w.Write([]byte("GET"))
		}).Name("get").Action("s3:GetObject")

		r.Func().PUT("", testutil.StringResponseWriterBuilder("PUT")).Name("put").Action("s3:PutObject")

		r.Func().QueryGET("", testutil.StringResponseWriterBuilder("ACL")).Name("acl").Action("s3:GetObjectAcl").Query("acl")

		request(t, r, "GET", "/mybucket/path/to/file.txt")
		assertEqual(t, capturedBucket, "mybucket")
		assertEqual(t, capturedKey, "path/to/file.txt")
	})

	assertEqual(t, request(t, r, "GET", "/mybucket/file.txt").Body.String(), "GET")
	assertEqual(t, request(t, r, "PUT", "/mybucket/file.txt").Body.String(), "PUT")
	assertEqual(t, request(t, r, "GET", "/mybucket/file.txt?acl").Body.String(), "ACL")
}

// Test: Route introspection
func TestRouteIntrospection(t *testing.T) {
	r := teapot.New()

	r.Func().GET("/users", testutil.NoopResponse).
		Name("users.list").
		Action("app:ListUsers")

	r.Func().POST("/users", testutil.NoopResponse).
		Name("users.create").
		Action("app:CreateUser")

	routes := r.Routes()
	assert.Len(t, routes, 2)

	// Check first route
	assert.Equal(t, "GET", routes[0].Method)
	assert.Equal(t, "/users", routes[0].Pattern)
	assert.Equal(t, "users.list", routes[0].Name)
	assert.Equal(t, "app:ListUsers", routes[0].Action)
}

// Test: Edge case - empty route name
func TestEmptyRouteName(t *testing.T) {
	r := teapot.New()

	// Should work without a name
	r.Func().GET("/test", testutil.StringResponseWriterBuilder("OK"))

	assertEqual(t, request(t, r, "GET", "/test").Body.String(), "OK")
}

// Test: Edge case - trailing slashes
func TestTrailingSlashes(t *testing.T) {
	r := teapot.New()

	r.Func().GET("/path", testutil.StringResponseWriterBuilder("OK"))

	// Chi handles trailing slash redirects by default
	// Just verify the route works
	w := request(t, r, "GET", "/path")
	assertEqual(t, w.Body.String(), "OK")
}

// Test: Same name different methods (Laravel-style resources)
func TestSameNameDifferentMethods(t *testing.T) {
	r := teapot.New()

	r.Func().GET("/users/{id}", testutil.StringResponseWriterBuilder("SHOW")).Name("users.show")

	r.Func().PUT("/users/{id}", testutil.StringResponseWriterBuilder("UPDATE")).Name("users.update")

	r.Func().DELETE("/users/{id}", testutil.StringResponseWriterBuilder("DELETE")).Name("users.destroy")

	// All should generate the same URL pattern
	url := r.MustURL("users.show", "id", "42")
	assertEqual(t, url, "/users/42")

	url = r.MustURL("users.update", "id", "42")
	assertEqual(t, url, "/users/42")

	// But routes should work correctly
	assertEqual(t, request(t, r, "GET", "/users/42").Body.String(), "SHOW")
	assertEqual(t, request(t, r, "PUT", "/users/42").Body.String(), "UPDATE")
	assertEqual(t, request(t, r, "DELETE", "/users/42").Body.String(), "DELETE")
}

// Test: NotFound handler
func TestNotFoundHandler(t *testing.T) {
	r := teapot.New()

	r.Func().GET("/exists", testutil.StringResponseWriterBuilder("OK"))

	// Non-existent route should return 404
	w := request(t, r, "GET", "/nonexistent")
	assertEqual(t, w.Code, 404)
}

// Test: Handler function access to context values
func TestContextAccess(t *testing.T) {
	r := teapot.New()

	var values []string
	r.Func().GET("/{bucket}/{key:.*}", func(w http.ResponseWriter, r *http.Request) {
		values = append(values, teapot.GetAction(r))
		values = append(values, teapot.GetRouteName(r))
		values = append(values, teapot.URLParam(r, "bucket"))
		values = append(values, teapot.URLParam(r, "key"))
		_, _ = w.Write([]byte("OK"))
	}).Name("object.get").Action("s3:GetObject")

	request(t, r, "GET", "/mybucket/path/to/file.txt")

	expected := []string{"s3:GetObject", "object.get", "mybucket", "path/to/file.txt"}
	assert.Equal(t, expected, values)
}

func TestMiddlewareGroupRoutes(t *testing.T) {
	asserts := assert.New(t)
	r := teapot.New()

	// Register some routes directly
	r.GET("/direct1", nil).Name("direct1")
	r.GET("/direct2", nil).Name("direct2")

	// Register routes in a middleware group
	r.MiddlewareGroup(func(r *teapot.Router) {
		r.GET("/group1", nil).Name("group1")
		r.GET("/group2", nil).Name("group2")
		r.GET("/group3", nil).Name("group3")
		r.GET("/group4", nil).Name("group4")
		r.GET("/group5", nil).Name("group5")
	})

	routes := r.Routes()
	asserts.Len(routes, 7, "expected 7 routes after middleware group registration")

	// Verify route details
	routeNames := make(map[string]bool)
	for _, route := range routes {
		routeNames[route.Name] = true
		asserts.Equal("GET", route.Method, "all routes should be GET")
	}

	// Check all expected routes are present
	expectedNames := []string{"direct1", "direct2", "group1", "group2", "group3", "group4", "group5"}
	for _, name := range expectedNames {
		asserts.True(routeNames[name], "route %s should be registered", name)
	}
}

// Test: MiddlewareGroup actually applies middleware
func TestMiddlewareGroupAppliesMiddleware(t *testing.T) {
	r := teapot.New()

	// Track which routes the middleware was called for
	var callLog []string

	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callLog = append(callLog, r.URL.Path)
			w.Header().Set("X-Auth", "checked")
			next.ServeHTTP(w, r)
		})
	}

	// Routes without middleware
	r.Func().GET("/public", testutil.StringResponseWriterBuilder("public"))

	// Routes with middleware group
	r.MiddlewareGroup(func(r *teapot.Router) {
		r.Func().GET("/admin", testutil.StringResponseWriterBuilder("admin"))
		r.Func().GET("/dashboard", testutil.StringResponseWriterBuilder("dashboard"))
	}, authMiddleware)

	// Test public route - middleware should NOT be called
	callLog = []string{}
	w := request(t, r, "GET", "/public")
	assertEqual(t, w.Body.String(), "public")
	assertEqual(t, len(callLog), 0)
	assertEqual(t, w.Header().Get("X-Auth"), "")

	// Test admin route - middleware SHOULD be called
	callLog = []string{}
	w = request(t, r, "GET", "/admin")
	assertEqual(t, w.Body.String(), "admin")
	assertEqual(t, len(callLog), 1)
	assertEqual(t, callLog[0], "/admin")
	assertEqual(t, w.Header().Get("X-Auth"), "checked")

	// Test dashboard route - middleware SHOULD be called
	callLog = []string{}
	w = request(t, r, "GET", "/dashboard")
	assertEqual(t, w.Body.String(), "dashboard")
	assertEqual(t, len(callLog), 1)
	assertEqual(t, callLog[0], "/dashboard")
	assertEqual(t, w.Header().Get("X-Auth"), "checked")
}

// Test: Multiple middleware in a group
func TestMiddlewareGroupMultipleMiddlewares(t *testing.T) {
	r := teapot.New()

	var executionOrder []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "mw1-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "mw1-after")
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "mw2-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "mw2-after")
		})
	}

	mw3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "mw3-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "mw3-after")
		})
	}

	r.MiddlewareGroup(func(r *teapot.Router) {
		r.Func().GET("/test", func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "handler")
			_, _ = w.Write([]byte("ok"))
		})
	}, mw1, mw2, mw3)

	executionOrder = []string{}
	request(t, r, "GET", "/test")

	// Middleware should execute in order: mw1 -> mw2 -> mw3 -> handler -> mw3 -> mw2 -> mw1
	expected := []string{
		"mw1-before", "mw2-before", "mw3-before",
		"handler",
		"mw3-after", "mw2-after", "mw1-after",
	}

	assert.Equal(t, expected, executionOrder)
}

// Test: Nested middleware groups - execution order
func TestMiddlewareGroupNestedExecution(t *testing.T) {
	r := teapot.New()

	var callLog []string

	outerMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callLog = append(callLog, "outer")
			next.ServeHTTP(w, r)
		})
	}

	innerMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callLog = append(callLog, "inner")
			next.ServeHTTP(w, r)
		})
	}

	r.MiddlewareGroup(func(r *teapot.Router) {
		r.Func().GET("/outer-only", testutil.StringResponseWriterBuilder("outer-only"))

		// Nested middleware group
		r.MiddlewareGroup(func(r *teapot.Router) {
			r.Func().GET("/both", testutil.StringResponseWriterBuilder("both"))
		}, innerMw)
	}, outerMw)

	// Test outer-only route
	callLog = []string{}
	request(t, r, "GET", "/outer-only")
	assertEqual(t, len(callLog), 1)
	assertEqual(t, callLog[0], "outer")

	// Test nested route (should have both middlewares)
	callLog = []string{}
	request(t, r, "GET", "/both")
	assertEqual(t, len(callLog), 2)
	assertEqual(t, callLog[0], "outer")
	assertEqual(t, callLog[1], "inner")
}

// Test: MiddlewareGroup with different HTTP methods
func TestMiddlewareGroupMixedMethods(t *testing.T) {
	asserts := assert.New(t)
	r := teapot.New()

	var authCalled bool
	authMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCalled = true
			next.ServeHTTP(w, r)
		})
	}

	r.MiddlewareGroup(func(r *teapot.Router) {
		r.Func().GET("/resource", testutil.StringResponseWriterBuilder("GET")).Name("resource.show")

		r.Func().POST("/resource", testutil.StringResponseWriterBuilder("POST")).Name("resource.create")

		r.Func().PUT("/resource", testutil.StringResponseWriterBuilder("PUT")).Name("resource.update")

		r.Func().DELETE("/resource", testutil.StringResponseWriterBuilder("DELETE")).Name("resource.destroy")
	}, authMw)

	routes := r.Routes()
	asserts.Len(routes, 4)

	// Test each method - all should have middleware
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for _, method := range methods {
		authCalled = false
		w := request(t, r, method, "/resource")
		asserts.Equal(method, w.Body.String())
		asserts.True(authCalled, "middleware should be called for %s", method)
	}
}

// Test: MiddlewareGroup with QueryGET
func TestMiddlewareGroupWithQueryRoutes(t *testing.T) {
	asserts := assert.New(t)
	r := teapot.New()

	var mwCalled int
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mwCalled++
			next.ServeHTTP(w, r)
		})
	}

	r.MiddlewareGroup(func(r *teapot.Router) {
		r.Func().QueryGET("/api", testutil.StringResponseWriterBuilder("base")).Name("api.base")

		r.Func().QueryGET("/api", testutil.StringResponseWriterBuilder("filter")).Name("api.filter").Query("filter")

		r.Func().QueryGET("/api", testutil.StringResponseWriterBuilder("sort")).Name("api.sort").Query("sort")
	}, mw)

	routes := r.Routes()
	asserts.Len(routes, 3, "all query routes should be registered")

	// Test base route
	mwCalled = 0
	w := request(t, r, "GET", "/api")
	asserts.Equal("base", w.Body.String())
	asserts.Equal(1, mwCalled)

	// Test filter route
	mwCalled = 0
	w = request(t, r, "GET", "/api?filter")
	asserts.Equal("filter", w.Body.String())
	asserts.Equal(1, mwCalled)

	// Test sort route
	mwCalled = 0
	w = request(t, r, "GET", "/api?sort")
	asserts.Equal("sort", w.Body.String())
	asserts.Equal(1, mwCalled)
}

// Test: MiddlewareGroup with route names and actions
func TestMiddlewareGroupWithNamesAndActions(t *testing.T) {
	asserts := assert.New(t)
	r := teapot.New()

	r.MiddlewareGroup(func(r *teapot.Router) {
		r.Func().GET("/users", testutil.NoopResponse).
			Name("users.index").
			Action("app:ListUsers")

		r.Func().POST("/users", testutil.NoopResponse).
			Name("users.create").
			Action("app:CreateUser")
	}, func(next http.Handler) http.Handler {
		return next
	})

	routes := r.Routes()
	asserts.Len(routes, 2)

	// Verify first route
	asserts.Equal("GET", routes[0].Method)
	asserts.Equal("/users", routes[0].Pattern)
	asserts.Equal("users.index", routes[0].Name)
	asserts.Equal("app:ListUsers", routes[0].Action)

	// Verify second route
	asserts.Equal("POST", routes[1].Method)
	asserts.Equal("/users", routes[1].Pattern)
	asserts.Equal("users.create", routes[1].Name)
	asserts.Equal("app:CreateUser", routes[1].Action)

	// Test URL generation
	url := r.MustURL("users.index")
	asserts.Equal("/users", url)
}

// Test: MiddlewareGroup combined with NamedGroup - full integration
func TestMiddlewareGroupCombinedWithNamedGroup(t *testing.T) {
	asserts := assert.New(t)
	r := teapot.New()

	var authCalled, adminCalled bool

	authMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCalled = true
			next.ServeHTTP(w, r)
		})
	}

	adminMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			adminCalled = true
			next.ServeHTTP(w, r)
		})
	}

	// Middleware group with named group inside
	r.MiddlewareGroup(func(r *teapot.Router) {
		r.NamedGroup("/admin", "admin", func(r *teapot.Router) {
			r.Func().GET("/users", testutil.StringResponseWriterBuilder("admin users")).Name("users")

			r.Func().GET("/settings", testutil.StringResponseWriterBuilder("admin settings")).Name("settings")
		})
	}, authMw, adminMw)

	routes := r.Routes()
	asserts.Len(routes, 2)

	// Verify routes have correct paths and names
	asserts.Equal("/admin/users", routes[0].Pattern)
	asserts.Equal("admin.users", routes[0].Name)
	asserts.Equal("/admin/settings", routes[1].Pattern)
	asserts.Equal("admin.settings", routes[1].Name)

	// Test URL generation with grouped names
	url := r.MustURL("admin.users")
	asserts.Equal("/admin/users", url)

	// Test that both middlewares are applied
	authCalled = false
	adminCalled = false
	w := request(t, r, "GET", "/admin/users")
	asserts.Equal("admin users", w.Body.String())
	asserts.True(authCalled, "auth middleware should be called")
	asserts.True(adminCalled, "admin middleware should be called")
}

// Test: MiddlewareGroup with route-specific middleware
func TestMiddlewareGroupWithRouteSpecificMiddleware(t *testing.T) {
	r := teapot.New()

	var groupMwCalled, routeMwCalled bool

	groupMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			groupMwCalled = true
			next.ServeHTTP(w, r)
		})
	}

	routeMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			routeMwCalled = true
			next.ServeHTTP(w, r)
		})
	}

	r.MiddlewareGroup(func(r *teapot.Router) {
		r.Func().GET("/normal", testutil.StringResponseWriterBuilder("normal"))

		r.Func().GET("/extra", testutil.StringResponseWriterBuilder("extra")).With(routeMw)
	}, groupMw)

	// Test normal route - only group middleware
	groupMwCalled = false
	routeMwCalled = false
	request(t, r, "GET", "/normal")
	assert.True(t, groupMwCalled, "group middleware should be called for /normal")
	assert.False(t, routeMwCalled, "route middleware should NOT be called for /normal")

	// Test extra route - both middlewares
	groupMwCalled = false
	routeMwCalled = false
	request(t, r, "GET", "/extra")
	assert.True(t, groupMwCalled, "group middleware should be called for /extra")
	assert.True(t, routeMwCalled, "route middleware should be called for /extra")
}

// Test: Routes added via With() are visible in Routes()
func TestWithRoutesRegistration(t *testing.T) {
	asserts := assert.New(t)
	r := teapot.New()

	mw := func(next http.Handler) http.Handler {
		return next
	}

	// Register routes via With()
	r.GET("/direct", nil).Name("direct")
	r.With(mw).GET("/with1", nil).Name("with1")
	r.With(mw).GET("/with2", nil).Name("with2")
	r.With(mw).POST("/with3", nil).Name("with3")

	routes := r.Routes()
	asserts.Len(routes, 4, "all routes including With() should be registered")

	// Verify route names
	routeNames := make(map[string]bool)
	for _, route := range routes {
		routeNames[route.Name] = true
	}

	asserts.True(routeNames["direct"])
	asserts.True(routeNames["with1"])
	asserts.True(routeNames["with2"])
	asserts.True(routeNames["with3"])
}

// Test: Routes added via chained With() are visible
func TestChainedWithRoutesRegistration(t *testing.T) {
	asserts := assert.New(t)
	r := teapot.New()

	mw1 := func(next http.Handler) http.Handler { return next }
	mw2 := func(next http.Handler) http.Handler { return next }

	r.GET("/direct", nil).Name("direct")
	r.With(mw1).GET("/with1", nil).Name("with1")
	r.With(mw1).With(mw2).GET("/chained", nil).Name("chained")
	r.With(mw1).With(mw2).POST("/chained2", nil).Name("chained2")

	routes := r.Routes()
	asserts.Len(routes, 4, "all routes including chained With() should be registered")

	// Verify all names are present
	routeNames := make(map[string]bool)
	for _, route := range routes {
		routeNames[route.Name] = true
	}

	asserts.True(routeNames["direct"])
	asserts.True(routeNames["with1"])
	asserts.True(routeNames["chained"])
	asserts.True(routeNames["chained2"])
}

// Test: Routes added via Group() are visible in Routes()
func TestGroupRoutesRegistration(t *testing.T) {
	asserts := assert.New(t)
	r := teapot.New()

	r.GET("/direct", nil).Name("direct")

	r.Group("/api", func(r *teapot.Router) {
		r.GET("/users", nil).Name("users")
		r.GET("/posts", nil).Name("posts")
		r.POST("/comments", nil).Name("comments")
	})

	routes := r.Routes()
	asserts.Len(routes, 4, "all routes including Group() should be registered")

	// Verify paths and names
	routePaths := make(map[string]string)
	for _, route := range routes {
		routePaths[route.Name] = route.Pattern
	}

	asserts.Equal("/direct", routePaths["direct"])
	asserts.Equal("/api/users", routePaths["users"])
	asserts.Equal("/api/posts", routePaths["posts"])
	asserts.Equal("/api/comments", routePaths["comments"])
}

// Test: Routes added in nested groups are visible
func TestNestedGroupRoutesRegistration(t *testing.T) {
	asserts := assert.New(t)
	r := teapot.New()

	r.GET("/root", nil).Name("root")

	r.Group("/api", func(r *teapot.Router) {
		r.GET("/status", nil).Name("status")

		r.Group("/v1", func(r *teapot.Router) {
			r.GET("/users", nil).Name("users")
			r.GET("/posts", nil).Name("posts")

			r.Group("/admin", func(r *teapot.Router) {
				r.GET("/settings", nil).Name("settings")
			})
		})
	})

	routes := r.Routes()
	asserts.Len(routes, 5, "all routes including nested groups should be registered")

	// Verify paths
	routePaths := make(map[string]string)
	for _, route := range routes {
		routePaths[route.Name] = route.Pattern
	}

	asserts.Equal("/root", routePaths["root"])
	asserts.Equal("/api/status", routePaths["status"])
	asserts.Equal("/api/v1/users", routePaths["users"])
	asserts.Equal("/api/v1/posts", routePaths["posts"])
	asserts.Equal("/api/v1/admin/settings", routePaths["settings"])
}

// Test: Routes added via With() inside Group() are visible
func TestWithInsideGroupRoutesRegistration(t *testing.T) {
	asserts := assert.New(t)
	r := teapot.New()

	mw := func(next http.Handler) http.Handler { return next }

	r.Group("/api", func(r *teapot.Router) {
		r.GET("/public", nil).Name("public")
		r.With(mw).GET("/protected", nil).Name("protected")
		r.With(mw).POST("/admin", nil).Name("admin")
	})

	routes := r.Routes()
	asserts.Len(routes, 3, "all routes including With() inside Group() should be registered")

	routeNames := make(map[string]bool)
	for _, route := range routes {
		routeNames[route.Name] = true
	}

	asserts.True(routeNames["public"])
	asserts.True(routeNames["protected"])
	asserts.True(routeNames["admin"])
}

// Test: Complex combination - With(), Group(), NamedGroup(), MiddlewareGroup()
func TestComplexRouteCombinationsRegistration(t *testing.T) {
	asserts := assert.New(t)
	r := teapot.New()

	mw := func(next http.Handler) http.Handler { return next }

	// Direct routes
	r.GET("/home", nil).Name("home")
	r.With(mw).GET("/protected", nil).Name("protected")

	// Regular group
	r.Group("/api", func(r *teapot.Router) {
		r.GET("/status", nil).Name("status")
	})

	// Named group
	r.NamedGroup("/users", "users", func(r *teapot.Router) {
		r.GET("", nil).Name("list")
		r.GET("/{id}", nil).Name("show")
	})

	// Middleware group
	r.MiddlewareGroup(func(r *teapot.Router) {
		r.GET("/admin", nil).Name("admin")
		r.GET("/dashboard", nil).Name("dashboard")
	}, mw)

	// Nested combination
	r.Group("/blog", func(r *teapot.Router) {
		r.With(mw).GET("/posts", nil).Name("posts")

		r.NamedGroup("/authors", "authors", func(r *teapot.Router) {
			r.GET("", nil).Name("list")
			r.GET("/{id}", nil).Name("show")
		})
	})

	routes := r.Routes()
	asserts.Len(routes, 10, "all routes from complex combinations should be registered")

	// Just verify we got the expected count - detailed verification covered in other tests
	routeNames := make(map[string]bool)
	for _, route := range routes {
		routeNames[route.Name] = true
	}

	expectedNames := []string{
		"home", "protected", "status",
		"users.list", "users.show",
		"admin", "dashboard",
		"posts", "authors.list", "authors.show",
	}

	for _, name := range expectedNames {
		asserts.True(routeNames[name], "route %s should be registered", name)
	}
}

// Test: Routes added to sub-router after parent finalization are still registered
func TestSubRouterAfterFinalization(t *testing.T) {
	asserts := assert.New(t)
	r := teapot.New()

	// Add route before finalization
	r.Func().GET("/before", testutil.StringResponseWriterBuilder("before")).Name("before")

	// Create sub-router before finalization
	mw := func(next http.Handler) http.Handler { return next }
	subRouter := r.With(mw)

	// Finalize parent
	r.Finalize()
	asserts.True(r.IsFinalized(), "parent should be finalized")

	// Add routes to sub-router after parent finalization
	subRouter.Func().GET("/after1", testutil.StringResponseWriterBuilder("after1")).Name("after1")

	r.Func().GET("/after2", testutil.StringResponseWriterBuilder("after2")).Name("after2")

	// All routes should be registered and accessible
	routes := r.Routes()
	asserts.Len(routes, 3, "all routes should be registered even after finalization")

	routeNames := make(map[string]bool)
	for _, route := range routes {
		routeNames[route.Name] = true
	}

	asserts.True(routeNames["before"])
	asserts.True(routeNames["after1"], "route added to sub-router after finalization should be registered")
	asserts.True(routeNames["after2"], "route added to parent after finalization should be registered")

	// Test that routes actually work
	w := request(t, r, "GET", "/before")
	asserts.Equal("before", w.Body.String())

	w = request(t, r, "GET", "/after1")
	asserts.Equal("after1", w.Body.String())

	w = request(t, r, "GET", "/after2")
	asserts.Equal("after2", w.Body.String())
}

// Test: MiddlewareGroup routes added after finalization
func TestMiddlewareGroupAfterFinalization(t *testing.T) {
	asserts := assert.New(t)
	r := teapot.New()

	mw := func(next http.Handler) http.Handler { return next }

	r.GET("/before", nil).Name("before")
	r.Finalize()

	// Add middleware group after finalization
	r.MiddlewareGroup(func(r *teapot.Router) {
		r.GET("/after1", nil).Name("after1")
		r.GET("/after2", nil).Name("after2")
	}, mw)

	routes := r.Routes()
	asserts.Len(routes, 3, "routes in middleware group after finalization should be registered")

	routeNames := make(map[string]bool)
	for _, route := range routes {
		routeNames[route.Name] = true
	}

	asserts.True(routeNames["before"])
	asserts.True(routeNames["after1"])
	asserts.True(routeNames["after2"])
}
