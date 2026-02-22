package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

func TestExactMatch(t *testing.T) {
	r := teapot.New()
	r.SetDebugLog(true)

	r.GET("/{$}", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("root"))
	}))

	r.GET("/{path}", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("child"))
	}))

	t.Run("root match", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "root", w.Body.String())
	})

	t.Run("child match", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/abc", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "child", w.Body.String())
	})

	t.Run("root nested not match", func(t *testing.T) {
		// "/{$}" should not match "/abc"
		// Chi v5.1+ handles this if passed through
		req := httptest.NewRequest("GET", "/abc", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		// It should hit /{path} instead of /{$}
		assert.Equal(t, "child", w.Body.String())
	})
}

func TestExactMatchInGroups(t *testing.T) {
	r := teapot.New()

	r.Group("/admin", func(r *teapot.Router) {
		r.GET("/{$}", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("admin-root"))
		}))
		r.GET("/{path}", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("admin-child"))
		}))
	})

	t.Run("admin root match", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "admin-root", w.Body.String())
	})

	t.Run("URL generation with exact match", func(t *testing.T) {
		r := teapot.New()
		r.GET("/users/{$}", http.HandlerFunc(dummyHandler)).Name("users.index")

		url := r.MustURL("users.index")
		// For URL generation, {$} should be removed to produce /users/
		assert.Equal(t, "/users/", url)
	})
}

func dummyHandler(w http.ResponseWriter, req *http.Request) {}
