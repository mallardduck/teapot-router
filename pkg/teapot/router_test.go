package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// Helper to make test requests
func request(t *testing.T, router http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// Test: Basic HTTP methods
func TestBasicHTTPMethods(t *testing.T) {
	r := teapot.New()

	r.GET("/get", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GET"))
	})
	r.POST("/post", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("POST"))
	})
	r.PUT("/put", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PUT"))
	})
	r.DELETE("/delete", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("DELETE"))
	})
	r.HEAD("/head", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	assertEqual(t, request(t, r, "GET", "/get").Body.String(), "GET")
	assertEqual(t, request(t, r, "POST", "/post").Body.String(), "POST")
	assertEqual(t, request(t, r, "PUT", "/put").Body.String(), "PUT")
	assertEqual(t, request(t, r, "DELETE", "/delete").Body.String(), "DELETE")
	assertEqual(t, request(t, r, "HEAD", "/head").Code, 200)
}

// Test: URL parameters
func TestURLParameters(t *testing.T) {
	r := teapot.New()

	r.GET("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := teapot.URLParam(r, "id")
		w.Write([]byte("User: " + id))
	})

	w := request(t, r, "GET", "/users/123")
	assertEqual(t, w.Body.String(), "User: 123")
}

// Test: Wildcard parameters
func TestWildcardParameters(t *testing.T) {
	r := teapot.New()

	r.GET("/files/{path:.*}", func(w http.ResponseWriter, r *http.Request) {
		path := teapot.URLParam(r, "path")
		w.Write([]byte("Path: " + path))
	})

	w := request(t, r, "GET", "/files/documents/report.pdf")
	assertEqual(t, w.Body.String(), "Path: documents/report.pdf")
}

// Test: Named routes and URL generation
func TestNamedRoutes(t *testing.T) {
	r := teapot.New()

	r.GET("/users/{id}", func(w http.ResponseWriter, r *http.Request) {}).
		Name("users.show")

	r.GET("/posts/{postId}/comments/{commentId}", func(w http.ResponseWriter, r *http.Request) {}).
		Name("posts.comments.show")

	// Test URL generation
	url := r.MustURL("users.show", "id", "42")
	assertEqual(t, url, "/users/42")

	url = r.MustURL("posts.comments.show", "postId", "10", "commentId", "99")
	assertEqual(t, url, "/posts/10/comments/99")

	// Test URL() with error handling
	_, err := r.URL("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent route")
	}
}

// Test: S3 Action and Route Name context injection
func TestActionAndNameContext(t *testing.T) {
	r := teapot.New()

	var capturedAction, capturedName string
	r.GET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		capturedAction = teapot.GetAction(r)
		capturedName = teapot.GetRouteName(r)
		w.Write([]byte("OK"))
	}).Name("bucket.list").Action("s3:ListBucket")

	request(t, r, "GET", "/bucket")
	assertEqual(t, capturedAction, "s3:ListBucket")
	assertEqual(t, capturedName, "bucket.list")
}

// Test: Query parameter existence matching
func TestQueryParameterMatching(t *testing.T) {
	r := teapot.New()

	r.QueryGET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("LIST"))
	}).Name("bucket.list")

	r.QueryGET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ACL"))
	}).Name("bucket.acl").Query("acl")

	r.QueryGET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("VERSIONING"))
	}).Name("bucket.versioning").Query("versioning")

	// No query params = first route (base route)
	assertEqual(t, request(t, r, "GET", "/bucket").Body.String(), "LIST")

	// ?acl = second route
	assertEqual(t, request(t, r, "GET", "/bucket?acl").Body.String(), "ACL")

	// ?versioning = third route
	assertEqual(t, request(t, r, "GET", "/bucket?versioning").Body.String(), "VERSIONING")
}

// Test: Query parameter value matching
func TestQueryValueMatching(t *testing.T) {
	r := teapot.New()

	r.QueryGET("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("FULL"))
	}).QueryValue("type", "full")

	r.QueryGET("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PARTIAL"))
	}).QueryValue("type", "partial")

	r.QueryGET("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("DEFAULT"))
	})

	assertEqual(t, request(t, r, "GET", "/search?type=full").Body.String(), "FULL")
	assertEqual(t, request(t, r, "GET", "/search?type=partial").Body.String(), "PARTIAL")
	assertEqual(t, request(t, r, "GET", "/search").Body.String(), "DEFAULT")
	assertEqual(t, request(t, r, "GET", "/search?type=other").Body.String(), "DEFAULT")
}

