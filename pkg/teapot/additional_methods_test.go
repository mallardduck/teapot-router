package teapot_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// Test: PATCH and OPTIONS methods
func TestPatchAndOptions(t *testing.T) {
	r := teapot.New()

	r.PATCH("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("PATCH"))
	})
	r.OPTIONS("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OPTIONS"))
	})

	assert.Equal(t, "PATCH", request(t, r, "PATCH", "/resource").Body.String())
	assert.Equal(t, "OPTIONS", request(t, r, "OPTIONS", "/resource").Body.String())
}

// Test: QueryPOST
func TestQueryPOST(t *testing.T) {
	r := teapot.New()

	r.QueryPOST("/upload", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("UPLOAD"))
	}).Query("uploads")

	r.QueryPOST("/upload", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("CREATE"))
	})

	assert.Equal(t, "UPLOAD", request(t, r, "POST", "/upload?uploads").Body.String())
	assert.Equal(t, "CREATE", request(t, r, "POST", "/upload").Body.String())
}

// Test: QueryPUT
func TestQueryPUT(t *testing.T) {
	r := teapot.New()

	r.QueryPUT("/object", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ACL"))
	}).Query("acl")

	r.QueryPUT("/object", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("PUT"))
	})

	assert.Equal(t, "ACL", request(t, r, "PUT", "/object?acl").Body.String())
	assert.Equal(t, "PUT", request(t, r, "PUT", "/object").Body.String())
}

// Test: QueryDELETE
func TestQueryDELETE(t *testing.T) {
	r := teapot.New()

	r.QueryDELETE("/object", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("DELETE_ALL"))
	}).Query("deleteAll")

	r.QueryDELETE("/object", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("DELETE"))
	})

	assert.Equal(t, "DELETE_ALL", request(t, r, "DELETE", "/object?deleteAll").Body.String())
	assert.Equal(t, "DELETE", request(t, r, "DELETE", "/object").Body.String())
}

// Test: QueryHEAD
func TestQueryHEAD(t *testing.T) {
	r := teapot.New()

	r.QueryHEAD("/object", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "metadata")
		w.WriteHeader(200)
	}).Query("metadata")

	r.QueryHEAD("/object", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	w := request(t, r, "HEAD", "/object?metadata")
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "metadata", w.Header().Get("X-Custom"))

	w = request(t, r, "HEAD", "/object")
	assert.Equal(t, 200, w.Code)
}

// Test: QueryPATCH
func TestQueryPATCH(t *testing.T) {
	r := teapot.New()

	r.QueryPATCH("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("PARTIAL"))
	}).Query("partial")

	r.QueryPATCH("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("PATCH"))
	})

	assert.Equal(t, "PARTIAL", request(t, r, "PATCH", "/resource?partial").Body.String())
	assert.Equal(t, "PATCH", request(t, r, "PATCH", "/resource").Body.String())
}

// Test: QueryOPTIONS
func TestQueryOPTIONS(t *testing.T) {
	r := teapot.New()

	r.QueryOPTIONS("/resource", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "GET, POST, CORS")
		w.WriteHeader(200)
	}).Query("cors")

	r.QueryOPTIONS("/resource", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "GET, POST")
		w.WriteHeader(200)
	})

	w := request(t, r, "OPTIONS", "/resource?cors")
	assert.Equal(t, "GET, POST, CORS", w.Header().Get("Allow"))

	w = request(t, r, "OPTIONS", "/resource")
	assert.Equal(t, "GET, POST", w.Header().Get("Allow"))
}

// Test: Finalize and IsFinalized
func TestFinalizeAndIsFinalized(t *testing.T) {
	r := teapot.New()

	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	// Initially not finalized
	assert.False(t, r.IsFinalized(), "router should not be finalized initially")

	// Finalize the router
	r.Finalize()

	// Should now be finalized
	assert.True(t, r.IsFinalized(), "router should be finalized after calling Finalize()")

	// Routes should still work after finalization
	assert.Equal(t, "OK", request(t, r, "GET", "/test").Body.String())
}

// Test: URL edge cases
func TestURLEdgeCases(t *testing.T) {
	r := teapot.New()

	r.GET("/users/{id}/posts/{postId}", func(w http.ResponseWriter, r *http.Request) {}).
		Name("user.posts.show")

	// Test with no parameters
	_, err := r.URL("user.posts.show")
	assert.Error(t, err, "expected error when no parameters provided")

	// Test with odd number of parameters
	_, err = r.URL("user.posts.show", "id")
	assert.Error(t, err, "expected error when odd number of parameters provided")

	// Test with correct parameters
	url, err := r.URL("user.posts.show", "id", "123", "postId", "456")
	assert.NoError(t, err)
	assert.Equal(t, "/users/123/posts/456", url)
}

// Test: MustURL panics
func TestMustURLPanic(t *testing.T) {
	r := teapot.New()

	defer func() {
		if rec := recover(); rec == nil {
			t.Error("expected MustURL to panic for nonexistent route")
		}
	}()

	r.MustURL("nonexistent.route")
}

// Test: Query and QueryValue chaining
func TestQueryChaining(t *testing.T) {
	r := teapot.New()

	r.QueryGET("/search", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("FULL_ADMIN"))
	}).Query("admin").QueryValue("type", "full")

	r.QueryGET("/search", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ADMIN"))
	}).Query("admin")

	r.QueryGET("/search", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("FULL"))
	}).QueryValue("type", "full")

	r.QueryGET("/search", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("DEFAULT"))
	})

	assert.Equal(t, "FULL_ADMIN", request(t, r, "GET", "/search?admin&type=full").Body.String())
	assert.Equal(t, "ADMIN", request(t, r, "GET", "/search?admin").Body.String())
	assert.Equal(t, "FULL", request(t, r, "GET", "/search?type=full").Body.String())
	assert.Equal(t, "DEFAULT", request(t, r, "GET", "/search").Body.String())
}
