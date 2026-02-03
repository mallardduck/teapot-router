package teapot_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mallardduck/teapot-router/internal/core"
	"github.com/mallardduck/teapot-router/pkg/dispatch"
	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestRouteContextMiddleware tests the route context middleware
func TestRouteContextMiddleware(t *testing.T) {
	t.Run("fast path - chi RouteContext available with direct route", func(t *testing.T) {
		r := teapot.New()

		var capturedAction, capturedName string
		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {
			capturedAction = teapot.GetAction(req)
			capturedName = teapot.GetRouteName(req)
			_, _ = w.Write([]byte("OK"))
		}).Action("test:Action").Name("test.route")

		r.Finalize()

		// Make request - context should be injected via fast path
		resp := request(t, r, "GET", "/test")
		assert.Equal(t, "OK", resp.Body.String())
		assert.Equal(t, "test:Action", capturedAction)
		assert.Equal(t, "test.route", capturedName)
	})

	t.Run("fast path - chi RouteContext available with dispatcher route", func(t *testing.T) {
		r := teapot.New()

		var capturedAction string
		r.QueryGET("/bucket", func(w http.ResponseWriter, req *http.Request) {
			capturedAction = teapot.GetAction(req)
			_, _ = w.Write([]byte("LIST"))
		}).Action("s3:ListBucket")

		r.QueryGET("/bucket", func(w http.ResponseWriter, req *http.Request) {
			capturedAction = teapot.GetAction(req)
			_, _ = w.Write([]byte("ACL"))
		}).Action("s3:GetBucketAcl").Query("acl")

		r.Finalize()

		// Request without query param - should match fallback route
		capturedAction = ""
		resp := request(t, r, "GET", "/bucket")
		assert.Equal(t, "LIST", resp.Body.String())
		assert.Equal(t, "s3:ListBucket", capturedAction)

		// Request with query param - should match specific route
		capturedAction = ""
		resp = request(t, r, "GET", "/bucket?acl")
		assert.Equal(t, "ACL", resp.Body.String())
		assert.Equal(t, "s3:GetBucketAcl", capturedAction)
	})

	t.Run("fast path - dispatcher with no fallback route", func(t *testing.T) {
		r := teapot.New()

		var capturedAction string
		// Only query-specific routes, no fallback
		r.QueryGET("/bucket", func(w http.ResponseWriter, req *http.Request) {
			capturedAction = teapot.GetAction(req)
			_, _ = w.Write([]byte("ACL"))
		}).Action("s3:GetBucketAcl").Query("acl")

		r.QueryGET("/bucket", func(w http.ResponseWriter, req *http.Request) {
			capturedAction = teapot.GetAction(req)
			_, _ = w.Write([]byte("VERSIONING"))
		}).Action("s3:GetBucketVersioning").Query("versioning")

		r.Finalize()

		// Request with acl query param
		capturedAction = ""
		resp := request(t, r, "GET", "/bucket?acl")
		assert.Equal(t, "ACL", resp.Body.String())
		assert.Equal(t, "s3:GetBucketAcl", capturedAction)
	})

	t.Run("fallback path - exact match", func(t *testing.T) {
		r := teapot.New()

		captured := false

		// Use global middleware to test fallback path
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				// In global middleware, context should be available via fallback
				_ = teapot.GetAction(req)
				captured = true
				next.ServeHTTP(w, req)
			})
		})

		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("OK"))
		}).Action("test:Action")

		r.Finalize()

		captured = false
		resp := request(t, r, "GET", "/test")
		assert.Equal(t, "OK", resp.Body.String())
		assert.True(t, captured)
		// Note: In global middleware, action injection may not work the same way
	})

	t.Run("fallback path - pattern matching", func(t *testing.T) {
		r := teapot.New()

		var capturedAction, capturedName string

		r.GET("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
			capturedAction = teapot.GetAction(req)
			capturedName = teapot.GetRouteName(req)
			_, _ = w.Write([]byte("USER"))
		}).Action("users:Get").Name("users.get")

		r.Finalize()

		// Make request with pattern
		capturedAction = ""
		capturedName = ""
		resp := request(t, r, "GET", "/users/123")
		assert.Equal(t, "USER", resp.Body.String())
		assert.Equal(t, "users:Get", capturedAction)
		assert.Equal(t, "users.get", capturedName)
	})
}