// Test: Multiple query parameter matching (priority)
func TestMultipleQueryParameterPriority(t *testing.T) {
	r := teapot.New()

	// More specific (2 params) should match first
	r.QueryGET("/object", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("TWO_PARAMS"))
	}).Query("partNumber").Query("uploadId")

	// Less specific (1 param)
	r.QueryGET("/object", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("UPLOAD_ID"))
	}).Query("uploadId")

	// No query params
	r.QueryGET("/object", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("BASE"))
	})

	// Two params = most specific route
	assertEqual(t, request(t, r, "GET", "/object?uploadId=123&partNumber=1").Body.String(), "TWO_PARAMS")

	// One param = less specific route
	assertEqual(t, request(t, r, "GET", "/object?uploadId=123").Body.String(), "UPLOAD_ID")

	// No params = base route
	assertEqual(t, request(t, r, "GET", "/object").Body.String(), "BASE")
}

// Test: Route groups with path prefix
func TestRouteGroups(t *testing.T) {
	r := teapot.New()

	r.Group("/api", func(r *teapot.Router) {
		r.GET("/users", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("USERS"))
		})
		r.GET("/posts", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("POSTS"))
		})
	})

	assertEqual(t, request(t, r, "GET", "/api/users").Body.String(), "USERS")
	assertEqual(t, request(t, r, "GET", "/api/posts").Body.String(), "POSTS")
}

// Test: Named groups (path + name prefix)
func TestNamedGroups(t *testing.T) {
	r := teapot.New()

	r.NamedGroup("/{bucket}", "bucket", func(r *teapot.Router) {
		r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("LIST"))
		}).Name("list")

		r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ACL"))
		}).Name("acl").Query("acl")

		r.NamedGroup("/{key:.*}", "object", func(r *teapot.Router) {
			r.GET("", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("GET_OBJECT"))
			}).Name("get")
		})
	})

	// Test routes work
	assertEqual(t, request(t, r, "GET", "/mybucket").Body.String(), "LIST")
	assertEqual(t, request(t, r, "GET", "/mybucket?acl").Body.String(), "ACL")
	assertEqual(t, request(t, r, "GET", "/mybucket/path/to/file.txt").Body.String(), "GET_OBJECT")

	// Test name composition
	url := r.MustURL("bucket.list", "bucket", "test")
	assertEqual(t, url, "/test")

	url = r.MustURL("bucket.object.get", "bucket", "test", "key", "file.txt")
	assertEqual(t, url, "/test/file.txt")
}

// Test: Route-specific middleware
func TestRouteMiddleware(t *testing.T) {
	r := teapot.New()

	middlewareCalled := false
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}

	r.GET("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}).With(mw)

	r.GET("/public", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Protected route should trigger middleware
	middlewareCalled = false
	request(t, r, "GET", "/protected")
	if !middlewareCalled {
		t.Error("middleware was not called for protected route")
	}

	// Public route should not trigger middleware
	middlewareCalled = false
	request(t, r, "GET", "/public")
	if middlewareCalled {
		t.Error("middleware was called for public route")
	}
}

// Test: Global middleware via Use()
func TestGlobalMiddleware(t *testing.T) {
	r := teapot.New()

	callCount := 0
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			next.ServeHTTP(w, r)
		})
	})

	r.GET("/a", func(w http.ResponseWriter, r *http.Request) {})
	r.GET("/b", func(w http.ResponseWriter, r *http.Request) {})

	request(t, r, "GET", "/a")
	request(t, r, "GET", "/b")

	assertEqual(t, callCount, 2)
}

// Test: S3 bucket operations scenario
func TestS3BucketOperations(t *testing.T) {
	r := teapot.New()

	r.NamedGroup("/{bucket}", "bucket", func(r *teapot.Router) {
		r.PUT("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("CREATE"))
		}).Name("create").Action("s3:CreateBucket")

		r.DELETE("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("DELETE"))
		}).Name("delete").Action("s3:DeleteBucket")

		r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("LIST"))
		}).Name("list").Action("s3:ListBucket")

		r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ACL"))
		}).Name("acl.get").Action("s3:GetBucketAcl").Query("acl")
	})

	assertEqual(t, request(t, r, "PUT", "/mybucket").Body.String(), "CREATE")
	assertEqual(t, request(t, r, "DELETE", "/mybucket").Body.String(), "DELETE")
	assertEqual(t, request(t, r, "GET", "/mybucket").Body.String(), "LIST")
	assertEqual(t, request(t, r, "GET", "/mybucket?acl").Body.String(), "ACL")
}

