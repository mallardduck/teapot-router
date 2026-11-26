package teapot

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDispatchHostHasSubdomain verifies Matchers.HostHasSubdomain inside a
// Dispatch group: subdomain requests reach the subdomain route, everything
// else falls through to the default route.
func TestDispatchHostHasSubdomain(t *testing.T) {
	r := New()

	r.Dispatch("GET", "/", func(d *DispatchBuilder, m Matchers) {
		d.DefaultFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("path"))
		})
		d.When(m.HostHasSubdomain("s3.example.com")).DoFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("subdomain"))
		})
	})

	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{"subdomain host matches", "mybucket.s3.example.com", "subdomain"},
		{"subdomain host with port matches", "mybucket.s3.example.com:9000", "subdomain"},
		{"bare canonical domain falls through", "s3.example.com", "path"},
		{"unrelated host falls through", "example.org", "path"},
		{"suffix-spoofing host falls through", "s3.example.com.evil.com", "path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://placeholder/", nil)
			req.Host = tt.host
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.expected, w.Body.String())
		})
	}
}

// TestDispatchHostHasSubdomainCombinedWithQuery verifies that HostHasSubdomain
// is ANDed with other matchers in the same When call.
func TestDispatchHostHasSubdomainCombinedWithQuery(t *testing.T) {
	r := New()

	r.Dispatch("GET", "/", func(d *DispatchBuilder, m Matchers) {
		d.DefaultFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("default"))
		})
		d.When(m.HostHasSubdomain("s3.example.com"), m.QueryExists("acl")).DoFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("subdomain-acl"))
		})
	})

	tests := []struct {
		name     string
		host     string
		url      string
		expected string
	}{
		{"subdomain host and query both match", "mybucket.s3.example.com", "http://placeholder/?acl", "subdomain-acl"},
		{"subdomain host without query falls through", "mybucket.s3.example.com", "http://placeholder/", "default"},
		{"query without subdomain host falls through", "s3.example.com", "http://placeholder/?acl", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			req.Host = tt.host
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.expected, w.Body.String())
		})
	}
}