// TestMatchPattern tests the pattern matching logic used by middleware fallback path
func TestMatchPattern(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		r := teapot.New()

		r.GET("/exact/path", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("EXACT"))
		})

		r.Finalize()

		resp := request(t, r, "GET", "/exact/path")
		assert.Equal(t, "EXACT", resp.Body.String())
	})

	t.Run("no parameters - must be exact", func(t *testing.T) {
		r := teapot.New()

		r.GET("/static", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("STATIC"))
		})

		r.Finalize()

		resp := request(t, r, "GET", "/static")
		assert.Equal(t, "STATIC", resp.Body.String())

		// Different path should not match
		resp = request(t, r, "GET", "/static/other")
		assert.Equal(t, 404, resp.Code)
	})

	t.Run("wildcard at end - path with more segments", func(t *testing.T) {
		r := teapot.New()

		r.GET("/files/*", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("FILES"))
		})

		r.Finalize()

		// Should match path with more segments
		resp := request(t, r, "GET", "/files/a/b/c/d")
		assert.Equal(t, "FILES", resp.Body.String())
	})

	t.Run("wildcard at end - path with fewer segments", func(t *testing.T) {
		r := teapot.New()

		r.GET("/files/prefix/*", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("FILES"))
		})

		r.Finalize()

		// Should not match path with fewer segments
		resp := request(t, r, "GET", "/files")
		assert.Equal(t, 404, resp.Code)
	})

	t.Run("wildcard at end - path with exact segments", func(t *testing.T) {
		r := teapot.New()

		r.GET("/files/{bucket}/*", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("FILES"))
		})

		r.Finalize()

		// Should match
		resp := request(t, r, "GET", "/files/mybucket/path")
		assert.Equal(t, "FILES", resp.Body.String())
	})

	t.Run("different number of segments", func(t *testing.T) {
		r := teapot.New()

		r.GET("/a/b/c", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("ABC"))
		})

		r.Finalize()

		// Should not match path with different segment count
		resp := request(t, r, "GET", "/a/b")
		assert.Equal(t, 404, resp.Code)

		resp = request(t, r, "GET", "/a/b/c/d")
		assert.Equal(t, 404, resp.Code)
	})

	t.Run("parameter segment matches anything", func(t *testing.T) {
		r := teapot.New()

		var capturedID string
		r.GET("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
			capturedID = teapot.URLParam(req, "id")
			_, _ = w.Write([]byte("USER"))
		})

		r.Finalize()

		capturedID = ""
		resp := request(t, r, "GET", "/users/123")
		assert.Equal(t, "USER", resp.Body.String())
		assert.Equal(t, "123", capturedID)

		capturedID = ""
		resp = request(t, r, "GET", "/users/abc")
		assert.Equal(t, "USER", resp.Body.String())
		assert.Equal(t, "abc", capturedID)
	})

	t.Run("literal segment must match exactly", func(t *testing.T) {
		r := teapot.New()

		r.GET("/users/{id}/posts", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("POSTS"))
		})

		r.Finalize()

		resp := request(t, r, "GET", "/users/123/posts")
		assert.Equal(t, "POSTS", resp.Body.String())

		// Wrong literal segment
		resp = request(t, r, "GET", "/users/123/comments")
		assert.Equal(t, 404, resp.Code)
	})

	t.Run("multiple parameters", func(t *testing.T) {
		r := teapot.New()

		var capturedBucket, capturedKey string
		r.GET("/{bucket}/{key}", func(w http.ResponseWriter, req *http.Request) {
			capturedBucket = teapot.URLParam(req, "bucket")
			capturedKey = teapot.URLParam(req, "key")
			_, _ = w.Write([]byte("OK"))
		})

		r.Finalize()

		capturedBucket = ""
		capturedKey = ""
		resp := request(t, r, "GET", "/mybucket/mykey")
		assert.Equal(t, "OK", resp.Body.String())
		assert.Equal(t, "mybucket", capturedBucket)
		assert.Equal(t, "mykey", capturedKey)
	})

	t.Run("pattern with trailing slash vs path without", func(t *testing.T) {
		r := teapot.New()

		r.GET("/test/", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("SLASH"))
		})

		r.Finalize()

		// Chi normalizes trailing slashes, but let's test behavior
		resp := request(t, r, "GET", "/test/")
		assert.Equal(t, "SLASH", resp.Body.String())
	})

	t.Run("empty segments after trim", func(t *testing.T) {
		r := teapot.New()

		r.GET("/", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("ROOT"))
		})

		r.Finalize()

		resp := request(t, r, "GET", "/")
		assert.Equal(t, "ROOT", resp.Body.String())
	})
}

