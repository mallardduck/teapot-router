package urlbuilder_test

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/urlbuilder"
)

// TestBucketURLFromRequest verifies URL generation from request Host header
func TestBucketURLFromRequest(t *testing.T) {
	b := urlbuilder.New("")

	req := httptest.NewRequest("GET", "http://localhost:8080/mybucket", nil)
	req.Host = "localhost:8080"

	url := b.BucketURL(req, "mybucket")
	expected := "http://localhost:8080/mybucket"

	assert.Equal(t, expected, url)
}

// TestObjectURLFromRequest verifies object URL generation from request
func TestObjectURLFromRequest(t *testing.T) {
	b := urlbuilder.New("")

	req := httptest.NewRequest("GET", "http://localhost:8080/mybucket/mykey", nil)
	req.Host = "localhost:8080"

	url := b.ObjectURL(req, "mybucket", "mykey")
	expected := "http://localhost:8080/mybucket/mykey"

	assert.Equal(t, expected, url)
}

// TestCanonicalDomain verifies canonical domain overrides request host
func TestCanonicalDomain(t *testing.T) {
	b := urlbuilder.New("s3.example.com")

	req := httptest.NewRequest("GET", "http://localhost:8080/mybucket", nil)
	req.Host = "localhost:8080"

	url := b.BucketURL(req, "mybucket")
	expected := "https://s3.example.com/mybucket"

	assert.Equal(t, expected, url)
}

// TestCanonicalDomainHTTPS verifies canonical domain uses HTTPS
func TestCanonicalDomainHTTPS(t *testing.T) {
	b := urlbuilder.New("s3.production.com")

	req := httptest.NewRequest("GET", "http://localhost/bucket/key", nil)

	url := b.ObjectURL(req, "bucket", "key")
	expected := "https://s3.production.com/bucket/key"

	assert.Equal(t, expected, url)
}

// TestTLSDetection verifies HTTPS detection from TLS state
func TestTLSDetection(t *testing.T) {
	b := urlbuilder.New("")

	req := httptest.NewRequest("GET", "https://example.com/bucket", nil)
	req.Host = "example.com"
	req.TLS = &tls.ConnectionState{} // Simulate TLS connection

	url := b.BucketURL(req, "bucket")
	expected := "https://example.com/bucket"

	assert.Equal(t, expected, url)
}

// TestXForwardedProto verifies X-Forwarded-Proto header detection
func TestXForwardedProto(t *testing.T) {
	b := urlbuilder.New("")

	req := httptest.NewRequest("GET", "http://localhost/bucket", nil)
	req.Host = "example.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	url := b.BucketURL(req, "bucket")
	expected := "https://example.com/bucket"

	assert.Equal(t, expected, url)
}

// TestXForwardedProtoCaseInsensitive verifies header is case-insensitive
func TestXForwardedProtoCaseInsensitive(t *testing.T) {
	b := urlbuilder.New("")

	req := httptest.NewRequest("GET", "http://localhost/bucket", nil)
	req.Host = "example.com"
	req.Header.Set("X-Forwarded-Proto", "HTTPS")

	url := b.BucketURL(req, "bucket")
	expected := "https://example.com/bucket"

	assert.Equal(t, expected, url)
}

// TestDefaultHTTP verifies default scheme is http
func TestDefaultHTTP(t *testing.T) {
	b := urlbuilder.New("")

	req := httptest.NewRequest("GET", "http://localhost/bucket", nil)
	req.Host = "localhost"
	// No TLS, no X-Forwarded-Proto

	url := b.BucketURL(req, "bucket")
	expected := "http://localhost/bucket"

	assert.Equal(t, expected, url)
}

// TestObjectURLWithSlashInKey verifies keys with slashes work correctly
func TestObjectURLWithSlashInKey(t *testing.T) {
	b := urlbuilder.New("")

	req := httptest.NewRequest("GET", "http://localhost/bucket", nil)
	req.Host = "localhost"

	url := b.ObjectURL(req, "mybucket", "path/to/object.txt")
	expected := "http://localhost/mybucket/path/to/object.txt"

	assert.Equal(t, expected, url)
}

// TestNonStandardPort verifies port numbers are preserved
func TestNonStandardPort(t *testing.T) {
	b := urlbuilder.New("")

	req := httptest.NewRequest("GET", "http://localhost:9000/bucket", nil)
	req.Host = "localhost:9000"

	url := b.BucketURL(req, "mybucket")
	expected := "http://localhost:9000/mybucket"

	assert.Equal(t, expected, url)
}

