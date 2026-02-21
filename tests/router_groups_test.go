package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestGroupNamePrefix tests name prefix handling to kill mutant at router.go:405
func TestGroupNamePrefix(t *testing.T) {
	t.Run("group with empty name prefix uses parent prefix", func(t *testing.T) {
		r := teapot.New()

		// Outer group with prefix
		r.NamedGroup("/api", "api", func(sub *teapot.Router) {
			// Inner group with empty prefix - line 405: if namePrefix == ""
			sub.NamedGroup("/v1", "", func(sub2 *teapot.Router) {
				sub2.Func().GET("/test", func(w http.ResponseWriter, req *http.Request) {}).Name("test")
			})
		})

		routes := r.Routes()
		require.Len(t, routes, 1)
		// Empty prefix should inherit parent's "api." prefix
		// Test to catch CONDITIONALS_NEGATION at router.go:405
		assert.Equal(t, "api.test", routes[0].Name)
		// Should NOT be just "test" (would happen if condition was negated)
		assert.NotEqual(t, "test", routes[0].Name)
		// Should NOT have extra dot "api..test"
		assert.NotEqual(t, "api..test", routes[0].Name)
		// Must contain parent prefix
		assert.Contains(t, routes[0].Name, "api.")
	})

	t.Run("group with non-empty prefix concatenates", func(t *testing.T) {
		r := teapot.New()

		r.NamedGroup("/api", "api", func(sub *teapot.Router) {
			sub.NamedGroup("/v1", "v1", func(sub2 *teapot.Router) {
				sub2.Func().GET("/test", func(w http.ResponseWriter, req *http.Request) {}).Name("test")
			})
		})

		routes := r.Routes()
		require.Len(t, routes, 1)
		assert.Equal(t, "api.v1.test", routes[0].Name)
	})

	t.Run("root group with empty prefix", func(t *testing.T) {
		r := teapot.New()

		// Root level group with empty prefix
		r.NamedGroup("/api", "", func(sub *teapot.Router) {
			sub.Func().GET("/test", func(w http.ResponseWriter, req *http.Request) {}).Name("test")
		})

		routes := r.Routes()
		require.Len(t, routes, 1)
		// No prefix at all
		assert.Equal(t, "test", routes[0].Name)
	})

	t.Run("deeply nested groups with mixed empty prefixes", func(t *testing.T) {
		r := teapot.New()

		r.NamedGroup("/a", "a", func(sub *teapot.Router) {
			sub.NamedGroup("/b", "", func(sub2 *teapot.Router) { // Empty inherits "a"
				sub2.NamedGroup("/c", "c", func(sub3 *teapot.Router) { // Adds "c" to "a"
					sub3.Func().GET("/test", func(w http.ResponseWriter, req *http.Request) {}).Name("test")
				})
			})
		})

		routes := r.Routes()
		require.Len(t, routes, 1)
		assert.Equal(t, "a.c.test", routes[0].Name)
	})
}