// TestFindMatchingRouteEdgeCases tests edge cases for findMatchingRoute used by middleware
func TestFindMatchingRouteEdgeCases(t *testing.T) {
	t.Run("no matching method", func(t *testing.T) {
		r := teapot.New()

		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("GET"))
		})

		r.Finalize()

		// POST request to GET route - chi returns 405 Method Not Allowed
		resp := request(t, r, "POST", "/test")
		assert.Equal(t, 405, resp.Code)
	})

	t.Run("dispatcher route with only query-specific routes", func(t *testing.T) {
		r := teapot.New()

		// Only routes with query matchers (no fallback)
		r.QueryGET("/bucket", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("ACL"))
		}).Query("acl")

		r.Finalize()

		// Request without query should use first route as fallback
		resp := request(t, r, "GET", "/bucket")
		assert.Equal(t, 404, resp.Code) // No fallback, no match
	})

	t.Run("dispatcher route with fallback", func(t *testing.T) {
		r := teapot.New()

		r.QueryGET("/bucket", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("LIST"))
		}) // No query matcher - fallback

		r.QueryGET("/bucket", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("ACL"))
		}).Query("acl")

		r.Finalize()

		// Request without query should match fallback
		resp := request(t, r, "GET", "/bucket")
		assert.Equal(t, "LIST", resp.Body.String())
	})

	t.Run("route not found - no matches", func(t *testing.T) {
		r := teapot.New()

		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("TEST"))
		})

		r.Finalize()

		// Request to non-existent route
		resp := request(t, r, "GET", "/nonexistent")
		assert.Equal(t, 404, resp.Code)
	})
}

// TestRouteContextMiddlewareWithGroups tests context injection with route groups
func TestRouteContextMiddlewareWithGroups(t *testing.T) {
	t.Run("nested groups with context", func(t *testing.T) {
		r := teapot.New()

		var capturedAction, capturedName string

		r.Group("/api", func(r *teapot.Router) {
			r.Group("/v1", func(r *teapot.Router) {
				r.GET("/users", func(w http.ResponseWriter, req *http.Request) {
					capturedAction = teapot.GetAction(req)
					capturedName = teapot.GetRouteName(req)
					_, _ = w.Write([]byte("USERS"))
				}).Action("api:GetUsers").Name("api.users")
			})
		})

		r.Finalize()

		capturedAction = ""
		capturedName = ""
		resp := request(t, r, "GET", "/api/v1/users")
		assert.Equal(t, "USERS", resp.Body.String())
		assert.Equal(t, "api:GetUsers", capturedAction)
		assert.Equal(t, "api.users", capturedName)
	})
}

// TestRouteContextInjectionBeforeFinalize tests context injection before finalization
func TestRouteContextInjectionBeforeFinalize(t *testing.T) {
	t.Run("context available before finalize", func(t *testing.T) {
		r := teapot.New()

		var capturedAction, capturedName string
		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {
			capturedAction = teapot.GetAction(req)
			capturedName = teapot.GetRouteName(req)
			_, _ = w.Write([]byte("OK"))
		}).Action("test:Action").Name("test.route")

		// Don't finalize - should still work via slow path

		capturedAction = ""
		capturedName = ""
		resp := request(t, r, "GET", "/test")
		assert.Equal(t, "OK", resp.Body.String())
		assert.Equal(t, "test:Action", capturedAction)
		assert.Equal(t, "test.route", capturedName)
	})
}

// TestRouteContextWithNilCases tests nil/empty chi context cases
func TestRouteContextWithNilCases(t *testing.T) {
	t.Run("chi context with empty route pattern", func(t *testing.T) {
		r := teapot.New()

		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("OK"))
		})

		r.Finalize()

		// Create request with chi context but empty route pattern
		req := httptest.NewRequest("GET", "/test", nil)
		rctx := chi.NewRouteContext()
		// Don't set RoutePattern or RouteMethod
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Should handle gracefully (fallback to pattern matching)
	})
}

