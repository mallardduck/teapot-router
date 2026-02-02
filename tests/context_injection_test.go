package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestContextInjection_DirectRoutes tests context injection for direct (non-query) routes
func TestContextInjection_DirectRoutes(t *testing.T) {
	r := teapot.New()

	// Handler that captures context values
	handler := func(expectedName, expectedAction string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			name := teapot.GetRouteName(r)
			action := teapot.GetAction(r)

			assert.Equal(t, expectedName, name, "route name should be injected")
			assert.Equal(t, expectedAction, action, "action should be injected")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}
	}

	// Direct route with both Name and Action
	r.GET("/users/{id}", handler("users.show", "read:user")).
		Name("users.show").
		Action("read:user")

	// Direct route with only Name
	r.GET("/posts/{id}", handler("posts.show", "")).
		Name("posts.show")

	// Direct route with only Action
	r.GET("/comments/{id}", handler("", "read:comment")).
		Action("read:comment")

	// Direct route with neither
	r.GET("/health", handler("", ""))

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"with both", "GET", "/users/123"},
		{"with name only", "GET", "/posts/456"},
		{"with action only", "GET", "/comments/789"},
		{"with neither", "GET", "/health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "OK", w.Body.String())
		})
	}
}

// TestContextInjection_QueryRoutes tests context injection for query-multiplexed routes
func TestContextInjection_QueryRoutes(t *testing.T) {
	r := teapot.New()

	handler := func(expectedName, expectedAction string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			name := teapot.GetRouteName(r)
			action := teapot.GetAction(r)

			assert.Equal(t, expectedName, name, "route name should be injected")
			assert.Equal(t, expectedAction, action, "action should be injected")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}
	}

	// Query routes with Name and Action
	r.QueryGET("/{bucket}", handler("bucket.list", "s3:ListBucket")).
		Name("bucket.list").
		Action("s3:ListBucket")

	r.QueryGET("/{bucket}", handler("bucket.acl", "s3:GetBucketAcl")).
		Query("acl").
		Name("bucket.acl").
		Action("s3:GetBucketAcl")

	r.QueryGET("/{bucket}", handler("bucket.versioning", "s3:GetBucketVersioning")).
		Query("versioning").
		Name("bucket.versioning").
		Action("s3:GetBucketVersioning")

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"fallback route", "GET", "/mybucket"},
		{"acl route", "GET", "/mybucket?acl"},
		{"versioning route", "GET", "/mybucket?versioning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "OK", w.Body.String())
		})
	}
}

// TestContextInjection_WithFinalize tests that context injection works after finalization
func TestContextInjection_WithFinalize(t *testing.T) {
	r := teapot.New()

	handler := func(expectedName, expectedAction string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			name := teapot.GetRouteName(r)
			action := teapot.GetAction(r)

			assert.Equal(t, expectedName, name, "route name should be injected after finalize")
			assert.Equal(t, expectedAction, action, "action should be injected after finalize")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}
	}

	// Mix of direct and query routes
	r.GET("/direct", handler("direct.route", "test:Direct")).
		Name("direct.route").
		Action("test:Direct")

	r.QueryGET("/query", handler("query.route", "test:Query")).
		Query("test").
		Name("query.route").
		Action("test:Query")

	// Finalize to enable optimizations
	r.Finalize()

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"direct after finalize", "GET", "/direct"},
		{"query after finalize", "GET", "/query?test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "OK", w.Body.String())
		})
	}
}

// TestContextInjection_WithMiddleware tests that context is available in middleware
func TestContextInjection_WithMiddleware(t *testing.T) {
	r := teapot.New()

	// Track middleware execution
	middlewareCalled := false

	// Middleware that checks context
	checkContextMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true

			// Context should be injected before middleware runs
			name := teapot.GetRouteName(r)
			action := teapot.GetAction(r)

			assert.Equal(t, "protected.resource", name, "context should be available in middleware")
			assert.Equal(t, "read:protected", action, "context should be available in middleware")

			next.ServeHTTP(w, r)
		})
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}

	r.GET("/protected", handler).
		Name("protected.resource").
		Action("read:protected").
		With(checkContextMiddleware)

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.True(t, middlewareCalled, "middleware should have been called")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

