package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// fullController implements every resource action. Each action records its
// name in calls and writes a status code.
type fullController struct {
	calls []string
}

func (c *fullController) hit(name string, code int, w http.ResponseWriter) {
	c.calls = append(c.calls, name)
	w.WriteHeader(code)
}

func (c *fullController) Index(w http.ResponseWriter, _ *http.Request)   { c.hit("index", 200, w) }
func (c *fullController) Create(w http.ResponseWriter, _ *http.Request)  { c.hit("create", 200, w) }
func (c *fullController) Store(w http.ResponseWriter, _ *http.Request)   { c.hit("store", 201, w) }
func (c *fullController) Show(w http.ResponseWriter, _ *http.Request)    { c.hit("show", 200, w) }
func (c *fullController) Edit(w http.ResponseWriter, _ *http.Request)    { c.hit("edit", 200, w) }
func (c *fullController) Update(w http.ResponseWriter, _ *http.Request)  { c.hit("update", 200, w) }
func (c *fullController) Destroy(w http.ResponseWriter, _ *http.Request) { c.hit("destroy", 204, w) }

// readOnlyController implements only Index and Show.
type readOnlyController struct{}

func (readOnlyController) Index(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }
func (readOnlyController) Show(w http.ResponseWriter, _ *http.Request)  { w.WriteHeader(200) }

// showController implements only Show, and records the URL parameter it saw.
type showController struct {
	param string
	key   string
}

func (c *showController) Show(w http.ResponseWriter, r *http.Request) {
	c.param = teapot.URLParam(r, c.key)
	w.WriteHeader(200)
}

// TestResourceRESTful pins the default route table for a full controller.
// HEAD is not a per-resource route; it reaches the Show action by fallback.
func TestResourceRESTful(t *testing.T) {
	r := teapot.New()
	ctrl := &fullController{}
	r.Resource("photos", ctrl)

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
		{"PATCH", "/photos/123", "update", 200},
		{"DELETE", "/photos/123", "destroy", 204},
		{"HEAD", "/photos/123", "show", 200},
	}

	for _, tt := range tests {
		ctrl.calls = nil
		req := httptest.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, tt.expectedCode, w.Code, "%s %s", tt.method, tt.path)
		assert.Equal(t, []string{tt.expectedCall}, ctrl.calls, "%s %s", tt.method, tt.path)
	}
}

// bucketController mirrors an S3-style bucket API.
type bucketController struct {
	calls []string
}

func (c *bucketController) Index(w http.ResponseWriter, _ *http.Request) {
	c.calls = append(c.calls, "list-buckets")
	w.WriteHeader(200)
}

func (c *bucketController) Store(w http.ResponseWriter, _ *http.Request) {
	c.calls = append(c.calls, "create-bucket")
	w.WriteHeader(200)
}

func (c *bucketController) Show(w http.ResponseWriter, _ *http.Request) {
	c.calls = append(c.calls, "head-bucket")
	w.WriteHeader(200)
}

func (c *bucketController) Destroy(w http.ResponseWriter, _ *http.Request) {
	c.calls = append(c.calls, "delete-bucket")
	w.WriteHeader(204)
}

// TestResourceS3Style verifies creation with PUT via StoreVerb.
func TestResourceS3Style(t *testing.T) {
	r := teapot.New()
	ctrl := &bucketController{}
	r.Resource("buckets", ctrl, teapot.StoreVerb(teapot.PUT))

	tests := []struct {
		method       string
		path         string
		expectedCall string
		expectedCode int
	}{
		{"GET", "/buckets", "list-buckets", 200},
		{"PUT", "/buckets", "create-bucket", 200},
		{"GET", "/buckets/my-bucket", "head-bucket", 200},
		{"DELETE", "/buckets/my-bucket", "delete-bucket", 204},
	}

	for _, tt := range tests {
		ctrl.calls = nil
		req := httptest.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, tt.expectedCode, w.Code, "%s %s", tt.method, tt.path)
		assert.Equal(t, []string{tt.expectedCall}, ctrl.calls, "%s %s", tt.method, tt.path)
	}
}