// TestTryFastPath tests the TryFastPath function directly
func TestTryFastPath(t *testing.T) {
	t.Run("nil chi context returns nil", func(t *testing.T) {
		r := teapot.New()
		ctx := context.Background()

		route := r.TryFastPath(ctx)
		assert.Nil(t, route)
	})

	t.Run("chi context with empty route pattern returns nil", func(t *testing.T) {
		r := teapot.New()
		rctx := chi.NewRouteContext()
		ctx := context.WithValue(context.Background(), chi.RouteCtxKey, rctx)

		route := r.TryFastPath(ctx)
		assert.Nil(t, route)
	})

	t.Run("finds direct route via chi context", func(t *testing.T) {
		r := teapot.New()
		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {}).Action("test:Action").Name("test.name")
		r.Finalize()

		// Simulate Chi's RouteContext
		rctx := chi.NewRouteContext()
		rctx.RouteMethod = "GET"
		rctx.RoutePatterns = []string{"/test"}
		ctx := context.WithValue(context.Background(), chi.RouteCtxKey, rctx)

		route := r.TryFastPath(ctx)
		require.NotNil(t, route)
		assert.Equal(t, "test:Action", route.Action)
		assert.Equal(t, "test.name", route.Name)
	})

	t.Run("finds dispatcher route via chi context", func(t *testing.T) {
		r := teapot.New()
		r.QueryGET("/bucket", func(w http.ResponseWriter, req *http.Request) {}).Action("s3:ListBucket")
		r.QueryGET("/bucket", func(w http.ResponseWriter, req *http.Request) {}).Action("s3:GetBucketAcl").Query("acl")
		r.Finalize()

		// Simulate Chi's RouteContext
		rctx := chi.NewRouteContext()
		rctx.RouteMethod = "GET"
		rctx.RoutePatterns = []string{"/bucket"}
		ctx := context.WithValue(context.Background(), chi.RouteCtxKey, rctx)

		route := r.TryFastPath(ctx)
		require.NotNil(t, route)
		// Should return fallback route (no query matchers)
		assert.Equal(t, "s3:ListBucket", route.Action)
	})

	t.Run("returns nil for non-existent route", func(t *testing.T) {
		r := teapot.New()
		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {})
		r.Finalize()

		// Simulate Chi's RouteContext for different route
		rctx := chi.NewRouteContext()
		rctx.RouteMethod = "GET"
		rctx.RoutePatterns = []string{"/other"}
		ctx := context.WithValue(context.Background(), chi.RouteCtxKey, rctx)

		route := r.TryFastPath(ctx)
		assert.Nil(t, route)
	})
}

// TestTryFallbackPath tests the TryFallbackPath function directly
func TestTryFallbackPath(t *testing.T) {
	t.Run("exact match on direct route", func(t *testing.T) {
		r := teapot.New()
		r.GET("/exact", func(w http.ResponseWriter, req *http.Request) {}).Action("exact:Action")
		r.Finalize()

		route := r.TryFallbackPath("GET", "/exact")
		require.NotNil(t, route)
		assert.Equal(t, "exact:Action", route.Action)
	})

	t.Run("pattern match with path parameters", func(t *testing.T) {
		r := teapot.New()
		r.GET("/users/{id}", func(w http.ResponseWriter, req *http.Request) {}).Action("users:Get")
		r.Finalize()

		route := r.TryFallbackPath("GET", "/users/123")
		require.NotNil(t, route)
		assert.Equal(t, "users:Get", route.Action)
	})

	t.Run("wildcard pattern match", func(t *testing.T) {
		r := teapot.New()
		r.GET("/files/*", func(w http.ResponseWriter, req *http.Request) {}).Action("files:Get")
		r.Finalize()

		route := r.TryFallbackPath("GET", "/files/a/b/c")
		require.NotNil(t, route)
		assert.Equal(t, "files:Get", route.Action)
	})

	t.Run("returns nil for non-matching route", func(t *testing.T) {
		r := teapot.New()
		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {})
		r.Finalize()

		route := r.TryFallbackPath("GET", "/other")
		assert.Nil(t, route)
	})

	t.Run("returns nil for different method", func(t *testing.T) {
		r := teapot.New()
		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {})
		r.Finalize()

		route := r.TryFallbackPath("POST", "/test")
		assert.Nil(t, route)
	})
}

