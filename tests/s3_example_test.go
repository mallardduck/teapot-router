package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestS3StyleAPI demonstrates a complete S3-style API implementation
func TestS3StyleAPI(t *testing.T) {
	r := teapot.New()

	// Service endpoint
	r.GET("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("LIST_BUCKETS"))
	}).Name("service.list").Action("s3:ListAllMyBuckets")

	// Bucket operations
	r.NamedGroup("/{bucket}", "bucket", func(r *teapot.Router) {
		// Basic bucket operations (no query multiplexing - use standard methods)
		r.DELETE("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("DELETE_BUCKET"))
		}).Name("delete").Action("s3:DeleteBucket")

		r.HEAD("", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}).Name("head").Action("s3:HeadBucket")

		// Query-based bucket operations (S3's special sauce!) - use QueryGET/QueryPUT
		r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("LIST_OBJECTS"))
		}).Name("list").Action("s3:ListBucket")

		r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("GET_BUCKET_ACL"))
		}).Name("acl.get").Action("s3:GetBucketAcl").Query("acl")

		r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("LIST_VERSIONS"))
		}).Name("versions").Action("s3:ListBucketVersions").Query("versions")

		r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("GET_LIFECYCLE"))
		}).Name("lifecycle").Action("s3:GetLifecycleConfiguration").Query("lifecycle")

		// PUT bucket operations - needs query multiplexing for ACL
		r.QueryPUT("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("CREATE_BUCKET"))
		}).Name("create").Action("s3:CreateBucket")

		r.QueryPUT("", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("PUT_BUCKET_ACL"))
		}).Name("acl.put").Action("s3:PutBucketAcl").Query("acl")

		// Object operations
		r.NamedGroup("/{key:.*}", "object", func(r *teapot.Router) {
			// Basic object operations (no query multiplexing)
			r.DELETE("", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("DELETE_OBJECT"))
			}).Name("delete").Action("s3:DeleteObject")

			r.HEAD("", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			}).Name("head").Action("s3:HeadObject")

			// Query-based object operations - use QueryGET/QueryPUT/QueryPOST
			r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
				key := teapot.URLParam(r, "key")
				action := teapot.GetAction(r)
				w.Write([]byte("GET_OBJECT:" + key + ":" + action))
			}).Name("get").Action("s3:GetObject")

			r.QueryPUT("", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("PUT_OBJECT"))
			}).Name("put").Action("s3:PutObject")

			r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("GET_OBJECT_ACL"))
			}).Name("acl.get").Action("s3:GetObjectAcl").Query("acl")

			r.QueryPOST("", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("INITIATE_MULTIPART"))
			}).Name("multipart.initiate").Action("s3:CreateMultipartUpload").Query("uploads")

			r.QueryPUT("", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("UPLOAD_PART"))
			}).Name("multipart.upload").Action("s3:UploadPart").Query("partNumber").Query("uploadId")
		})
	})

	// Test service endpoint
	w := request(t, r, "GET", "/")
	if w.Body.String() != "LIST_BUCKETS" {
		t.Errorf("service endpoint failed: %s", w.Body.String())
	}

	// Test basic bucket operations
	tests := []struct {
		method   string
		path     string
		expected string
	}{
		{"PUT", "/mybucket", "CREATE_BUCKET"},
		{"DELETE", "/mybucket", "DELETE_BUCKET"},
		{"GET", "/mybucket", "LIST_OBJECTS"},
		{"GET", "/mybucket?acl", "GET_BUCKET_ACL"},
		{"PUT", "/mybucket?acl", "PUT_BUCKET_ACL"},
		{"GET", "/mybucket?versions", "LIST_VERSIONS"},
		{"GET", "/mybucket?lifecycle", "GET_LIFECYCLE"},
	}

	for _, tt := range tests {
		w := request(t, r, tt.method, tt.path)
		if w.Body.String() != tt.expected {
			t.Errorf("%s %s: got %q, want %q", tt.method, tt.path, w.Body.String(), tt.expected)
		}
	}

	// Test object operations
	w = request(t, r, "GET", "/mybucket/path/to/file.txt")
	if w.Body.String() != "GET_OBJECT:path/to/file.txt:s3:GetObject" {
		t.Errorf("object get failed: %s", w.Body.String())
	}

	w = request(t, r, "PUT", "/mybucket/file.txt")
	if w.Body.String() != "PUT_OBJECT" {
		t.Errorf("object put failed: %s", w.Body.String())
	}

	w = request(t, r, "GET", "/mybucket/file.txt?acl")
	if w.Body.String() != "GET_OBJECT_ACL" {
		t.Errorf("object acl failed: %s", w.Body.String())
	}

	w = request(t, r, "POST", "/mybucket/file.txt?uploads")
	if w.Body.String() != "INITIATE_MULTIPART" {
		t.Errorf("multipart initiate failed: %s", w.Body.String())
	}

	w = request(t, r, "PUT", "/mybucket/file.txt?partNumber=1&uploadId=abc123")
	if w.Body.String() != "UPLOAD_PART" {
		t.Errorf("upload part failed: %s", w.Body.String())
	}

	// Test URL generation
	url := r.MustURL("bucket.list", "bucket", "test-bucket")
	if url != "/test-bucket" {
		t.Errorf("URL generation failed: got %q, want %q", url, "/test-bucket")
	}

	url = r.MustURL("bucket.object.get", "bucket", "test-bucket", "key", "path/to/file.txt")
	if url != "/test-bucket/path/to/file.txt" {
		t.Errorf("URL generation failed: got %q, want %q", url, "/test-bucket/path/to/file.txt")
	}
}

// Helper to make test requests
func request(t *testing.T, router http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