// TestSchemePriorityCanonical verifies canonical domain takes priority
func TestSchemePriorityCanonical(t *testing.T) {
	b := urlbuilder.New("s3.example.com")

	req := httptest.NewRequest("GET", "http://localhost/bucket", nil)
	req.Host = "localhost"
	req.Header.Set("X-Forwarded-Proto", "http")
	// Even though X-Forwarded-Proto says http, canonical domain should force https

	url := b.BucketURL(req, "bucket")
	expected := "https://s3.example.com/bucket"

	assert.Equal(t, expected, url)
}

// TestSchemePriorityXForwardedProto verifies X-Forwarded-Proto takes priority over TLS
func TestSchemePriorityXForwardedProto(t *testing.T) {
	b := urlbuilder.New("")

	req := httptest.NewRequest("GET", "http://localhost/bucket", nil)
	req.Host = "localhost"
	req.Header.Set("X-Forwarded-Proto", "https")
	// TLS is nil, but X-Forwarded-Proto should be used

	url := b.BucketURL(req, "bucket")
	expected := "https://localhost/bucket"

	assert.Equal(t, expected, url)
}

// TestMultipleBuckets verifies builder can generate URLs for different buckets
func TestMultipleBuckets(t *testing.T) {
	b := urlbuilder.New("s3.example.com")

	req := httptest.NewRequest("GET", "http://localhost/", nil)

	tests := []struct {
		bucket   string
		expected string
	}{
		{"bucket1", "https://s3.example.com/bucket1"},
		{"bucket2", "https://s3.example.com/bucket2"},
		{"my-bucket", "https://s3.example.com/my-bucket"},
	}

	for _, tt := range tests {
		url := b.BucketURL(req, tt.bucket)
		assert.Equal(t, tt.expected, url, "bucket %s", tt.bucket)
	}
}

// TestMultipleObjects verifies builder can generate URLs for different objects
func TestMultipleObjects(t *testing.T) {
	b := urlbuilder.New("s3.example.com")

	req := httptest.NewRequest("GET", "http://localhost/", nil)

	tests := []struct {
		bucket   string
		key      string
		expected string
	}{
		{"bucket1", "file.txt", "https://s3.example.com/bucket1/file.txt"},
		{"bucket2", "data.json", "https://s3.example.com/bucket2/data.json"},
		{"photos", "2024/01/photo.jpg", "https://s3.example.com/photos/2024/01/photo.jpg"},
	}

	for _, tt := range tests {
		url := b.ObjectURL(req, tt.bucket, tt.key)
		assert.Equal(t, tt.expected, url, "bucket=%s, key=%s", tt.bucket, tt.key)
	}
}

// TestBuildURL verifies BuildURL method works with arbitrary paths
func TestBuildURL(t *testing.T) {
	b := urlbuilder.New("s3.example.com")

	req := httptest.NewRequest("GET", "http://localhost/", nil)

	tests := []struct {
		path     string
		expected string
	}{
		{"/mybucket", "https://s3.example.com/mybucket"},
		{"/mybucket/mykey", "https://s3.example.com/mybucket/mykey"},
		{"/api/v1/buckets", "https://s3.example.com/api/v1/buckets"},
		{"/", "https://s3.example.com/"},
	}

	for _, tt := range tests {
		url := b.BuildURL(req, tt.path)
		assert.Equal(t, tt.expected, url, "path=%s", tt.path)
	}
}

// TestBuildURLFromRequest verifies BuildURL uses request host when no canonical domain
func TestBuildURLFromRequest(t *testing.T) {
	b := urlbuilder.New("")

	req := httptest.NewRequest("GET", "http://localhost:9000/", nil)
	req.Host = "localhost:9000"

	url := b.BuildURL(req, "/mybucket/mykey")
	expected := "http://localhost:9000/mybucket/mykey"

	assert.Equal(t, expected, url)
}

// TestBuildURLWithTLS verifies BuildURL respects TLS
func TestBuildURLWithTLS(t *testing.T) {
	b := urlbuilder.New("")

	req := httptest.NewRequest("GET", "https://example.com/", nil)
	req.Host = "example.com"
	req.TLS = &tls.ConnectionState{}

	url := b.BuildURL(req, "/mybucket")
	expected := "https://example.com/mybucket"

	assert.Equal(t, expected, url)
}