// TestContextInjection_NamedGroups tests context injection in named groups
func TestContextInjection_NamedGroups(t *testing.T) {
	r := teapot.New()

	handler := func(expectedName, expectedAction string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			name := teapot.GetRouteName(r)
			action := teapot.GetAction(r)

			assert.Equal(t, expectedName, name, "route name should include group prefix")
			assert.Equal(t, expectedAction, action, "action should be injected")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}
	}

	// Named group with prefixed names
	r.NamedGroup("/api/v1", "api.v1", func(r *teapot.Router) {
		r.GET("/users", handler("api.v1.users.index", "list:users")).
			Name("users.index").
			Action("list:users")

		r.GET("/users/{id}", handler("api.v1.users.show", "read:user")).
			Name("users.show").
			Action("read:user")
	})

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"group index", "GET", "/api/v1/users"},
		{"group show", "GET", "/api/v1/users/123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "OK", w.Body.String())
		})
	}
}

// TestContextInjection_WildcardParams tests context with wildcard parameters
func TestContextInjection_WildcardParams(t *testing.T) {
	r := teapot.New()

	handler := func(w http.ResponseWriter, r *http.Request) {
		name := teapot.GetRouteName(r)
		action := teapot.GetAction(r)

		// Also verify wildcard param is accessible
		key := teapot.URLParam(r, "key")

		assert.Equal(t, "object.get", name)
		assert.Equal(t, "s3:GetObject", action)
		assert.NotEmpty(t, key, "wildcard param should be accessible")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}

	r.GET("/{bucket}/{key:.*}", handler).
		Name("object.get").
		Action("s3:GetObject")

	req := httptest.NewRequest("GET", "/mybucket/path/to/file.txt", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

// TestContextInjection_AutoPromotion tests context injection with auto-promoted routes
func TestContextInjection_AutoPromotion(t *testing.T) {
	r := teapot.New()

	handler := func(expectedName, expectedAction string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			name := teapot.GetRouteName(r)
			action := teapot.GetAction(r)

			assert.Equal(t, expectedName, name, "context should work after auto-promotion")
			assert.Equal(t, expectedAction, action, "context should work after auto-promotion")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}
	}

	// Direct route first
	r.GET("/{bucket}", handler("bucket.head", "s3:HeadBucket")).
		Name("bucket.head").
		Action("s3:HeadBucket")

	// Query route on same path (triggers auto-promotion)
	r.QueryGET("/{bucket}", handler("bucket.acl", "s3:GetBucketAcl")).
		Query("acl").
		Name("bucket.acl").
		Action("s3:GetBucketAcl")

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"direct route after promotion", "GET", "/mybucket"},
		{"query route after promotion", "GET", "/mybucket?acl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "OK", w.Body.String())
		})
	}
}

// TestContextInjection_MixedScenarios tests complex real-world scenarios
func TestContextInjection_MixedScenarios(t *testing.T) {
	r := teapot.New()

	// Collector to track all context values seen
	type contextCapture struct {
		path   string
		name   string
		action string
	}
	var captures []contextCapture

	handler := func(path string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			captures = append(captures, contextCapture{
				path:   path,
				name:   teapot.GetRouteName(r),
				action: teapot.GetAction(r),
			})
			w.WriteHeader(http.StatusOK)
		}
	}

	// Build a complex router with multiple types of routes
	r.GET("/", handler("/")).
		Name("root").
		Action("index")

	r.NamedGroup("/api", "api", func(r *teapot.Router) {
		r.GET("/status", handler("/api/status")).
			Name("status").
			Action("health:check")

		r.Group("/v1", func(r *teapot.Router) {
			r.GET("/users", handler("/api/v1/users")).
				Name("users.list").
				Action("list:users")

			r.QueryGET("/users", handler("/api/v1/users?search")).
				Query("search").
				Name("users.search").
				Action("search:users")
		})
	})

	r.GET("/{bucket}/{key:.*}", handler("/{bucket}/{key:.*}")).
		Name("object.get").
		Action("s3:GetObject")

	r.Finalize()

	tests := []struct {
		method         string
		path           string
		expectedName   string
		expectedAction string
	}{
		{"GET", "/", "root", "index"},
		{"GET", "/api/status", "api.status", "health:check"},
		{"GET", "/api/v1/users", "api.users.list", "list:users"},
		{"GET", "/api/v1/users?search=test", "api.users.search", "search:users"},
		{"GET", "/mybucket/path/to/file.txt", "object.get", "s3:GetObject"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			captures = nil // Reset

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			require := assert.New(t)
			require.Len(captures, 1, "handler should have been called once")

			if len(captures) > 0 {
				capture := captures[0]
				assert.Equal(t, tt.expectedName, capture.name, "name mismatch for %s", tt.path)
				assert.Equal(t, tt.expectedAction, capture.action, "action mismatch for %s", tt.path)
			}
		})
	}
}

