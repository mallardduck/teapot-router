package teapot_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/internal/testutil"
	"github.com/mallardduck/teapot-router/pkg/teapot"
)

var dummyHandler http.Handler = http.HandlerFunc(testutil.NoopResponse)

// TestDuplicateRouteName verifies panic on duplicate method+name
func TestDuplicateRouteName(t *testing.T) {
	asserts := assert.New(t)
	msg := testutil.CapturePanic(func() {
		r := teapot.New()
		r.GET("/users", dummyHandler).Name("users")
		r.GET("/users", dummyHandler).Name("users") // Should panic
	})

	if msg == "" {
		t.Error("expected panic for duplicate route name")
	} else {
		asserts.Contains(msg, "duplicate route")
		asserts.Contains(msg, "GET")
		asserts.Contains(msg, "users")
	}
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
	msg := testutil.CapturePanic(func() {
		r := teapot.New()
		r.GET("/users/{id}", dummyHandler).Name("users.show")
		r.PUT("/users/{uuid}", dummyHandler).Name("users.show") // Different path - should panic
	})

	if msg == "" {
		t.Error("expected panic for same name with different paths")
	} else {
		asserts.Contains(msg, "teapot: route name")
		asserts.Contains(msg, "users.show")
		asserts.Contains(msg, "used with different paths")
	}
}

// TestDuplicateInNamedGroup verifies validation works in groups
func TestDuplicateInNamedGroup(t *testing.T) {
	asserts := assert.New(t)
	msg := testutil.CapturePanic(func() {
		r := teapot.New()
		r.NamedGroup("/api", "api", func(r *teapot.Router) {
			r.GET("/users", dummyHandler).Name("users")
			r.GET("/users", dummyHandler).Name("users") // Should panic with full name "api.users"
		})
	})

	if msg == "" {
		t.Error("expected panic for duplicate route in group")
	} else {
		asserts.Contains(msg, "duplicate route")
		// Should include full name with prefix
		asserts.Contains(msg, "api.users")
	}
}

// TestValidationWithQueryRoutes verifies validation works with QueryGET
func TestValidationWithQueryRoutes(t *testing.T) {
	msg := testutil.CapturePanic(func() {
		r := teapot.New()
		r.QueryGET("/bucket", dummyHandler).Name("bucket.list")
		r.QueryGET("/bucket", dummyHandler).Name("bucket.list") // Should panic
	})

	if msg == "" {
		t.Error("expected panic for duplicate query route name")
	}
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
	msg := testutil.CapturePanic(func() {
		r := teapot.New()
		r.GET("/users/{id}", dummyHandler).Name("users.show")
		r.GET("/posts/{id}", dummyHandler).Name("users.show")
	})

	if msg == "" {
		t.Fatal("expected panic")
	}

	// Should include method
	asserts.Contains(msg, "GET", "panic message should include method")

	// Should include route name
	asserts.Contains(msg, "users.show", "panic message should include route name")

	// Should include both paths for comparison
	asserts.Contains(msg, "/users/{id}", "panic message should include existing path")
	asserts.Contains(msg, "/posts/{id}", "panic message should include new path")
}

// TestValidationBeforeFinalize ensures validation happens during registration, not finalization
func TestValidationBeforeFinalize(t *testing.T) {
	msg := testutil.CapturePanic(func() {
		r := teapot.New()
		r.GET("/test", dummyHandler).Name("test")
		r.GET("/test", dummyHandler).Name("test") // Should panic here
		r.Finalize()                              // Should never reach here
	})

	if msg == "" {
		t.Error("expected panic during route registration")
	}
}
