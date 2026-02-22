package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

func TestRouterImprovements(t *testing.T) {
	t.Run("Handle supports http.Handler", func(t *testing.T) {
		r := teapot.New()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		r.Handle("GET", "/test", handler).Name("test")

		ts := httptest.NewServer(r)
		defer ts.Close()

		res, err := http.Get(ts.URL + "/test")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Errorf("expected OK, got %v", res.Status)
		}

		// Verify name registration
		url, err := r.URL("test")
		if err != nil {
			t.Fatal(err)
		}
		if url != "/test" {
			t.Errorf("expected /test, got %q", url)
		}
	})

	t.Run("RegisterExternal (phantom routes)", func(t *testing.T) {
		r := teapot.New()
		r.RegisterExternal("GET", "/external", "ext", "ExternalAction")

		routes := r.Routes()
		found := false
		for _, rt := range routes {
			if rt.Pattern == "/external" && rt.Name == "ext" && rt.Action == "ExternalAction" {
				found = true
				break
			}
		}
		if !found {
			t.Error("RegisterExternal route not found in Routes()")
		}

		// Verify URL generation works for phantom routes
		url, err := r.URL("ext")
		if err != nil {
			t.Fatal(err)
		}
		if url != "/external" {
			t.Errorf("expected /external, got %q", url)
		}

		// Verify it doesn't actually dispatch
		ts := httptest.NewServer(r)
		defer ts.Close()
		res, err := http.Get(ts.URL + "/external")
		assert.NoError(t, err)
		defer res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 for phantom route, got %v", res.Status)
		}
	})

	t.Run("Mount propagation", func(t *testing.T) {
		parent := teapot.New()
		child := teapot.New()

		child.GET("/hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).Name("hello")
		parent.Mount("/api", child)

		routes := parent.Routes()
		found := false
		for _, rt := range routes {
			if rt.Pattern == "/api/hello" && rt.Name == "hello" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Mounted route not propagated correctly")
		}

		// Verify URL generation in parent
		url, err := parent.URL("hello")
		if err != nil {
			t.Fatal(err)
		}
		if url != "/api/hello" {
			t.Errorf("expected /api/hello, got %q", url)
		}
	})

	t.Run("SubRouter", func(t *testing.T) {
		r := teapot.New()
		sub := r.SubRouter("/admin")
		sub.GET("/dashboard", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).Name("dash")

		routes := r.Routes()
		found := false
		for _, rt := range routes {
			if rt.Pattern == "/admin/dashboard" && rt.Name == "dash" {
				found = true
				break
			}
		}
		if !found {
			t.Error("SubRouter route not propagated correctly")
		}

		// Verify URL generation in parent
		url, err := r.URL("dash")
		if err != nil {
			t.Fatal(err)
		}
		if url != "/admin/dashboard" {
			t.Errorf("expected /admin/dashboard, got %q", url)
		}
	})

	t.Run("SubRouter with wildcards", func(t *testing.T) {
		r := teapot.New()
		sub := r.SubRouter("/files")
		sub.GET("/{bucket}/{key:.*}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).Name("get_file")

		url, err := r.URL("get_file", "bucket", "mybucket", "key", "path/to/file.txt")
		if err != nil {
			t.Fatal(err)
		}
		if url != "/files/mybucket/path/to/file.txt" {
			t.Errorf("expected /files/mybucket/path/to/file.txt, got %q", url)
		}
	})

	t.Run("AggregateRoutes", func(t *testing.T) {
		r1 := teapot.New()
		r1.GET("/r1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

		r2 := teapot.New()
		r2.GET("/r2", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

		combined := teapot.AggregateRoutes(r1, r2)
		if len(combined) != 2 {
			t.Errorf("expected 2 routes, got %d", len(combined))
		}
	})
}
