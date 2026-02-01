package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestResourceRESTful verifies standard REST resource scaffolding
func TestResourceRESTful(t *testing.T) {
	r := teapot.New()

	var calls []string

	r.Resource("photos", "/photos", "photo", teapot.ResourceHandlers{
		Index: func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "index")
			w.WriteHeader(200)
		},
		Create: func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "create")
			w.WriteHeader(200)
		},
		Store: func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "store")
			w.WriteHeader(201)
		},
		Show: func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "show")
			w.WriteHeader(200)
		},
		Edit: func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "edit")
			w.WriteHeader(200)
		},
		Update: func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "update")
			w.WriteHeader(200)
		},
		Destroy: func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "destroy")
			w.WriteHeader(204)
		},
		Head: func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "head")
			w.WriteHeader(200)
		},
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

		if w.Code != tt.expectedCode {
			t.Errorf("%s %s: expected code %d, got %d", tt.method, tt.path, tt.expectedCode, w.Code)
		}

		if len(calls) != 1 || calls[0] != tt.expectedCall {
			t.Errorf("%s %s: expected [%s], got %v", tt.method, tt.path, tt.expectedCall, calls)
		}
	}
}

// TestResourceS3Style verifies S3-style resource with PUT for creation
func TestResourceS3Style(t *testing.T) {
	r := teapot.New()

	var calls []string

	// S3-style bucket operations
	r.Resource("buckets", "/buckets", "bucket", teapot.ResourceHandlers{
		Index: func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "list-buckets")
			w.WriteHeader(200)
		},
		Store: func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "create-bucket")
			w.WriteHeader(200)
		},
		Show: func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "head-bucket")
			w.WriteHeader(200)
		},
		Destroy: func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "delete-bucket")
			w.WriteHeader(204)
		},
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

		if w.Code != tt.expectedCode {
			t.Errorf("%s %s: expected code %d, got %d", tt.method, tt.path, tt.expectedCode, w.Code)
		}

		if len(calls) != 1 || calls[0] != tt.expectedCall {
			t.Errorf("%s %s: expected [%s], got %v", tt.method, tt.path, tt.expectedCall, calls)
		}
	}
}

// TestResourcePartialHandlers verifies nil handlers are skipped
func TestResourcePartialHandlers(t *testing.T) {
	r := teapot.New()

	// Only provide Index and Show handlers
	r.Resource("users", "/users", "user", teapot.ResourceHandlers{
		Index: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		},
		Show: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		},
		// Store, Update, Destroy are nil - routes should not be registered
	})

	// These should work
	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("GET /users: expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/users/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("GET /users/123: expected 200, got %d", w.Code)
	}

	// These should 405 (path exists but method not allowed, no handlers registered)
	req = httptest.NewRequest("POST", "/users", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 405 {
		t.Errorf("POST /users: expected 405 Method Not Allowed (no Store handler), got %d", w.Code)
	}

	req = httptest.NewRequest("PUT", "/users/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 405 {
		t.Errorf("PUT /users/123: expected 405 Method Not Allowed (no Update handler), got %d", w.Code)
	}
}

// TestResourceNaming verifies route names are correctly set
func TestResourceNaming(t *testing.T) {
	r := teapot.New()

	r.Resource("posts", "/posts", "post", teapot.ResourceHandlers{
		Index: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		},
		Store: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(201)
		},
		Show: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		},
		Update: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		},
		Destroy: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(204)
		},
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
		if err != nil {
			t.Errorf("URL(%s): unexpected error: %v", tt.name, err)
			continue
		}
		if url != tt.expected {
			t.Errorf("URL(%s): expected %s, got %s", tt.name, tt.expected, url)
		}
	}
}

// TestResourceCustomMethods verifies custom StoreMethod and UpdateMethod
func TestResourceCustomMethods(t *testing.T) {
	r := teapot.New()

	var storeCalls, updateCalls int

	r.Resource("items", "/items", "item", teapot.ResourceHandlers{
		Store: func(w http.ResponseWriter, r *http.Request) {
			storeCalls++
			w.WriteHeader(201)
		},
		Update: func(w http.ResponseWriter, r *http.Request) {
			updateCalls++
			w.WriteHeader(200)
		},
		StoreMethod:  teapot.PUT,  // Custom: PUT instead of POST
		UpdateMethod: teapot.POST, // Custom: POST instead of PUT
	})

	// Store should use PUT
	req := httptest.NewRequest("PUT", "/items", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("PUT /items: expected 201, got %d", w.Code)
	}
	if storeCalls != 1 {
		t.Errorf("PUT /items: expected 1 store call, got %d", storeCalls)
	}

	// POST /items should 405 (PUT is registered, POST is not)
	req = httptest.NewRequest("POST", "/items", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 405 {
		t.Errorf("POST /items: expected 405 Method Not Allowed, got %d", w.Code)
	}

	// Update should use POST
	req = httptest.NewRequest("POST", "/items/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("POST /items/123: expected 200, got %d", w.Code)
	}
	if updateCalls != 1 {
		t.Errorf("POST /items/123: expected 1 update call, got %d", updateCalls)
	}

	// PUT /items/123 should 405 (POST is registered, PUT is not)
	req = httptest.NewRequest("PUT", "/items/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 405 {
		t.Errorf("PUT /items/123: expected 405 Method Not Allowed, got %d", w.Code)
	}
}

// TestResourceURLParams verifies URL parameters work correctly
func TestResourceURLParams(t *testing.T) {
	r := teapot.New()

	var capturedParam string

	r.Resource("products", "/products", "productId", teapot.ResourceHandlers{
		Show: func(w http.ResponseWriter, r *http.Request) {
			capturedParam = teapot.URLParam(r, "productId")
			w.WriteHeader(200)
		},
	})

	req := httptest.NewRequest("GET", "/products/abc-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if capturedParam != "abc-123" {
		t.Errorf("expected productId='abc-123', got %q", capturedParam)
	}
}

// TestResourceWithGroup verifies Resource works inside groups
func TestResourceWithGroup(t *testing.T) {
	r := teapot.New()

	r.NamedGroup("/api", "api", func(r *teapot.Router) {
		r.Resource("users", "/users", "userId", teapot.ResourceHandlers{
			Index: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			},
			Show: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			},
		})
	})

	// Test routes have correct paths
	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("GET /api/users: expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/users/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("GET /api/users/123: expected 200, got %d", w.Code)
	}

	// Test routes have correct names
	url, err := r.URL("api.users.index")
	if err != nil {
		t.Errorf("URL(api.users.index): unexpected error: %v", err)
	}
	if url != "/api/users" {
		t.Errorf("URL(api.users.index): expected /api/users, got %s", url)
	}

	url, err = r.URL("api.users.show", "userId", "123")
	if err != nil {
		t.Errorf("URL(api.users.show): unexpected error: %v", err)
	}
	if url != "/api/users/123" {
		t.Errorf("URL(api.users.show): expected /api/users/123, got %s", url)
	}
}
