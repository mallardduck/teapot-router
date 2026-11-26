package teapot

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoutesHTMLPinsCurrentTable characterizes today's HTML listing: a table
// with one row per route. Sorting work must not change this.
func TestRoutesHTMLPinsCurrentTable(t *testing.T) {
	r := New()
	r.GET("/users", NoopHandler).Name("users.index")

	req := httptest.NewRequest("GET", "/routes", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	NewListRoutesHandler(r, nil).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "<table>")
	assert.Contains(t, rec.Body.String(), `<td class="name">users.index</td>`)
}