// TestResourcePartialHandlers verifies actions the controller lacks are not registered.
func TestResourcePartialHandlers(t *testing.T) {
	r := teapot.New()
	r.Resource("users", readOnlyController{})

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "GET /users")

	req = httptest.NewRequest("GET", "/users/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "GET /users/123")

	req = httptest.NewRequest("POST", "/users", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 405, w.Code, "POST /users - no Store action")

	req = httptest.NewRequest("PUT", "/users/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 405, w.Code, "PUT /users/123 - no Update action")
}

// TestResourceNaming verifies route names and URL generation, including the
// default singular parameter ("posts" gives {post}).
func TestResourceNaming(t *testing.T) {
	r := teapot.New()
	r.Resource("posts", &fullController{})

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

// TestResourceURLParams verifies a custom parameter name from Param.
func TestResourceURLParams(t *testing.T) {
	r := teapot.New()
	ctrl := &showController{key: "productId"}
	r.Resource("products", ctrl, teapot.Param("productId"))

	req := httptest.NewRequest("GET", "/products/abc-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "abc-123", ctrl.param)
}

// TestResourceWithGroup verifies Resource works inside a named group.
func TestResourceWithGroup(t *testing.T) {
	r := teapot.New()

	r.NamedGroup("/api", "api", func(r *teapot.Router) {
		r.Resource("users", readOnlyController{})
	})

	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "GET /api/users")

	req = httptest.NewRequest("GET", "/api/users/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "GET /api/users/123")

	url, err := r.URL("api.users.index")
	assert.NoError(t, err, "URL(api.users.index)")
	assert.Equal(t, "/api/users", url, "URL(api.users.index)")

	url, err = r.URL("api.users.show", "user", "123")
	assert.NoError(t, err, "URL(api.users.show)")
	assert.Equal(t, "/api/users/123", url, "URL(api.users.show)")
}

// TestResourceOnlyAndExcept verifies Only and Except narrow the registered actions.
func TestResourceOnlyAndExcept(t *testing.T) {
	r := teapot.New()
	r.Resource("photos", &fullController{}, teapot.Only("index", "show"))
	r.Resource("videos", &fullController{}, teapot.Except("destroy"))

	_, err := r.URL("photos.index")
	assert.NoError(t, err)
	_, err = r.URL("photos.destroy", "photo", "1")
	assert.Error(t, err, "photos.destroy should not be registered with Only")

	req := httptest.NewRequest("DELETE", "/videos/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 405, w.Code, "DELETE /videos/1 should be excluded by Except")
}

// TestResourceStoreVerbPUT verifies StoreVerb changes only the store method.
func TestResourceStoreVerbPUT(t *testing.T) {
	r := teapot.New()
	ctrl := &fullController{}
	r.Resource("items", ctrl, teapot.StoreVerb(teapot.PUT))

	req := httptest.NewRequest("PUT", "/items", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code, "PUT /items")

	req = httptest.NewRequest("POST", "/items", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 405, w.Code, "POST /items should be unregistered")

	// Update keeps both PUT and PATCH regardless of StoreVerb.
	req = httptest.NewRequest("PATCH", "/items/123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "PATCH /items/123")
}

// TestResourceMiddlewareOnlyAffectsItsRoutes verifies Middleware scope.
func TestResourceMiddlewareOnlyAffectsItsRoutes(t *testing.T) {
	r := teapot.New()
	r.Resource("photos", readOnlyController{}, teapot.Middleware(tagMiddleware("photos")))
	r.Resource("videos", readOnlyController{})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/photos", nil))
	assert.Equal(t, "photos", rec.Header().Get("X-Tag"))

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/videos", nil))
	assert.Empty(t, rec.Header().Values("X-Tag"))
}

// TestResourceCustomNames verifies Names overrides individual action names.
func TestResourceCustomNames(t *testing.T) {
	r := teapot.New()
	r.Resource("photos", readOnlyController{}, teapot.Names(map[string]string{"index": "list"}))

	url, err := r.URL("photos.list")
	assert.NoError(t, err)
	assert.Equal(t, "/photos", url)
}

// TestResourcePathOverride verifies Path replaces the default "/" + name path.
func TestResourcePathOverride(t *testing.T) {
	r := teapot.New()
	r.Resource("photos", readOnlyController{}, teapot.Path("/pictures"))

	req := httptest.NewRequest("GET", "/pictures", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "GET /pictures")

	url, err := r.URL("photos.index")
	assert.NoError(t, err)
	assert.Equal(t, "/pictures", url)
}
