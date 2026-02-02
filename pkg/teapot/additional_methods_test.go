package teapot_test

import (
	"net/http"
	"testing"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// Test: PATCH and OPTIONS methods
func TestPatchAndOptions(t *testing.T) {
	r := teapot.New()

	r.PATCH("/resource", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PATCH"))
	})
	r.OPTIONS("/resource", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OPTIONS"))
	})

	assertEqual(t, request(t, r, "PATCH", "/resource").Body.String(), "PATCH")
	assertEqual(t, request(t, r, "OPTIONS", "/resource").Body.String(), "OPTIONS")
}

// Test: QueryPOST
func TestQueryPOST(t *testing.T) {
	r := teapot.New()

	r.QueryPOST("/upload", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("UPLOAD"))
	}).Query("uploads")

	r.QueryPOST("/upload", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("CREATE"))
	})

	assertEqual(t, request(t, r, "POST", "/upload?uploads").Body.String(), "UPLOAD")
	assertEqual(t, request(t, r, "POST", "/upload").Body.String(), "CREATE")
}

// Test: QueryPUT
func TestQueryPUT(t *testing.T) {
	r := teapot.New()

	r.QueryPUT("/object", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ACL"))
	}).Query("acl")

	r.QueryPUT("/object", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PUT"))
	})

	assertEqual(t, request(t, r, "PUT", "/object?acl").Body.String(), "ACL")
	assertEqual(t, request(t, r, "PUT", "/object").Body.String(), "PUT")
}

// Test: QueryDELETE
func TestQueryDELETE(t *testing.T) {
	r := teapot.New()

	r.QueryDELETE("/object", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("DELETE_ALL"))
	}).Query("deleteAll")

	r.QueryDELETE("/object", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("DELETE"))
	})

	assertEqual(t, request(t, r, "DELETE", "/object?deleteAll").Body.String(), "DELETE_ALL")
	assertEqual(t, request(t, r, "DELETE", "/object").Body.String(), "DELETE")
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
	assertEqual(t, w.Code, 200)
	assertEqual(t, w.Header().Get("X-Custom"), "metadata")

	w = request(t, r, "HEAD", "/object")
	assertEqual(t, w.Code, 200)
}

// Test: QueryPATCH
func TestQueryPATCH(t *testing.T) {
	r := teapot.New()

	r.QueryPATCH("/resource", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PARTIAL"))
	}).Query("partial")

	r.QueryPATCH("/resource", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PATCH"))
	})

	assertEqual(t, request(t, r, "PATCH", "/resource?partial").Body.String(), "PARTIAL")
	assertEqual(t, request(t, r, "PATCH", "/resource").Body.String(), "PATCH")
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
	assertEqual(t, w.Header().Get("Allow"), "GET, POST, CORS")

	w = request(t, r, "OPTIONS", "/resource")
	assertEqual(t, w.Header().Get("Allow"), "GET, POST")
}

// Test: Finalize and IsFinalized
func TestFinalizeAndIsFinalized(t *testing.T) {
	r := teapot.New()

	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Initially not finalized
	if r.IsFinalized() {
		t.Error("router should not be finalized initially")
	}

	// Finalize the router
	r.Finalize()

	// Should now be finalized
	if !r.IsFinalized() {
		t.Error("router should be finalized after calling Finalize()")
	}

	// Routes should still work after finalization
	assertEqual(t, request(t, r, "GET", "/test").Body.String(), "OK")
}

// Test: URL edge cases
func TestURLEdgeCases(t *testing.T) {
	r := teapot.New()

	r.GET("/users/{id}/posts/{postId}", func(w http.ResponseWriter, r *http.Request) {}).
		Name("user.posts.show")

	// Test with no parameters
	_, err := r.URL("user.posts.show")
	if err == nil {
		t.Error("expected error when no parameters provided")
	}

	// Test with odd number of parameters
	_, err = r.URL("user.posts.show", "id")
	if err == nil {
		t.Error("expected error when odd number of parameters provided")
	}

	// Test with correct parameters
	url, err := r.URL("user.posts.show", "id", "123", "postId", "456")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	assertEqual(t, url, "/users/123/posts/456")
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
		w.Write([]byte("FULL_ADMIN"))
	}).Query("admin").QueryValue("type", "full")

	r.QueryGET("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ADMIN"))
	}).Query("admin")

	r.QueryGET("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("FULL"))
	}).QueryValue("type", "full")

	r.QueryGET("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("DEFAULT"))
	})

	assertEqual(t, request(t, r, "GET", "/search?admin&type=full").Body.String(), "FULL_ADMIN")
	assertEqual(t, request(t, r, "GET", "/search?admin").Body.String(), "ADMIN")
	assertEqual(t, request(t, r, "GET", "/search?type=full").Body.String(), "FULL")
	assertEqual(t, request(t, r, "GET", "/search").Body.String(), "DEFAULT")
}