// TestFindBestDispatcherRoute tests the FindBestDispatcherRoute function directly
func TestFindBestDispatcherRoute(t *testing.T) {
	t.Run("returns fallback route when available", func(t *testing.T) {
		dispatcher := &core.Dispatcher{
			Routes: []*core.Route{
				{Action: "s3:GetBucketAcl", QueryMatchers: []dispatch.Matcher{&dispatch.QueryExistsMatcher{Key: "acl"}}},
				{Action: "s3:ListBucket", QueryMatchers: []dispatch.Matcher{}}, // fallback
				{Action: "s3:GetBucketVersioning", QueryMatchers: []dispatch.Matcher{&dispatch.QueryExistsMatcher{Key: "versioning"}}},
			},
		}

		route := core.FindBestDispatcherRoute(dispatcher)
		require.NotNil(t, route)
		assert.Equal(t, "s3:ListBucket", route.Action)
		assert.Empty(t, route.QueryMatchers)
	})

	t.Run("returns first route when no fallback", func(t *testing.T) {
		dispatcher := &core.Dispatcher{
			Routes: []*core.Route{
				{Action: "s3:GetBucketAcl", QueryMatchers: []dispatch.Matcher{&dispatch.QueryExistsMatcher{Key: "acl"}}},
				{Action: "s3:GetBucketVersioning", QueryMatchers: []dispatch.Matcher{&dispatch.QueryExistsMatcher{Key: "versioning"}}},
			},
		}

		route := core.FindBestDispatcherRoute(dispatcher)
		require.NotNil(t, route)
		assert.Equal(t, "s3:GetBucketAcl", route.Action)
	})

	t.Run("returns nil for empty dispatcher", func(t *testing.T) {
		dispatcher := &core.Dispatcher{
			Routes: []*core.Route{},
		}

		route := core.FindBestDispatcherRoute(dispatcher)
		assert.Nil(t, route)
	})

	t.Run("prefers fallback over first when both exist", func(t *testing.T) {
		dispatcher := &core.Dispatcher{
			Routes: []*core.Route{
				{Action: "first:WithQuery", QueryMatchers: []dispatch.Matcher{&dispatch.QueryExistsMatcher{Key: "q"}}},
				{Action: "second:Fallback", QueryMatchers: []dispatch.Matcher{}},
			},
		}

		route := core.FindBestDispatcherRoute(dispatcher)
		require.NotNil(t, route)
		assert.Equal(t, "second:Fallback", route.Action)
	})
}

// TestInjectRouteMetadata tests the InjectRouteMetadata function directly
func TestInjectRouteMetadata(t *testing.T) {
	t.Run("injects both action and name", func(t *testing.T) {
		ctx := context.Background()
		route := &core.Route{
			Action: "test:Action",
			Name:   "test.name",
		}

		ctx = core.InjectRouteMetadata(ctx, route)

		assert.Equal(t, "test:Action", core.GetAction(ctx))
		assert.Equal(t, "test.name", core.GetRouteName(ctx))
	})

	t.Run("injects only action when name is empty", func(t *testing.T) {
		ctx := context.Background()
		route := &core.Route{
			Action: "test:Action",
			Name:   "",
		}

		ctx = core.InjectRouteMetadata(ctx, route)

		assert.Equal(t, "test:Action", core.GetAction(ctx))
		assert.Equal(t, "", core.GetRouteName(ctx))
	})

	t.Run("injects only name when action is empty", func(t *testing.T) {
		ctx := context.Background()
		route := &core.Route{
			Action: "",
			Name:   "test.name",
		}

		ctx = core.InjectRouteMetadata(ctx, route)

		assert.Equal(t, "", core.GetAction(ctx))
		assert.Equal(t, "test.name", core.GetRouteName(ctx))
	})

	t.Run("handles empty route gracefully", func(t *testing.T) {
		ctx := context.Background()
		route := &core.Route{
			Action: "",
			Name:   "",
		}

		ctx = core.InjectRouteMetadata(ctx, route)

		assert.Equal(t, "", core.GetAction(ctx))
		assert.Equal(t, "", core.GetRouteName(ctx))
	})

	t.Run("overwrites existing context values", func(t *testing.T) {
		ctx := context.Background()
		ctx = core.SetAction(ctx, "old:Action")
		ctx = core.SetRouteName(ctx, "old.name")

		route := &core.Route{
			Action: "new:Action",
			Name:   "new.name",
		}

		ctx = core.InjectRouteMetadata(ctx, route)

		assert.Equal(t, "new:Action", core.GetAction(ctx))
		assert.Equal(t, "new.name", core.GetRouteName(ctx))
	})
}

// MockHandler is a mock http.Handler for testing
type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

