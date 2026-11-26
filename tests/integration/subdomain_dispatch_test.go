package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/dispatch"
	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestSubdomainDispatcherMountedOnRoute pins the current behavior of a
// dispatch.Dispatcher using HostHasSubdomain as the handler of a teapot route.
// It is a characterization test: it documents what already works so that
// later changes to the subdomain matching path cannot silently break it.
func TestSubdomainDispatcherMountedOnRoute(t *testing.T) {
	vhost := dispatch.New(func(b *dispatch.Builder) {
		b.When(dispatch.HostHasSubdomain("s3.example.com")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("subdomain"))
		})
		b.Default(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("path"))
		})
	})

	r := teapot.New()
	r.GET("/", vhost).Name("root")

	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{"subdomain host reaches subdomain handler", "mybucket.s3.example.com", "subdomain"},
		{"subdomain host with port reaches subdomain handler", "mybucket.s3.example.com:9000", "subdomain"},
		{"bare canonical domain falls through to default", "s3.example.com", "path"},
		{"unrelated host falls through to default", "example.org", "path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://placeholder/", nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			assert.Equal(t, tt.expected, rec.Body.String())
		})
	}
}
