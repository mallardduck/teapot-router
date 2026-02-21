package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mallardduck/teapot-router/internal/testutil"
	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestRouteNaming tests route naming functionality to kill mutants in router.go:77
func TestRouteNaming(t *testing.T) {
	t.Run("simple route name", func(t *testing.T) {
		r := teapot.New()

		r.Func().GET("/test", testutil.NoopResponse).Name("test.route")

		routes := r.Routes()
		require.Len(t, routes, 1)
		assert.Equal(t, "test.route", routes[0].Name)
	})

	t.Run("route name with prefix from group", func(t *testing.T) {
		r := teapot.New()

		r.NamedGroup("/api", "api", func(sub *teapot.Router) {
			sub.Func().GET("/users", testutil.NoopResponse).Name("users.list")
		})

		routes := r.Routes()
		require.Len(t, routes, 1)
		// Name prefix should be concatenated (line 77: fullName := rb.router.namePrefix + name)
		// Test EXACT concatenation to catch ARITHMETIC_BASE mutations
		assert.Equal(t, "api.users.list", routes[0].Name)
		// Verify it's not just "users.list" or "apiusers.list"
		assert.NotEqual(t, "users.list", routes[0].Name)
		assert.NotEqual(t, "apiusers.list", routes[0].Name)
		// Verify prefix is actually present
		assert.True(t, len("api.users.list") > len("users.list"))
		assert.Contains(t, routes[0].Name, "api.")
		assert.Contains(t, routes[0].Name, "users.list")
	})

	t.Run("nested group name prefixes", func(t *testing.T) {
		r := teapot.New()

		r.NamedGroup("/api", "api", func(sub *teapot.Router) {
			sub.NamedGroup("/v1", "v1", func(sub2 *teapot.Router) {
				sub2.Func().GET("/users", testutil.NoopResponse).Name("users")
			})
		})

		routes := r.Routes()
		require.Len(t, routes, 1)
		// Should concatenate all prefixes
		assert.Equal(t, "api.v1.users", routes[0].Name)
	})

	t.Run("route without name has empty string", func(t *testing.T) {
		r := teapot.New()

		r.Func().GET("/test", testutil.NoopResponse)

		routes := r.Routes()
		require.Len(t, routes, 1)
		assert.Equal(t, "", routes[0].Name)
	})
}

// TestDuplicateRouteNames tests duplicate name detection to kill mutants at router.go:82,84,90
func TestDuplicateRouteNames(t *testing.T) {
	t.Run("duplicate name same method panics", func(t *testing.T) {
		r := teapot.New()

		r.Func().GET("/test1", testutil.NoopResponse).Name("duplicate")

		// Line 84: if existingRoute.Method == rb.route.Method
		// Verify panic occurs and capture error message
		var panicMsg string
		assert.Panics(t, func() {
			panicMsg = testutil.CapturePanic(func() {
				r.Func().GET("/test2", testutil.NoopResponse).Name("duplicate")
			})
			if panicMsg != "" {
				panic(panicMsg) // re-panic for assert.Panics
			}
		}, "should panic on duplicate name with same method but different pattern")

		// Verify panic message contains expected content
		assert.Contains(t, panicMsg, "duplicate")
		assert.Contains(t, panicMsg, "GET")
		assert.Contains(t, panicMsg, "test1")
		assert.Contains(t, panicMsg, "test2")
	})

	t.Run("duplicate name different methods same pattern ok", func(t *testing.T) {
		r := teapot.New()

		// Same name, different methods, same pattern - this is OK (Laravel-style resources)
		r.Func().GET("/test", testutil.NoopResponse).Name("resource")
		r.Func().POST("/test", testutil.NoopResponse).Name("resource")

		routes := r.Routes()
		assert.Len(t, routes, 2)

		// Verify both have same name and pattern but different methods
		assert.Equal(t, "resource", routes[0].Name)
		assert.Equal(t, "resource", routes[1].Name)
		assert.Equal(t, "/test", routes[0].Pattern)
		assert.Equal(t, "/test", routes[1].Pattern)
		// Methods must be different
		assert.NotEqual(t, routes[0].Method, routes[1].Method)
		// One should be GET, one should be POST
		methods := []string{routes[0].Method, routes[1].Method}
		assert.Contains(t, methods, "GET")
		assert.Contains(t, methods, "POST")
	})

	t.Run("duplicate name different methods different patterns panics", func(t *testing.T) {
		r := teapot.New()

		r.Func().GET("/test1", testutil.NoopResponse).Name("conflict")

		// Line 90: if existingRoute.Pattern != rb.route.Pattern
		assert.Panics(t, func() {
			r.Func().POST("/test2", testutil.NoopResponse).Name("conflict")
		}, "should panic on duplicate name with different methods and different patterns")
	})

	t.Run("same name in different groups with prefix ok", func(t *testing.T) {
		r := teapot.New()

		r.NamedGroup("/api", "api", func(sub *teapot.Router) {
			sub.Func().GET("/users", testutil.NoopResponse).Name("users")
		})

		r.NamedGroup("/admin", "admin", func(sub *teapot.Router) {
			sub.Func().GET("/users", testutil.NoopResponse).Name("users")
		})

		routes := r.Routes()
		assert.Len(t, routes, 2)
		assert.Equal(t, "api.users", routes[0].Name)
		assert.Equal(t, "admin.users", routes[1].Name)
	})

	t.Run("URL generation requires unique route names", func(t *testing.T) {
		r := teapot.New()

		r.Func().GET("/users", testutil.NoopResponse).Name("users.list")
		r.Func().GET("/users/{id}", testutil.NoopResponse).Name("users.show")

		// Should be able to generate URLs for both
		url1, err := r.URL("users.list")
		assert.NoError(t, err)
		assert.Equal(t, "/users", url1)

		url2, err := r.URL("users.show", "id", "123")
		assert.NoError(t, err)
		assert.Equal(t, "/users/123", url2)
	})
}

// TestRouteNameValidation tests edge cases in name validation
func TestRouteNameValidation(t *testing.T) {
	t.Run("empty name is valid", func(t *testing.T) {
		r := teapot.New()

		// Empty name should be OK
		r.Func().GET("/test", testutil.NoopResponse).Name("")

		routes := r.Routes()
		require.Len(t, routes, 1)
		assert.Equal(t, "", routes[0].Name)
	})

	t.Run("multiple routes without names ok", func(t *testing.T) {
		r := teapot.New()

		r.Func().GET("/test1", testutil.NoopResponse)
		r.Func().GET("/test2", testutil.NoopResponse)
		r.Func().GET("/test3", testutil.NoopResponse)

		routes := r.Routes()
		assert.Len(t, routes, 3)
	})

	t.Run("name can be set after route creation", func(t *testing.T) {
		r := teapot.New()

		rb := r.Func().GET("/test", testutil.NoopResponse)
		rb.Name("my.route")

		routes := r.Routes()
		require.Len(t, routes, 1)
		assert.Equal(t, "my.route", routes[0].Name)
	})

	t.Run("name and action can both be set", func(t *testing.T) {
		r := teapot.New()

		r.Func().GET("/test", testutil.NoopResponse).
			Name("test.route").
			Action("s3:GetObject")

		routes := r.Routes()
		require.Len(t, routes, 1)
		assert.Equal(t, "test.route", routes[0].Name)
		assert.Equal(t, "s3:GetObject", routes[0].Action)
	})
}
