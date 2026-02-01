package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestURLParams verifies URLParams helper returns all URL parameters
func TestURLParams(t *testing.T) {
	r := teapot.New()

	r.GET("/users/{id}/posts/{postId}", func(w http.ResponseWriter, r *http.Request) {
		params := teapot.URLParams(r)

		if len(params) != 2 {
			t.Errorf("expected 2 params, got %d", len(params))
		}
		if params["id"] != "123" {
			t.Errorf("expected id=123, got %q", params["id"])
		}
		if params["postId"] != "456" {
			t.Errorf("expected postId=456, got %q", params["postId"])
		}

		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/users/123/posts/456", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// TestURLParamsEmpty verifies URLParams returns empty map when no params
func TestURLParamsEmpty(t *testing.T) {
	r := teapot.New()

	r.GET("/simple", func(w http.ResponseWriter, r *http.Request) {
		params := teapot.URLParams(r)

		if params == nil {
			t.Error("expected empty map, got nil")
		}
		if len(params) != 0 {
			t.Errorf("expected 0 params, got %d", len(params))
		}

		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/simple", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
}

// TestChiAccessor verifies Chi() returns the underlying chi.Router
func TestChiAccessor(t *testing.T) {
	r := teapot.New()

	// Set custom NotFound handler via Chi
	notFoundCalled := false
	r.Chi().NotFound(func(w http.ResponseWriter, r *http.Request) {
		notFoundCalled = true
		w.WriteHeader(404)
		w.Write([]byte("Custom 404"))
	})

	r.GET("/exists", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	// Test normal route works
	req := httptest.NewRequest("GET", "/exists", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Test custom 404 handler is called
	req = httptest.NewRequest("GET", "/nonexistent", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !notFoundCalled {
		t.Error("custom NotFound handler was not called")
	}
	if w.Body.String() != "Custom 404" {
		t.Errorf("expected 'Custom 404', got %q", w.Body.String())
	}
}

// TestRouterWith verifies With() method creates router with middleware
func TestRouterWith(t *testing.T) {
	r := teapot.New()

	// Track middleware calls
	var calls []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "mw1")
			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "mw2")
			next.ServeHTTP(w, r)
		})
	}

	// Route without middleware
	r.GET("/public", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PUBLIC"))
	})

	// Route with mw1
	r.With(mw1).GET("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PROTECTED"))
	})

	// Route with mw1 and mw2
	r.With(mw1, mw2).GET("/admin", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ADMIN"))
	})

	// Test public route
	calls = nil
	req := httptest.NewRequest("GET", "/public", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if len(calls) != 0 {
		t.Errorf("public route should have no middleware, got %v", calls)
	}

	// Test protected route
	calls = nil
	req = httptest.NewRequest("GET", "/protected", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if len(calls) != 1 || calls[0] != "mw1" {
		t.Errorf("protected route expected [mw1], got %v", calls)
	}

	// Test admin route
	calls = nil
	req = httptest.NewRequest("GET", "/admin", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if len(calls) != 2 || calls[0] != "mw1" || calls[1] != "mw2" {
		t.Errorf("admin route expected [mw1 mw2], got %v", calls)
	}
}

// TestRouterWithChaining verifies With() can be chained
func TestRouterWithChaining(t *testing.T) {
	r := teapot.New()

	var order []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "1")
			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "2")
			next.ServeHTTP(w, r)
		})
	}

	// Chain With() calls
	r.With(mw1).With(mw2).GET("/test", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expected := []string{"1", "2", "handler"}
	if len(order) != len(expected) {
		t.Errorf("expected %v, got %v", expected, order)
	}
	for i, v := range expected {
		if i >= len(order) || order[i] != v {
			t.Errorf("expected %v, got %v", expected, order)
			break
		}
	}
}

// TestMiddlewareGroup verifies MiddlewareGroup applies middleware without path/name changes
func TestMiddlewareGroup(t *testing.T) {
	r := teapot.New()

	var calls []string

	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "auth")
			next.ServeHTTP(w, r)
		})
	}

	logging := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "logging")
			next.ServeHTTP(w, r)
		})
	}

	// Public routes (no middleware)
	r.GET("/public", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PUBLIC"))
	}).Name("public")

	// Protected routes (with auth + logging middleware)
	r.MiddlewareGroup(func(r *teapot.Router) {
		r.GET("/admin", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ADMIN"))
		}).Name("admin")

		r.GET("/dashboard", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("DASHBOARD"))
		}).Name("dashboard")
	}, auth, logging)

	// Test public route (no middleware)
	calls = nil
	req := httptest.NewRequest("GET", "/public", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if len(calls) != 0 {
		t.Errorf("public route should have no middleware, got %v", calls)
	}

	// Test admin route (has middleware)
	calls = nil
	req = httptest.NewRequest("GET", "/admin", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	expected := []string{"auth", "logging"}
	if len(calls) != 2 || calls[0] != "auth" || calls[1] != "logging" {
		t.Errorf("admin route expected %v, got %v", expected, calls)
	}

	// Test dashboard route (has middleware)
	calls = nil
	req = httptest.NewRequest("GET", "/dashboard", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if len(calls) != 2 || calls[0] != "auth" || calls[1] != "logging" {
		t.Errorf("dashboard route expected %v, got %v", expected, calls)
	}

	// Verify route names are correct (no prefix added)
	url, err := r.URL("admin")
	if err != nil {
		t.Errorf("expected admin route to exist, got error: %v", err)
	}
	if url != "/admin" {
		t.Errorf("expected /admin, got %s", url)
	}
}

// TestMiddlewareGroupNested verifies MiddlewareGroup can be nested
func TestMiddlewareGroupNested(t *testing.T) {
	r := teapot.New()

	var order []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1")
			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2")
			next.ServeHTTP(w, r)
		})
	}

	mw3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw3")
			next.ServeHTTP(w, r)
		})
	}

	r.MiddlewareGroup(func(r *teapot.Router) {
		r.MiddlewareGroup(func(r *teapot.Router) {
			r.GET("/nested", func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "handler")
				w.WriteHeader(200)
			})
		}, mw2, mw3)
	}, mw1)

	req := httptest.NewRequest("GET", "/nested", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expected := []string{"mw1", "mw2", "mw3", "handler"}
	if len(order) != len(expected) {
		t.Errorf("expected %v, got %v", expected, order)
	}
	for i, v := range expected {
		if i >= len(order) || order[i] != v {
			t.Errorf("expected %v, got %v", expected, order)
			break
		}
	}
}

// TestMiddlewareGroupWithNamedGroup verifies MiddlewareGroup works with NamedGroup
func TestMiddlewareGroupWithNamedGroup(t *testing.T) {
	r := teapot.New()

	var calls []string

	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "auth")
			next.ServeHTTP(w, r)
		})
	}

	r.MiddlewareGroup(func(r *teapot.Router) {
		r.NamedGroup("/api", "api", func(r *teapot.Router) {
			r.GET("/users", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("USERS"))
			}).Name("users")
		})
	}, auth)

	// Test route has middleware and correct path/name
	calls = nil
	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if len(calls) != 1 || calls[0] != "auth" {
		t.Errorf("expected [auth], got %v", calls)
	}

	// Verify route name includes prefix
	url, err := r.URL("api.users")
	if err != nil {
		t.Errorf("expected api.users route to exist, got error: %v", err)
	}
	if url != "/api/users" {
		t.Errorf("expected /api/users, got %s", url)
	}
}
