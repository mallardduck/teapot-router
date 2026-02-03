package teapot_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestNewListRoutesHandler verifies the HTTP routes handler returns JSON
func TestNewListRoutesHandler(t *testing.T) {
	r := teapot.New()

	// Register some routes
	r.GET("/users", dummyHandler).Name("users.index")
	r.POST("/users", dummyHandler).Name("users.store")
	r.GET("/users/{id}", dummyHandler).Name("users.show").Action("s3:GetObject")
	r.DELETE("/users/{id}", dummyHandler).Name("users.destroy")

	// Get the handler (no filter - show all)
	handler := teapot.NewListRoutesHandler(r, nil)

	// Test JSON response (with Accept header)
	req := httptest.NewRequest("GET", "/.internal/routes", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Errorf("expected JSON content type, got %s", w.Header().Get("Content-Type"))
	}

	// Parse response
	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	// Check count
	count, ok := response["count"].(float64)
	if !ok || count != 4 {
		t.Errorf("expected count=4, got %v", response["count"])
	}

	// Check routes exist
	routes, ok := response["routes"].([]any)
	if !ok || len(routes) != 4 {
		t.Errorf("expected 4 routes, got %v", response["routes"])
	}
}

// TestNewListRoutesHandlerHTML verifies HTML output for browsers
func TestNewListRoutesHandlerHTML(t *testing.T) {
	r := teapot.New()

	r.GET("/users", dummyHandler).Name("users.index")
	r.GET("/posts", dummyHandler).Name("posts.index")

	handler := teapot.NewListRoutesHandler(r, nil)

	// Test HTML response (no Accept header defaults to HTML)
	req := httptest.NewRequest("GET", "/.internal/routes", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	if !strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		t.Errorf("expected HTML content type, got %s", w.Header().Get("Content-Type"))
	}

	// Check HTML contains route info
	body := w.Body.String()
	if !strings.Contains(body, "/users") {
		t.Error("HTML should contain /users route")
	}
	if !strings.Contains(body, "users.index") {
		t.Error("HTML should contain route name")
	}
}

// TestNewListRoutesHandlerHTMLQueryParams verifies HTML output includes query params
func TestNewListRoutesHandlerHTMLQueryParams(t *testing.T) {
	r := teapot.New()

	r.QueryGET("/bucket", dummyHandler).Query("acl").Name("bucket.get-acl")
	r.QueryGET("/bucket", dummyHandler).QueryValue("list-type", "v2").Name("bucket.list-v2")

	handler := teapot.NewListRoutesHandler(r, nil)

	req := httptest.NewRequest("GET", "/.internal/routes", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, 200, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "<th>Query</th>")
	// Existence-only param: just the key
	assert.Contains(t, body, `<td class="query">acl</td>`)
	// Value param: key=value
	assert.Contains(t, body, `<td class="query">list-type=v2</td>`)
}