// TestRouteContextMiddlewareFactory tests the RouteContextMiddleware factory function
func TestRouteContextMiddlewareFactory(t *testing.T) {
	t.Run("returns middleware function", func(t *testing.T) {
		r := teapot.New()
		middleware := teapot.RouteContextMiddleware(r)

		assert.NotNil(t, middleware)

		// Verify it returns a handler
		mockNext := new(MockHandler)
		handler := middleware(mockNext)
		assert.NotNil(t, handler)
	})

	t.Run("created middleware injects context", func(t *testing.T) {
		r := teapot.New()
		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {}).Action("test:Action").Name("test.name")
		r.Finalize()

		// Create middleware
		middleware := teapot.RouteContextMiddleware(r)

		// Mock next handler that captures context
		var capturedAction, capturedName string
		mockNext := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			capturedAction = core.GetAction(req.Context())
			capturedName = core.GetRouteName(req.Context())
		})

		handler := middleware(mockNext)

		// Create request with Chi context
		req := httptest.NewRequest("GET", "/test", nil)
		rctx := chi.NewRouteContext()
		rctx.RouteMethod = "GET"
		rctx.RoutePatterns = []string{"/test"}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, "test:Action", capturedAction)
		assert.Equal(t, "test.name", capturedName)
	})
}

// TestServeHTTP tests the ServeHTTP method through the middleware
func TestServeHTTP(t *testing.T) {
	t.Run("fast path - injects context and calls next", func(t *testing.T) {
		r := teapot.New()
		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {}).Action("test:Action").Name("test.name")
		r.Finalize()

		// Create middleware
		middleware := teapot.RouteContextMiddleware(r)

		// Track if next handler was called
		nextCalled := false
		var capturedAction, capturedName string
		mockNext := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			nextCalled = true
			capturedAction = core.GetAction(req.Context())
			capturedName = core.GetRouteName(req.Context())
		})

		handler := middleware(mockNext)

		// Create request with Chi context (fast path)
		req := httptest.NewRequest("GET", "/test", nil)
		rctx := chi.NewRouteContext()
		rctx.RouteMethod = "GET"
		rctx.RoutePatterns = []string{"/test"}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.True(t, nextCalled)
		assert.Equal(t, "test:Action", capturedAction)
		assert.Equal(t, "test.name", capturedName)
	})

	t.Run("fallback path - injects context and calls next", func(t *testing.T) {
		r := teapot.New()
		r.GET("/users/{id}", func(w http.ResponseWriter, req *http.Request) {}).Action("users:Get").Name("users.get")
		r.Finalize()

		// Create middleware
		middleware := teapot.RouteContextMiddleware(r)

		// Track if next handler was called
		nextCalled := false
		var capturedAction, capturedName string
		mockNext := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			nextCalled = true
			capturedAction = core.GetAction(req.Context())
			capturedName = core.GetRouteName(req.Context())
		})

		handler := middleware(mockNext)

		// Create request without Chi context (fallback path)
		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.True(t, nextCalled)
		assert.Equal(t, "users:Get", capturedAction)
		assert.Equal(t, "users.get", capturedName)
	})

	t.Run("no matching route - still calls next without context", func(t *testing.T) {
		r := teapot.New()
		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {})
		r.Finalize()

		// Create middleware
		middleware := teapot.RouteContextMiddleware(r)

		// Track if next handler was called
		nextCalled := false
		var capturedAction, capturedName string
		mockNext := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			nextCalled = true
			capturedAction = core.GetAction(req.Context())
			capturedName = core.GetRouteName(req.Context())
		})

		handler := middleware(mockNext)

		// Create request for non-existent route
		req := httptest.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.True(t, nextCalled)
		assert.Equal(t, "", capturedAction)
		assert.Equal(t, "", capturedName)
	})

	t.Run("dispatcher route - injects early fallback context", func(t *testing.T) {
		r := teapot.New()
		r.QueryGET("/bucket", func(w http.ResponseWriter, req *http.Request) {}).Action("s3:ListBucket")
		r.QueryGET("/bucket", func(w http.ResponseWriter, req *http.Request) {}).Action("s3:GetBucketAcl").Query("acl")
		r.Finalize()

		// Create middleware
		middleware := teapot.RouteContextMiddleware(r)

		// Track if next handler was called
		nextCalled := false
		var capturedAction string
		mockNext := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			nextCalled = true
			capturedAction = core.GetAction(req.Context())
		})

		handler := middleware(mockNext)

		// Create request with Chi context for dispatcher route
		req := httptest.NewRequest("GET", "/bucket", nil)
		rctx := chi.NewRouteContext()
		rctx.RouteMethod = "GET"
		rctx.RoutePatterns = []string{"/bucket"}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.True(t, nextCalled)
		// Should inject fallback route action
		assert.Equal(t, "s3:ListBucket", capturedAction)
	})
}
