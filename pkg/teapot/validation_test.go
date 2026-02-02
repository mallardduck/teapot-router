package teapot_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

var dummyHandler = func(w http.ResponseWriter, r *http.Request) {}

// TestDuplicateRouteName verifies panic on duplicate method+name
func TestDuplicateRouteName(t *testing.T) {
	asserts := assert.New(t)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate route name")
		} else {
			msg := r.(string)
			asserts.Contains(msg, "duplicate route")
			asserts.Contains(msg, "GET:users")
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
	assert.Len(t, routes, 3)
}

// TestSameNameDifferentMethodsDifferentPath should panic
func TestSameNameDifferentMethodsDifferentPath(t *testing.T) {
	asserts := assert.New(t)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for same name with different paths")
		} else {
			msg := r.(string)
			asserts.Contains(msg, "used with different paths")
			asserts.Contains(msg, "users.show")
		}
	}()

	r := teapot.New()
	r.GET("/users/{id}", dummyHandler).Name("users.show")
	r.PUT("/users/{uuid}", dummyHandler).Name("users.show") // Different path - should panic
}

// TestDuplicateInNamedGroup verifies validation works in groups
func TestDuplicateInNamedGroup(t *testing.T) {
	asserts := assert.New(t)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate route in group")
		} else {
			msg := r.(string)
			asserts.Contains(msg, "duplicate route")
			// Should include full name with prefix
			asserts.Contains(msg, "api.users")
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
	asserts := assert.New(t)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		} else {
			msg := r.(string)

			// Should include method
			asserts.Contains(msg, "GET", "panic message should include method")

			// Should include route name
			asserts.Contains(msg, "users.show", "panic message should include route name")

			// Should include both paths for comparison
			asserts.Contains(msg, "/users/{id}", "panic message should include existing path")
			asserts.Contains(msg, "/posts/{id}", "panic message should include new path")
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
