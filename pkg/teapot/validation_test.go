package teapot_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

var dummyHandler = func(w http.ResponseWriter, r *http.Request) {}

// TestDuplicateRouteName verifies panic on duplicate method+name
func TestDuplicateRouteName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate route name")
		} else {
			msg := r.(string)
			if !strings.Contains(msg, "duplicate route") {
				t.Errorf("expected 'duplicate route' in panic message, got: %s", msg)
			}
			if !strings.Contains(msg, "GET:users") {
				t.Errorf("expected 'GET:users' in panic message, got: %s", msg)
			}
		}
	}()

	r := teapot.New()
	r.GET("/users", dummyHandler).Name("users")
	r.GET("/users", dummyHandler).Name("users") // Should panic
}

// TestSameNameDifferentMethodsSamePath is allowed (Laravel-style resources)
func TestSameNameDifferentMethodsSamePath(t *testing.T) {
	r := teapot.New()

	// This should NOT panic - same name, different methods, same path
	r.GET("/users/{id}", dummyHandler).Name("users.show")
	r.PUT("/users/{id}", dummyHandler).Name("users.show")
	r.DELETE("/users/{id}", dummyHandler).Name("users.show")

	// Verify all routes registered
	routes := r.Routes()
	if len(routes) != 3 {
		t.Errorf("expected 3 routes, got %d", len(routes))
	}
}

// TestSameNameDifferentMethodsDifferentPath should panic
func TestSameNameDifferentMethodsDifferentPath(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for same name with different paths")
		} else {
			msg := r.(string)
			if !strings.Contains(msg, "used with different paths") {
				t.Errorf("expected 'used with different paths' in panic message, got: %s", msg)
			}
			if !strings.Contains(msg, "users.show") {
				t.Errorf("expected 'users.show' in panic message, got: %s", msg)
			}
		}
	}()

	r := teapot.New()
	r.GET("/users/{id}", dummyHandler).Name("users.show")
	r.PUT("/users/{uuid}", dummyHandler).Name("users.show") // Different path - should panic
}

// TestDuplicateInNamedGroup verifies validation works in groups
func TestDuplicateInNamedGroup(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate route in group")
		} else {
			msg := r.(string)
			if !strings.Contains(msg, "duplicate route") {
				t.Errorf("expected 'duplicate route' in panic message, got: %s", msg)
			}
			// Should include full name with prefix
			if !strings.Contains(msg, "api.users") {
				t.Errorf("expected 'api.users' in panic message, got: %s", msg)
			}
		}
	}()

	r := teapot.New()
	r.NamedGroup("/api", "api", func(r *teapot.Router) {
		r.GET("/users", dummyHandler).Name("users")
		r.GET("/users", dummyHandler).Name("users") // Should panic with full name "api.users"
	})
}

// TestValidationWithQueryRoutes verifies validation works with QueryGET
func TestValidationWithQueryRoutes(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate query route name")
		}
	}()

	r := teapot.New()
	r.QueryGET("/bucket", dummyHandler).Name("bucket.list")
	r.QueryGET("/bucket", dummyHandler).Name("bucket.list") // Should panic
}

// TestNoNameNoValidation verifies routes without names don't trigger validation
func TestNoNameNoValidation(t *testing.T) {
	r := teapot.New()

	// These should NOT panic - no names assigned
	r.GET("/users", dummyHandler)
	r.GET("/users", dummyHandler)
	r.POST("/users", dummyHandler)

	// This is fine - multiple unnamed routes are allowed
	// (though it might cause routing issues, validation doesn't prevent it)
}

// TestPanicMessageQuality verifies panic messages are helpful
func TestPanicMessageQuality(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		} else {
			msg := r.(string)

			// Should include method
			if !strings.Contains(msg, "GET") {
				t.Errorf("panic message should include method, got: %s", msg)
			}

			// Should include route name
			if !strings.Contains(msg, "users.show") {
				t.Errorf("panic message should include route name, got: %s", msg)
			}

			// Should include both paths for comparison
			if !strings.Contains(msg, "/users/{id}") {
				t.Errorf("panic message should include existing path, got: %s", msg)
			}
			if !strings.Contains(msg, "/posts/{id}") {
				t.Errorf("panic message should include new path, got: %s", msg)
			}
		}
	}()

	r := teapot.New()
	r.GET("/users/{id}", dummyHandler).Name("users.show")
	r.GET("/posts/{id}", dummyHandler).Name("users.show")
}

// TestValidationBeforeFinalize ensures validation happens during registration, not finalization
func TestValidationBeforeFinalize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic during route registration")
		}
	}()

	r := teapot.New()
	r.GET("/test", dummyHandler).Name("test")
	r.GET("/test", dummyHandler).Name("test") // Should panic here
	r.Finalize()                              // Should never reach here
}