// TestNewListRoutesHandlerAsRoute verifies wiring NewListRoutesHandler into a route
func TestNewListRoutesHandlerAsRoute(t *testing.T) {
	r := teapot.New()

	r.GET("/api/users", dummyHandler).Name("api.users")
	r.GET("/api/posts", dummyHandler).Name("api.posts")
	r.GET("/.internal/routes", teapot.NewListRoutesHandler(r, nil)).Name("debug.routes")

	// Test it works via ServeHTTP
	req := httptest.NewRequest("GET", "/.internal/routes", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	// Should have 3 routes (2 API + 1 debug)
	count, ok := response["count"].(float64)
	if !ok || count != 3 {
		t.Errorf("expected count=3, got %v", response["count"])
	}
}

// TestFormatRoutesJSON verifies JSON formatting helper
func TestFormatRoutesJSON(t *testing.T) {
	r := teapot.New()

	r.GET("/users", dummyHandler).Name("users.index")
	r.POST("/users", dummyHandler).Name("users.store")
	r.GET("/users/{id}", dummyHandler).Name("users.show").Action("s3:GetObject")

	routes := r.Routes()

	var buf bytes.Buffer
	if err := teapot.FormatRoutesJSON(&buf, routes); err != nil {
		t.Fatalf("FormatRoutesJSON failed: %v", err)
	}

	// Parse output
	var output map[string]any
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Check structure
	count, ok := output["count"].(float64)
	if !ok || count != 3 {
		t.Errorf("expected count=3, got %v", output["count"])
	}

	routes_list, ok := output["routes"].([]any)
	if !ok || len(routes_list) != 3 {
		t.Errorf("expected 3 routes, got %v", output["routes"])
	}

	// Verify first route has expected fields
	firstRoute, ok := routes_list[0].(map[string]any)
	if !ok {
		t.Fatal("expected route to be a map")
	}

	if _, ok := firstRoute["Method"]; !ok {
		t.Error("route should have Method field")
	}
	if _, ok := firstRoute["Pattern"]; !ok {
		t.Error("route should have Pattern field")
	}
	if _, ok := firstRoute["Name"]; !ok {
		t.Error("route should have Name field")
	}
}

// TestFormatRoutesTable verifies table formatting helper
func TestFormatRoutesTable(t *testing.T) {
	r := teapot.New()

	r.GET("/users", dummyHandler).Name("users.index")
	r.POST("/users", dummyHandler).Name("users.store")

	routes := r.Routes()

	var buf bytes.Buffer
	if err := teapot.FormatRoutesTable(&buf, routes); err != nil {
		t.Fatalf("FormatRoutesTable failed: %v", err)
	}

	output := buf.String()

	// Check header
	if !strings.Contains(output, "METHOD") {
		t.Error("table should have METHOD header")
	}
	if !strings.Contains(output, "PATTERN") {
		t.Error("table should have PATTERN header")
	}
	if !strings.Contains(output, "NAME") {
		t.Error("table should have NAME header")
	}

	// Check routes
	if !strings.Contains(output, "/users") {
		t.Error("table should contain /users route")
	}
	if !strings.Contains(output, "users.index") {
		t.Error("table should contain route name")
	}
}

// TestFormatRoutesCompact verifies compact formatting helper
func TestFormatRoutesCompact(t *testing.T) {
	r := teapot.New()

	r.GET("/users", dummyHandler).Name("users.index")
	r.POST("/users", dummyHandler).Name("users.store")

	routes := r.Routes()

	var buf bytes.Buffer
	if err := teapot.FormatRoutesCompact(&buf, routes); err != nil {
		t.Fatalf("FormatRoutesCompact failed: %v", err)
	}

	output := buf.String()

	// Check routes (should be one per line)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}

	// Check format includes method and pattern
	if !strings.Contains(output, "GET") {
		t.Error("output should contain GET method")
	}
	if !strings.Contains(output, "/users") {
		t.Error("output should contain /users pattern")
	}
}

// TestRoutesSorting verifies routes are sorted consistently
func TestRoutesSorting(t *testing.T) {
	r := teapot.New()

	// Register routes in random order
	r.POST("/users", dummyHandler).Name("z")
	r.GET("/api/posts", dummyHandler).Name("a")
	r.GET("/users", dummyHandler).Name("b")
	r.DELETE("/users", dummyHandler).Name("c")

	routes := r.Routes()

	var buf bytes.Buffer
	fmtErr := teapot.FormatRoutesJSON(&buf, routes)
	assert.NoError(t, fmtErr)

	var output map[string]any
	err := json.Unmarshal(buf.Bytes(), &output)
	assert.NoError(t, err)
	routesList := output["routes"].([]any)

	// First route should be /api/posts (sorted by pattern)
	firstRoute := routesList[0].(map[string]any)
	if firstRoute["Pattern"] != "/api/posts" {
		t.Errorf("expected first route to be /api/posts, got %s", firstRoute["Pattern"])
	}

	// Routes with same pattern should be sorted by method
	// /users: DELETE, GET, POST (alphabetical)
	secondRoute := routesList[1].(map[string]any)
	if secondRoute["Pattern"] != "/users" || secondRoute["Method"] != "DELETE" {
		t.Errorf("expected DELETE /users, got %s %s", secondRoute["Method"], secondRoute["Pattern"])
	}
}

// TestNewListRoutesHandlerWithFilter verifies the filter excludes routes
func TestNewListRoutesHandlerWithFilter(t *testing.T) {
	r := teapot.New()

	r.GET("/api/users", dummyHandler).Name("api.users")
	r.GET("/api/posts", dummyHandler).Name("api.posts")
	r.GET("/.internal/routes", teapot.NewListRoutesHandler(r, func(route teapot.RouteInfo) bool {
		return !strings.HasPrefix(route.Pattern, "/.internal/")
	})).Name("debug.routes")

	req := httptest.NewRequest("GET", "/.internal/routes", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	// Should have 2 routes - the /.internal/routes route is filtered out
	count, ok := response["count"].(float64)
	if !ok || count != 2 {
		t.Errorf("expected count=2, got %v", response["count"])
	}

	// Verify no internal routes leaked through
	routes, _ := response["routes"].([]any)
	for _, route := range routes {
		r, _ := route.(map[string]any)
		pattern, _ := r["Pattern"].(string)
		if strings.HasPrefix(pattern, "/.internal/") {
			t.Errorf("internal route %q should have been filtered out", pattern)
		}
	}
}
