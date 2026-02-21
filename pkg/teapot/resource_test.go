package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestResourceRESTful verifies standard REST resource scaffolding
func TestResourceRESTful(t *testing.T) {
	r := teapot.New()

	var calls []string

	r.Resource("photos", "/photos", "photo", teapot.ResourceHandlers{
		Index: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "index")
			w.WriteHeader(200)
		}),
		Create: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "create")
			w.WriteHeader(200)
		}),
		Store: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "store")
			w.WriteHeader(201)
		}),
		Show: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "show")
			w.WriteHeader(200)
		}),
		Edit: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "edit")
			w.WriteHeader(200)
		}),
		Update: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "update")
			w.WriteHeader(200)
		}),
		Destroy: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "destroy")
			w.WriteHeader(204)
		}),
		Head: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "head")
			w.WriteHeader(200)
		}),
	})

	tests := []struct {
		method       string
		path         string
		expectedCall string
		expectedCode int
	}{
		{"GET", "/photos", "index", 200},
		{"GET", "/photos/create", "create", 200},
		{"POST", "/photos", "store", 201},
		{"GET", "/photos/123", "show", 200},
		{"GET", "/photos/123/edit", "edit", 200},
		{"PUT", "/photos/123", "update", 200},
		{"DELETE", "/photos/123", "destroy", 204},
		{"HEAD", "/photos/123", "head", 200},
	}

	for _, tt := range tests {
		calls = nil
		req := httptest.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, tt.expectedCode, w.Code, "%s %s", tt.method, tt.path)
		assert.Equal(t, []string{tt.expectedCall}, calls, "%s %s", tt.method, tt.path)
	}
}

// TestResourceS3Style verifies S3-style resource with PUT for creation
func TestResourceS3Style(t *testing.T) {
	r := teapot.New()

	var calls []string

	// S3-style bucket operations
	r.Resource("buckets", "/buckets", "bucket", teapot.ResourceHandlers{
		Index: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "list-buckets")
			w.WriteHeader(200)
		}),
		Store: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "create-bucket")
			w.WriteHeader(200)
		}),
		Show: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "head-bucket")
			w.WriteHeader(200)
		}),
		Destroy: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "delete-bucket")
			w.WriteHeader(204)
		}),
		StoreMethod: teapot.PUT, // S3 uses PUT to create buckets
	})

	tests := []struct {
		method       string
		path         string
		expectedCall string
		expectedCode int
	}{
		{"GET", "/buckets", "list-buckets", 200},
		{"PUT", "/buckets", "create-bucket", 200}, // S3-style: PUT instead of POST
		{"GET", "/buckets/my-bucket", "head-bucket", 200},
		{"DELETE", "/buckets/my-bucket", "delete-bucket", 204},
	}

	for _, tt := range tests {
		calls = nil
		req := httptest.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, tt.expectedCode, w.Code, "%s %s", tt.method, tt.path)
		assert.Equal(t, []string{tt.expectedCall}, calls, "%s %s", tt.method, tt.path)
	}
}

// TestResourcePartialHandlers verifies nil handlers are skipped
func TestResourcePartialHandlers(t *testing.T) {
	r := teapot.New()

	// Only provide Index and Show handlers
	r.Resource("users", "/users", "user", teapot.ResourceHandlers{
		Index: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}),
		Show: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}),
		// Store, Update, Destroy are nil - routes should not be registered
	})

	// These should work
	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "GET /users")

	req = httptest.NewRequest("GET", "/users/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "GET /users/123")

	// These should 405 (path exists but method not allowed, no handlers registered)
	req = httptest.NewRequest("POST", "/users", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 405, w.Code, "POST /users - expected 405 Method Not Allowed (no Store handler)")

	req = httptest.NewRequest("PUT", "/users/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 405, w.Code, "PUT /users/123 - expected 405 Method Not Allowed (no Update handler)")
}

// TestResourceNaming verifies route names are correctly set
func TestResourceNaming(t *testing.T) {
	r := teapot.New()

	r.Resource("posts", "/posts", "post", teapot.ResourceHandlers{
		Index: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}),
		Store: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(201)
		}),
		Show: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}),
		Update: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}),
		Destroy: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(204)
		}),
	})

	// Test URL generation
	tests := []struct {
		name     string
		params   []string
		expected string
	}{
		{"posts.index", nil, "/posts"},
		{"posts.store", nil, "/posts"},
		{"posts.show", []string{"post", "123"}, "/posts/123"},
		{"posts.update", []string{"post", "123"}, "/posts/123"},
		{"posts.destroy", []string{"post", "123"}, "/posts/123"},
	}

	for _, tt := range tests {
		url, err := r.URL(tt.name, tt.params...)
		assert.NoError(t, err, "URL(%s)", tt.name)
		assert.Equal(t, tt.expected, url, "URL(%s)", tt.name)
	}
}

// TestResourceCustomMethods verifies custom StoreMethod and UpdateMethod
func TestResourceCustomMethods(t *testing.T) {
	r := teapot.New()

	var storeCalls, updateCalls int

	r.Resource("items", "/items", "item", teapot.ResourceHandlers{
		Store: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			storeCalls++
			w.WriteHeader(201)
		}),
		Update: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			updateCalls++
			w.WriteHeader(200)
		}),
		StoreMethod:  teapot.PUT,  // Custom: PUT instead of POST
		UpdateMethod: teapot.POST, // Custom: POST instead of PUT
	})

	// Store should use PUT
	req := httptest.NewRequest("PUT", "/items", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code, "PUT /items")
	assert.Equal(t, 1, storeCalls, "PUT /items store calls")

	// POST /items should 405 (PUT is registered, POST is not)
	req = httptest.NewRequest("POST", "/items", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 405, w.Code, "POST /items - expected 405 Method Not Allowed")

	// Update should use POST
	req = httptest.NewRequest("POST", "/items/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "POST /items/123")
	assert.Equal(t, 1, updateCalls, "POST /items/123 update calls")

	// PUT /items/123 should 405 (POST is registered, PUT is not)
	req = httptest.NewRequest("PUT", "/items/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 405, w.Code, "PUT /items/123 - expected 405 Method Not Allowed")
}

// TestResourceURLParams verifies URL parameters work correctly
func TestResourceURLParams(t *testing.T) {
	r := teapot.New()

	var capturedParam string

	r.Resource("products", "/products", "productId", teapot.ResourceHandlers{
		Show: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedParam = teapot.URLParam(r, "productId")
			w.WriteHeader(200)
		}),
	})

	req := httptest.NewRequest("GET", "/products/abc-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "abc-123", capturedParam)
}

// TestResourceWithGroup verifies Resource works inside groups
func TestResourceWithGroup(t *testing.T) {
	r := teapot.New()

	r.NamedGroup("/api", "api", func(r *teapot.Router) {
		r.Resource("users", "/users", "userId", teapot.ResourceHandlers{
			Index: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			}),
			Show: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			}),
		})
	})

	// Test routes have correct paths
	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "GET /api/users")

	req = httptest.NewRequest("GET", "/api/users/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "GET /api/users/123")

	// Test routes have correct names
	url, err := r.URL("api.users.index")
	assert.NoError(t, err, "URL(api.users.index)")
	assert.Equal(t, "/api/users", url, "URL(api.users.index)")

	url, err = r.URL("api.users.show", "userId", "123")
	assert.NoError(t, err, "URL(api.users.show)")
	assert.Equal(t, "/api/users/123", url, "URL(api.users.show)")
}