// TestContextInjection_EmptyValues tests that empty strings work correctly
func TestContextInjection_EmptyValues(t *testing.T) {
	r := teapot.New()

	handler := func(w http.ResponseWriter, r *http.Request) {
		name := teapot.GetRouteName(r)
		action := teapot.GetAction(r)

		// Both should be empty strings (not nil)
		assert.Equal(t, "", name)
		assert.Equal(t, "", action)

		w.WriteHeader(http.StatusOK)
	}

	r.GET("/no-metadata", handler)

	req := httptest.NewRequest("GET", "/no-metadata", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestContextInjection_GlobalVsRouteMiddleware demonstrates the difference between
// global middleware (r.Use before Route()), route-level middleware (r.Use inside Route()),
// and route-specific middleware (.With)
func TestContextInjection_GlobalVsRouteMiddleware(t *testing.T) {
	r := teapot.New()

	// Track what context values each middleware sees
	var globalName, globalAction string
	var routeLevelName, routeLevelAction string
	var routeSpecificName, routeSpecificAction string

	// Global middleware (runs BEFORE Route() group - no RouteContextMiddleware)
	globalMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			globalName = teapot.GetRouteName(r)
			globalAction = teapot.GetAction(r)
			next.ServeHTTP(w, r)
		})
	}

	// Route-level middleware (runs INSIDE Route() group - after RouteContextMiddleware)
	routeLevelMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			routeLevelName = teapot.GetRouteName(r)
			routeLevelAction = teapot.GetAction(r)
			next.ServeHTTP(w, r)
		})
	}

	// Route-specific middleware (runs with .With() - after optimizedHandler context injection)
	routeSpecificMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			routeSpecificName = teapot.GetRouteName(r)
			routeSpecificAction = teapot.GetAction(r)
			next.ServeHTTP(w, r)
		})
	}

	// Add global middleware (before Route() group - no route context available)
	r.Use(globalMiddleware)

	handler := func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Create Route() group with RouteContextMiddleware for route metadata
	r.Route("/", func(r *teapot.Router) {
		// Add RouteContextMiddleware to inject route metadata early
		r.Use(teapot.RouteContextMiddleware(r))
		r.Use(routeLevelMiddleware)

		r.GET("/test", handler).
			Name("test.route").
			Action("test:Action").
			With(routeSpecificMiddleware)
	})

	// Execute request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Verify results
	assert.Equal(t, http.StatusOK, w.Code)

	// Global middleware runs BEFORE Route() group - no RouteContextMiddleware applied yet
	assert.Equal(t, "", globalName, "Global middleware should NOT see route name (no RouteContextMiddleware)")
	assert.Equal(t, "", globalAction, "Global middleware should NOT see action (no RouteContextMiddleware)")

	// Route-level middleware runs INSIDE Route() group - after RouteContextMiddleware
	assert.Equal(t, "test.route", routeLevelName, "Route-level middleware SHOULD see route name")
	assert.Equal(t, "test:Action", routeLevelAction, "Route-level middleware SHOULD see action")

	// Route-specific middleware runs after optimizedHandler injects context
	assert.Equal(t, "test.route", routeSpecificName, "Route-specific middleware SHOULD see route name")
	assert.Equal(t, "test:Action", routeSpecificAction, "Route-specific middleware SHOULD see action")
}

