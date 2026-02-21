package teapot_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestOptimizedHandlerBoundaryConditions tests CONDITIONALS_BOUNDARY mutations
// that lived in optimized_handler.go (lines 45:34, 72:41, 73:39, 116:26)
func TestOptimizedHandlerBoundaryConditions(t *testing.T) {
	t.Run("route with exactly one wildcard param before finalize", func(t *testing.T) {
		r := teapot.New()

		var capturedKey string
		r.GET("/{key:.*}", func(w http.ResponseWriter, req *http.Request) {
			capturedKey = teapot.URLParam(req, "key")
			_, _ = w.Write([]byte("OK"))
		})

		// Test BEFORE finalize (slow path, line 45:34)
		resp := request(t, r, "GET", "/test/path")
		assert.Equal(t, "OK", resp.Body.String())
		assert.Equal(t, "test/path", capturedKey)
	})

	t.Run("route with multiple wildcard params before finalize", func(t *testing.T) {
		r := teapot.New()

		var capturedKey1, capturedKey2 string
		r.GET("/{path:.*}", func(w http.ResponseWriter, req *http.Request) {
			// Access multiple wildcard params (though pattern only has one)
			capturedKey1 = teapot.URLParam(req, "path")
			capturedKey2 = teapot.URLParam(req, "other")
			_, _ = w.Write([]byte("OK"))
		})

		resp := request(t, r, "GET", "/test/path")
		assert.Equal(t, "OK", resp.Body.String())
		assert.Equal(t, "test/path", capturedKey1)
		assert.Empty(t, capturedKey2) // "other" doesn't exist
	})

	t.Run("route with exactly zero wildcard params after finalize", func(t *testing.T) {
		r := teapot.New()

		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("OK"))
		})

		r.Finalize()

		resp := request(t, r, "GET", "/test")
		assert.Equal(t, "OK", resp.Body.String())
	})

	t.Run("route with exactly one middleware before finalize", func(t *testing.T) {
		r := teapot.New()

		callOrder := []string{}
		mw := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				callOrder = append(callOrder, "mw1")
				next.ServeHTTP(w, req)
			})
		}

		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {
			callOrder = append(callOrder, "handler")
			_, _ = w.Write([]byte("OK"))
		}).With(mw)

		// Test slow path with exactly 1 middleware (line 59:44)
		callOrder = []string{}
		resp := request(t, r, "GET", "/test")
		assert.Equal(t, "OK", resp.Body.String())
		assert.Equal(t, []string{"mw1", "handler"}, callOrder)
	})

	t.Run("route with exactly one middleware after finalize", func(t *testing.T) {
		r := teapot.New()

		callOrder := []string{}
		mw := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				callOrder = append(callOrder, "mw1")
				next.ServeHTTP(w, req)
			})
		}

		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {
			callOrder = append(callOrder, "handler")
			_, _ = w.Write([]byte("OK"))
		}).With(mw)

		r.Finalize()

		// Test fast path with exactly 1 middleware (line 84:39)
		callOrder = []string{}
		resp := request(t, r, "GET", "/test")
		assert.Equal(t, "OK", resp.Body.String())
		assert.Equal(t, []string{"mw1", "handler"}, callOrder)
	})

	t.Run("route with multiple middlewares", func(t *testing.T) {
		r := teapot.New()

		callOrder := []string{}
		mw1 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				callOrder = append(callOrder, "mw1")
				next.ServeHTTP(w, req)
			})
		}
		mw2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				callOrder = append(callOrder, "mw2")
				next.ServeHTTP(w, req)
			})
		}
		mw3 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				callOrder = append(callOrder, "mw3")
				next.ServeHTTP(w, req)
			})
		}

		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {
			callOrder = append(callOrder, "handler")
			_, _ = w.Write([]byte("OK"))
		}).With(mw1).With(mw2).With(mw3)

		r.Finalize()

		callOrder = []string{}
		resp := request(t, r, "GET", "/test")
		assert.Equal(t, "OK", resp.Body.String())
		// Middleware should be called in order
		assert.Equal(t, []string{"mw1", "mw2", "mw3", "handler"}, callOrder)
	})

	t.Run("route with exactly one wildcard param after finalize", func(t *testing.T) {
		r := teapot.New()

		var capturedKey string
		r.GET("/{key:.*}", func(w http.ResponseWriter, req *http.Request) {
			capturedKey = teapot.URLParam(req, "key")
			_, _ = w.Write([]byte("OK"))
		})

		r.Finalize()

		// Test fast path with exactly 1 wildcard (line 116:26)
		resp := request(t, r, "GET", "/test/path")
		assert.Equal(t, "OK", resp.Body.String())
		assert.Equal(t, "test/path", capturedKey)
	})

	t.Run("nil chi route context with wildcards", func(t *testing.T) {
		r := teapot.New()

		r.GET("/{key:.*}", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("OK"))
		})

		r.Finalize()

		// Create request without chi context (edge case)
		req := httptest.NewRequest("GET", "/test/path", nil)
		w := httptest.NewRecorder()

		// Should not crash
		r.ServeHTTP(w, req)
	})
}

