package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mallardduck/teapot-router/internal/testutil"
	"github.com/mallardduck/teapot-router/pkg/teapot"
)

func TestDispatchNameConflictAcrossPathsPanics(t *testing.T) {
	r := teapot.New()
	r.GET("/a", http.HandlerFunc(testutil.NoopResponse)).Name("shared")

	msg := testutil.CapturePanic(func() {
		r.Dispatch("POST", "/b", func(d *teapot.DispatchBuilder, m teapot.Matchers) {
			d.FuncDefault(testutil.NoopResponse).Name("shared")
		})
	})
	require.NotEmpty(t, msg, "expected panic")
	assert.Contains(t, msg, "used with different paths")
}

func TestDispatchNameSharedAcrossMethodsSamePathAllowed(t *testing.T) {
	r := teapot.New()
	r.GET("/a", http.HandlerFunc(testutil.NoopResponse)).Name("shared")

	assert.NotPanics(t, func() {
		r.Dispatch("POST", "/a", func(d *teapot.DispatchBuilder, m teapot.Matchers) {
			d.FuncDefault(testutil.NoopResponse).Name("shared")
		})
	})
}

func TestMountedQueryRoutePropagatesToParent(t *testing.T) {
	sub := teapot.New()
	sub.QueryGET("/bucket", http.HandlerFunc(testutil.NoopResponse)).Query("acl").Name("acl")

	r := teapot.New()
	r.Mount("/api", sub)

	url, err := r.URL("acl")
	require.NoError(t, err)
	assert.Equal(t, "/api/bucket", url)
}

func TestMountNamedWithPlainHandlerServesRequests(t *testing.T) {
	r := teapot.New()
	r.MountNamed("/static", "static", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/static/file.txt", nil))
	assert.Equal(t, http.StatusTeapot, w.Code)
}

func TestNoopHandlerDoesNothing(t *testing.T) {
	w := httptest.NewRecorder()
	teapot.NoopHandler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestRoutesJSONFromRouteListingHandler(t *testing.T) {
	r := teapot.New()
	r.GET("/x", teapot.NoopHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/json")
	teapot.NewListRoutesHandler(r, nil)(w, req)

	assert.True(t, strings.Contains(w.Body.String(), `"count":1`))
}