// Test: S3 object operations scenario
func TestS3ObjectOperations(t *testing.T) {
	r := teapot.New()

	r.NamedGroup("/{bucket}/{key:.*}", "object", func(r *teapot.Router) {
		var capturedBucket, capturedKey string

		r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			capturedBucket = teapot.URLParam(r, "bucket")
			capturedKey = teapot.URLParam(r, "key")
			w.Write([]byte("GET"))
		}).Name("get").Action("s3:GetObject")

		r.PUT("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("PUT"))
		}).Name("put").Action("s3:PutObject")

		r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ACL"))
		}).Name("acl").Action("s3:GetObjectAcl").Query("acl")

		request(t, r, "GET", "/mybucket/path/to/file.txt")
		assertEqual(t, capturedBucket, "mybucket")
		assertEqual(t, capturedKey, "path/to/file.txt")
	})

	assertEqual(t, request(t, r, "GET", "/mybucket/file.txt").Body.String(), "GET")
	assertEqual(t, request(t, r, "PUT", "/mybucket/file.txt").Body.String(), "PUT")
	assertEqual(t, request(t, r, "GET", "/mybucket/file.txt?acl").Body.String(), "ACL")
}

// Test: Route introspection
func TestRouteIntrospection(t *testing.T) {
	r := teapot.New()

	r.GET("/users", func(w http.ResponseWriter, r *http.Request) {}).
		Name("users.list").
		Action("app:ListUsers")

	r.POST("/users", func(w http.ResponseWriter, r *http.Request) {}).
		Name("users.create").
		Action("app:CreateUser")

	routes := r.Routes()
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}

	// Check first route
	if routes[0].Method != "GET" || routes[0].Pattern != "/users" || routes[0].Name != "users.list" || routes[0].Action != "app:ListUsers" {
		t.Errorf("unexpected route info: %+v", routes[0])
	}
}

// Test: Edge case - empty route name
func TestEmptyRouteName(t *testing.T) {
	r := teapot.New()

	// Should work without a name
	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	assertEqual(t, request(t, r, "GET", "/test").Body.String(), "OK")
}

// Test: Edge case - trailing slashes
func TestTrailingSlashes(t *testing.T) {
	r := teapot.New()

	r.GET("/path", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Chi handles trailing slash redirects by default
	// Just verify the route works
	w := request(t, r, "GET", "/path")
	assertEqual(t, w.Body.String(), "OK")
}

// Test: Same name different methods (Laravel-style resources)
func TestSameNameDifferentMethods(t *testing.T) {
	r := teapot.New()

	r.GET("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("SHOW"))
	}).Name("users.show")

	r.PUT("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("UPDATE"))
	}).Name("users.update")

	r.DELETE("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("DELETE"))
	}).Name("users.destroy")

	// All should generate the same URL pattern
	url := r.MustURL("users.show", "id", "42")
	assertEqual(t, url, "/users/42")

	url = r.MustURL("users.update", "id", "42")
	assertEqual(t, url, "/users/42")

	// But routes should work correctly
	assertEqual(t, request(t, r, "GET", "/users/42").Body.String(), "SHOW")
	assertEqual(t, request(t, r, "PUT", "/users/42").Body.String(), "UPDATE")
	assertEqual(t, request(t, r, "DELETE", "/users/42").Body.String(), "DELETE")
}

// Test: NotFound handler
func TestNotFoundHandler(t *testing.T) {
	r := teapot.New()

	r.GET("/exists", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Non-existent route should return 404
	w := request(t, r, "GET", "/nonexistent")
	assertEqual(t, w.Code, 404)
}

// Test: Handler function access to context values
func TestContextAccess(t *testing.T) {
	r := teapot.New()

	var values []string
	r.GET("/{bucket}/{key:.*}", func(w http.ResponseWriter, r *http.Request) {
		values = append(values, teapot.GetAction(r))
		values = append(values, teapot.GetRouteName(r))
		values = append(values, teapot.URLParam(r, "bucket"))
		values = append(values, teapot.URLParam(r, "key"))
		w.Write([]byte("OK"))
	}).Name("object.get").Action("s3:GetObject")

	request(t, r, "GET", "/mybucket/path/to/file.txt")

	expected := []string{"s3:GetObject", "object.get", "mybucket", "path/to/file.txt"}
	for i, v := range expected {
		if values[i] != v {
			t.Errorf("index %d: got %q, want %q", i, values[i], v)
		}
	}
}