// TestContextInjection_RouteContextMiddleware tests the RouteContextMiddleware function
// when used inside a Route() group (recommended pattern)
func TestContextInjection_RouteContextMiddleware(t *testing.T) {
	r := teapot.New()

	// Track what context values global middleware sees
	var captures []struct {
		path   string
		name   string
		action string
	}

	// Global middleware that captures context
	auditMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			captures = append(captures, struct {
				path   string
				name   string
				action string
			}{
				path:   req.URL.Path,
				name:   teapot.GetRouteName(req),
				action: teapot.GetAction(req),
			})
			next.ServeHTTP(w, req)
		})
	}

	// Register various routes
	handler := func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// IMPORTANT: Use Route() to ensure RouteContext is available (recommended pattern)
	r.Use(chiMiddleware.StripSlashes) // Truly global middleware
	r.Route("/", func(r *teapot.Router) {
		// Add RouteContextMiddleware FIRST, then other middleware
		r.Use(teapot.RouteContextMiddleware(r))
		r.Use(auditMiddleware)

		// Direct routes
		r.GET("/users", handler).
			Name("users.index").
			Action("list:users")

		r.GET("/users/{id}", handler).
			Name("users.show").
			Action("read:user")

		// Query-multiplexed routes
		r.QueryGET("/{bucket}", handler).
			Name("bucket.list").
			Action("s3:ListBucket")

		r.QueryGET("/{bucket}", handler).
			Query("acl").
			Name("bucket.acl").
			Action("s3:GetBucketAcl")
	})

	if !r.IsFinalized() {
		r.Finalize()
	}

	tests := []struct {
		method         string
		path           string
		expectedName   string
		expectedAction string
	}{
		{"GET", "/users", "users.index", "list:users"},
		{"GET", "/users/123/", "users.show", "read:user"},
		{"GET", "/mybucket", "bucket.list", "s3:ListBucket"},
		// For query-multiplexed routes, middleware sees the fallback route
		// The Dispatcher will inject the correct route later in the handler
		{"GET", "/mybucket?acl", "bucket.list", "s3:ListBucket"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			captures = nil // Reset

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			require := assert.New(t)
			require.Len(captures, 1, "middleware should have been called once")

			if len(captures) > 0 {
				capture := captures[0]
				assert.Equal(t, tt.expectedName, capture.name,
					"global middleware should see route name for %s", tt.path)
				assert.Equal(t, tt.expectedAction, capture.action,
					"global middleware should see action for %s", tt.path)
			}
		})
	}
}

// TestContextInjection_RouteContextMiddleware_QueryFallback tests that query routes
// with the RouteContextMiddleware use fallback route when multiple query routes exist
func TestContextInjection_RouteContextMiddleware_QueryFallback(t *testing.T) {
	r := teapot.New()

	var capturedName, capturedAction string

	auditMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			capturedName = teapot.GetRouteName(req)
			capturedAction = teapot.GetAction(req)
			next.ServeHTTP(w, req)
		})
	}

	handler := func(w http.ResponseWriter, req *http.Request) {
		// Also verify context is correct in handler
		handlerName := teapot.GetRouteName(req)
		handlerAction := teapot.GetAction(req)
		_, _ = w.Write([]byte(handlerName + ":" + handlerAction))
	}

	r.Route("/", func(r *teapot.Router) {
		r.Use(teapot.RouteContextMiddleware(r))
		r.Use(auditMiddleware)

		// Fallback route (no query params)
		r.GET("/{bucket}", handler).
			Name("bucket.head").
			Action("s3:HeadBucket")

		// Query-specific routes (will trigger auto-promotion)
		r.QueryGET("/{bucket}", handler).
			Query("acl").
			Name("bucket.acl").
			Action("s3:GetBucketAcl")

		r.QueryGET("/{bucket}", handler).
			Query("versioning").
			Name("bucket.versioning").
			Action("s3:GetBucketVersioning")
	})

	// Test fallback route
	t.Run("fallback route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mybucket", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Global middleware should see fallback
		assert.Equal(t, "bucket.head", capturedName)
		assert.Equal(t, "s3:HeadBucket", capturedAction)
		// Handler should also see correct context
		assert.Equal(t, "bucket.head:s3:HeadBucket", w.Body.String())
	})

	// Test query-specific route
	t.Run("query-specific route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mybucket?acl", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Global middleware sees fallback (can't match query yet)
		assert.Equal(t, "bucket.head", capturedName)
		assert.Equal(t, "s3:HeadBucket", capturedAction)
		// Handler sees correct query-matched route
		assert.Equal(t, "bucket.acl:s3:GetBucketAcl", w.Body.String())
	})
}
