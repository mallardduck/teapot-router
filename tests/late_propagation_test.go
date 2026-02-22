package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mallardduck/teapot-router/internal/testutil"
	"github.com/mallardduck/teapot-router/pkg/teapot"
)

func TestLatePropagation(t *testing.T) {
	// 1. Create a standalone sub-router
	sub := teapot.New()
	sub.GET("/hello", http.HandlerFunc(testutil.StringResponseWriterBuilder("hello from sub"))).Name("sub.hello")

	// 2. Create main router
	r := teapot.New()
	r.GET("/main", http.HandlerFunc(testutil.StringResponseWriterBuilder("hello from main"))).Name("main.index")

	// 3. Mount sub-router later
	r.Mount("/api", sub)

	// 4. Verify route propagation (unified listing/URL generation)
	routes := r.Routes()
	foundSub := false
	for _, info := range routes {
		if info.Pattern == "/api/hello" {
			foundSub = true
			break
		}
	}
	assert.True(t, foundSub, "Expected to find propagated route /api/hello in main router")

	// 5. Verify URL generation
	url, err := r.URL("sub.hello")
	require.NoError(t, err, "Failed to generate URL for propagated route")
	assert.Equal(t, "/api/hello", url)

	// 6. Verify functional routing
	ts := httptest.NewServer(r)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/api/hello")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode, "Expected status OK, got %d", res.StatusCode)
}

func TestLatePropagation_Homing(t *testing.T) {
	// This test checks if adding routes to the sub-router AFTER it's mounted
	// also propagates them to the parent (Homing in as if live propagation).

	r := teapot.New()
	sub := teapot.New()

	r.Mount("/v1", sub)

	// Add route to sub AFTER mounting
	sub.GET("/world", http.HandlerFunc(testutil.StringResponseWriterBuilder("world"))).Name("sub.world")

	// Verify propagation
	url, err := r.URL("sub.world")
	require.NoError(t, err, "Failed to generate URL for late-added route")
	assert.Equal(t, "/v1/world", url)
}

func TestLatePropagation_DeepHoming(t *testing.T) {
	r1 := teapot.New()
	r2 := teapot.New()
	r3 := teapot.New()

	r1.Mount("/api", r2)
	r2.Mount("/v1", r3)

	r3.GET("/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).Name("users.index")

	url, err := r1.URL("users.index")
	require.NoError(t, err, "Failed to generate URL for deep homing")
	assert.Equal(t, "/api/v1/users", url)
}