// TestGroupPathPrefix tests path prefix handling to kill mutant at router.go:245
func TestGroupPathPrefix(t *testing.T) {
	t.Run("group applies path prefix", func(t *testing.T) {
		r := teapot.New()

		r.Group("/api", func(sub *teapot.Router) {
			sub.Func().GET("/users", func(w http.ResponseWriter, req *http.Request) {
				_, _ = w.Write([]byte("users"))
			})
		})

		r.Finalize()

		// Line 245: fullPattern := r.pathPrefix + pattern
		req := httptest.NewRequest("GET", "/api/users", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "users", w.Body.String())
	})

	t.Run("nested groups concatenate path prefixes", func(t *testing.T) {
		r := teapot.New()

		r.Group("/api", func(sub *teapot.Router) {
			sub.Group("/v1", func(sub2 *teapot.Router) {
				sub2.Func().GET("/test", func(w http.ResponseWriter, req *http.Request) {
					_, _ = w.Write([]byte("nested"))
				})
			})
		})

		r.Finalize()

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "nested", w.Body.String())
	})

	t.Run("empty path prefix works", func(t *testing.T) {
		r := teapot.New()

		r.Group("", func(sub *teapot.Router) {
			sub.Func().GET("/test", func(w http.ResponseWriter, req *http.Request) {
				_, _ = w.Write([]byte("ok"))
			})
		})

		r.Finalize()

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "ok", w.Body.String())
	})

	t.Run("routes report full pattern with prefix", func(t *testing.T) {
		r := teapot.New()

		r.NamedGroup("/api", "api", func(sub *teapot.Router) {
			sub.Func().GET("/users", func(w http.ResponseWriter, req *http.Request) {}).Name("users")
		})

		routes := r.Routes()
		require.Len(t, routes, 1)
		// Pattern should include the full path with prefix
		// Test EXACT concatenation to catch ARITHMETIC_BASE mutations at router.go:245
		assert.Equal(t, "/api/users", routes[0].Pattern)
		// Verify it's not malformed
		assert.NotEqual(t, "/users", routes[0].Pattern)
		assert.NotEqual(t, "/apiusers", routes[0].Pattern)
		assert.NotEqual(t, "api/users", routes[0].Pattern)
		// Verify both parts are present
		assert.Contains(t, routes[0].Pattern, "/api")
		assert.Contains(t, routes[0].Pattern, "/users")
		assert.True(t, len("/api/users") == len(routes[0].Pattern))
	})
}

// TestGroupMiddleware tests middleware inheritance in groups
func TestGroupMiddleware(t *testing.T) {
	t.Run("group middleware applies to routes", func(t *testing.T) {
		r := teapot.New()
		called := false

		mw := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				called = true
				next.ServeHTTP(w, req)
			})
		}

		r.Group("/api", func(sub *teapot.Router) {
			sub.Use(mw)
			sub.Func().GET("/test", func(w http.ResponseWriter, req *http.Request) {
				_, _ = w.Write([]byte("ok"))
			})
		})

		r.Finalize()

		called = false
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.True(t, called, "middleware should be called")
		assert.Equal(t, 200, w.Code)
	})

	t.Run("nested groups inherit parent middleware", func(t *testing.T) {
		r := teapot.New()
		calls := []string{}

		mw1 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls = append(calls, "mw1")
				next.ServeHTTP(w, req)
			})
		}
		mw2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls = append(calls, "mw2")
				next.ServeHTTP(w, req)
			})
		}

		r.Group("/api", func(sub *teapot.Router) {
			sub.Use(mw1)
			sub.Group("/v1", func(sub2 *teapot.Router) {
				sub2.Use(mw2)
				sub2.Func().GET("/test", func(w http.ResponseWriter, req *http.Request) {
					calls = append(calls, "handler")
					_, _ = w.Write([]byte("ok"))
				})
			})
		})

		r.Finalize()

		calls = []string{}
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Middleware should be called in order: mw1, then mw2, then handler
		assert.Equal(t, []string{"mw1", "mw2", "handler"}, calls)
	})
}

// TestRouteMethod tests Route() method for Chi-style routing
func TestRouteMethod(t *testing.T) {
	t.Run("Route creates sub-router with Chi semantics", func(t *testing.T) {
		r := teapot.New()

		r.Route("/api", func(sub *teapot.Router) {
			sub.Func().GET("/test", func(w http.ResponseWriter, req *http.Request) {
				_, _ = w.Write([]byte("ok"))
			})
		})

		r.Finalize()

		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "ok", w.Body.String())
	})

	t.Run("Route with RouteContextMiddleware has fast path", func(t *testing.T) {
		r := teapot.New()

		r.Route("/api", func(sub *teapot.Router) {
			sub.Use(teapot.RouteContextMiddleware(r))
			sub.Func().GET("/test", func(w http.ResponseWriter, req *http.Request) {
				// Should have access to route context here
				action := teapot.GetAction(req)
				_, _ = w.Write([]byte(action))
			}).Action("test:Action")
		})

		r.Finalize()

		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "test:Action", w.Body.String())
	})
}
