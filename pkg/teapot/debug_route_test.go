package teapot

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManualDebugRouteListsItself pins the manual pattern that RegisterDebugRoute
// will wrap: GET + NewListRoutesHandler + Name. The debug route is part of
// Routes(), so its own name appears in the JSON listing.
func TestManualDebugRouteListsItself(t *testing.T) {
	r := New()
	r.GET("/.internal/routes", NewListRoutesHandler(r, nil)).Name("debug.routes")

	req := httptest.NewRequest("GET", "/.internal/routes", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"debug.routes"`)
}

// RegisterDebugRoute should behave exactly like the manual pattern above.
func TestRegisterDebugRouteServesJSONListing(t *testing.T) {
	r := New()
	r.RegisterDebugRoute("/.internal/routes", "debug.routes")

	req := httptest.NewRequest("GET", "/.internal/routes", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"debug.routes"`)
}

func TestRegisterDebugRouteServesHTMLListing(t *testing.T) {
	r := New()
	r.RegisterDebugRoute("/.internal/routes", "debug.routes")

	req := httptest.NewRequest("GET", "/.internal/routes", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
}

func TestRegisterDebugRouteIsNamed(t *testing.T) {
	r := New()
	r.RegisterDebugRoute("/.internal/routes", "debug.routes")

	path, err := r.URL("debug.routes")
	assert.NoError(t, err)
	assert.Equal(t, "/.internal/routes", path)
}

func TestRegisterDebugRouteReturnsBuilder(t *testing.T) {
	r := New()
	rb := r.RegisterDebugRoute("/.internal/routes", "debug.routes")
	assert.NotNil(t, rb)
}
