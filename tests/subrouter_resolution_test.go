package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

func TestSubRouterResolutionBug(t *testing.T) {
	t.Run("subrouter with parameter shadows literal", func(t *testing.T) {
		r := teapot.New()

		// Global middleware that checks route name/action
		r.Use(teapot.RouteContextMiddleware(r))

		var resolvedName string
		var resolvedAction string
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req)
				resolvedName = teapot.GetRouteName(req)
				resolvedAction = teapot.GetAction(req)
			})
		})
		r.GET("/{greedy}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Sub-router mounted at /api
		api := teapot.New()

		// Route with parameter at first position
		api.GET("/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).Name("api.show").Action("ShowAction")

		// Literal route that should match /api/users
		api.GET("/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).Name("api.users").Action("UsersAction")

		r.Mount("/api", api)
		r.Finalize()

		// Request to /api/users
		req := httptest.NewRequest("GET", "/api/users", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "api.users", resolvedName, "Should resolve to api.users, not api.show")
		assert.Equal(t, "UsersAction", resolvedAction)
	})

	t.Run("late registration with parameter shadows literal", func(t *testing.T) {
		r := teapot.New()

		// Global middleware
		r.Use(teapot.RouteContextMiddleware(r))
		var resolvedName string
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req)
				resolvedName = teapot.GetRouteName(req)
			})
		})

		// Parameter route first
		r.GET("/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).Name("show")

		// Literal route later
		r.GET("/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).Name("users")

		r.Finalize()

		// Request to /users
		req := httptest.NewRequest("GET", "/users", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, "users", resolvedName)
	})
}