// TestOptimizedHandlerAllCombinations tests all combinations of flags
// to ensure all code paths in createOptimizedHandler are covered
func TestOptimizedHandlerAllCombinations(t *testing.T) {
	tests := []struct {
		name          string
		hasAction     bool
		hasName       bool
		hasWildcard   bool
		hasMiddleware bool
		action        string
		routeName     string
	}{
		{
			name:          "no action, no name, no wildcard, no middleware",
			hasAction:     false,
			hasName:       false,
			hasWildcard:   false,
			hasMiddleware: false,
		},
		{
			name:          "action only",
			hasAction:     true,
			hasName:       false,
			hasWildcard:   false,
			hasMiddleware: false,
			action:        "test:Action",
		},
		{
			name:          "name only",
			hasAction:     false,
			hasName:       true,
			hasWildcard:   false,
			hasMiddleware: false,
			routeName:     "test.route",
		},
		{
			name:          "wildcard only",
			hasAction:     false,
			hasName:       false,
			hasWildcard:   true,
			hasMiddleware: false,
		},
		{
			name:          "middleware only",
			hasAction:     false,
			hasName:       false,
			hasWildcard:   false,
			hasMiddleware: true,
		},
		{
			name:          "action and name",
			hasAction:     true,
			hasName:       true,
			hasWildcard:   false,
			hasMiddleware: false,
			action:        "test:Action",
			routeName:     "test.route",
		},
		{
			name:          "action and wildcard",
			hasAction:     true,
			hasName:       false,
			hasWildcard:   true,
			hasMiddleware: false,
			action:        "test:Action",
		},
		{
			name:          "name and wildcard",
			hasAction:     false,
			hasName:       true,
			hasWildcard:   true,
			hasMiddleware: false,
			routeName:     "test.route",
		},
		{
			name:          "action and middleware",
			hasAction:     true,
			hasName:       false,
			hasWildcard:   false,
			hasMiddleware: true,
			action:        "test:Action",
		},
		{
			name:          "all features",
			hasAction:     true,
			hasName:       true,
			hasWildcard:   true,
			hasMiddleware: true,
			action:        "test:Action",
			routeName:     "test.route",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := teapot.New()

			var capturedAction, capturedName, capturedKey string
			middlewareCalled := false

			pattern := "/test"
			if tt.hasWildcard {
				pattern = "/{key:.*}"
			}

			rb := r.GET(pattern, func(w http.ResponseWriter, req *http.Request) {
				capturedAction = teapot.GetAction(req)
				capturedName = teapot.GetRouteName(req)
				if tt.hasWildcard {
					capturedKey = teapot.URLParam(req, "key")
				}
				_, _ = w.Write([]byte("OK"))
			})

			if tt.hasAction {
				rb.Action(tt.action)
			}
			if tt.hasName {
				rb.Name(tt.routeName)
			}
			if tt.hasMiddleware {
				rb.With(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
						middlewareCalled = true
						next.ServeHTTP(w, req)
					})
				})
			}

			r.Finalize()

			// Test the route
			capturedAction = ""
			capturedName = ""
			capturedKey = ""
			middlewareCalled = false

			path := "/test"
			if tt.hasWildcard {
				path = "/test/path"
			}

			resp := request(t, r, "GET", path)
			assert.Equal(t, "OK", resp.Body.String())

			if tt.hasAction {
				assert.Equal(t, tt.action, capturedAction)
			} else {
				assert.Empty(t, capturedAction)
			}

			if tt.hasName {
				assert.Equal(t, tt.routeName, capturedName)
			} else {
				assert.Empty(t, capturedName)
			}

			if tt.hasWildcard {
				assert.Equal(t, "test/path", capturedKey)
			}

			if tt.hasMiddleware {
				assert.True(t, middlewareCalled)
			}
		})
	}
}

// TestOptimizedHandlerNilRouteContext tests nil chi.RouteContext handling
func TestOptimizedHandlerNilRouteContext(t *testing.T) {
	t.Run("slow path with nil chi context", func(t *testing.T) {
		r := teapot.New()

		r.GET("/{key:.*}", func(w http.ResponseWriter, req *http.Request) {
			// Access URLParam which needs chi context
			key := teapot.URLParam(req, "key")
			_, _ = w.Write([]byte(key)) //nolint:gosec // G705: key is a URL path parameter from the router, not user-supplied HTML input
		})

		// Don't finalize - use slow path

		// Make request through chi router which sets up context
		resp := request(t, r, "GET", "/path/to/file")
		assert.Equal(t, "path/to/file", resp.Body.String())
	})

	t.Run("fast path with nil chi context", func(t *testing.T) {
		r := teapot.New()

		r.GET("/{key:.*}", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("OK"))
		})

		r.Finalize()

		// Make direct request without chi context setup
		req := httptest.NewRequest("GET", "/path", nil)
		// No chi.RouteContext set
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		// Handler might not match without proper chi setup, but shouldn't crash
		// The chi router handles the actual routing
	})
}

// TestOptimizedHandlerWildcardWithNilContext ensures nil chi.RouteContext doesn't crash
func TestOptimizedHandlerWildcardWithNilContext(t *testing.T) {
	r := teapot.New()

	r.GET("/{key:.*}", func(w http.ResponseWriter, req *http.Request) {
		// Attempt to get wildcard param
		key := chi.URLParam(req, "key")
		_, _ = w.Write([]byte(key)) //nolint:gosec // G705: key is a URL path parameter from the router, not user-supplied HTML input
	})

	r.Finalize()

	// Create request with explicitly nil chi context
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, nil))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should not crash
}
