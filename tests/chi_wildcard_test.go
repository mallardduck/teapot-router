package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// Test Chi's wildcard syntax directly to understand how it works
func TestChiWildcardSyntax(t *testing.T) {
	tests := []struct {
		pattern  string
		url      string
		paramKey string
		expected string
	}{
		{"/files/*", "/files/documents/report.pdf", "*", "documents/report.pdf"},
		// Note: {path}* syntax doesn't work in Chi - use /* instead
		{"/files/{path}/*", "/files/documents/report.pdf", "*", "report.pdf"},
		{"/{bucket}/*", "/mybucket/path/to/file.txt", "*", "path/to/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			r := chi.NewRouter()
			var captured string

			r.Get(tt.pattern, func(w http.ResponseWriter, r *http.Request) {
				captured = chi.URLParam(r, tt.paramKey)
				_, _ = w.Write([]byte(captured))
			})

			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expected, captured, "pattern %s", tt.pattern)
		})
	}
}
