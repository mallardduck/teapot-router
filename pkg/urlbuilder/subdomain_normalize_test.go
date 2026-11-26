package urlbuilder_test

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/urlbuilder"
)

// A trailing dot is the explicit DNS root label. "example.com." and
// "example.com" name the same host, so matching treats them as equal. Hosts
// are otherwise matched literally: anything malformed fails closed.
func TestSubdomainFromHostTrailingDot(t *testing.T) {
	tests := []struct {
		name            string
		host            string
		canonicalDomain string
		wantLabel       string
		wantOK          bool
	}{
		{"trailing dot on host", "mybucket.s3.example.com.", "s3.example.com", "mybucket", true},
		{"trailing dot with port", "mybucket.s3.example.com.:9000", "s3.example.com", "mybucket", true},
		{"trailing dot on canonical domain", "mybucket.s3.example.com", "s3.example.com.", "mybucket", true},
		{"trailing dot on both", "mybucket.s3.example.com.", "s3.example.com.", "mybucket", true},
		{"bare canonical domain with trailing dot", "s3.example.com.", "s3.example.com", "", false},
		{"double trailing dot is malformed", "mybucket.s3.example.com..", "s3.example.com", "", false},
		{"root dot alone is not a label", ".", "s3.example.com", "", false},
		{"spoofing with trailing dot fails closed", "s3.example.com.evil.com.", "s3.example.com", "", false},
		{"mixed case with trailing dot", "MyBucket.S3.Example.Com.", "s3.example.com", "mybucket", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label, ok := urlbuilder.SubdomainFromHost(tt.host, tt.canonicalDomain)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantLabel, label)
		})
	}
}

// Generated URLs are canonical: lowercase, and never carry a trailing dot,
// regardless of how the canonical domain was configured.
func TestSubdomainURLCanonicalOutput(t *testing.T) {
	tests := []struct {
		name            string
		canonicalDomain string
		label           string
		path            string
		wantURL         string
	}{
		{"trailing dot configured", "s3.example.com.", "mybucket", "/key", "https://mybucket.s3.example.com/key"},
		{"uppercase configured", "S3.Example.COM", "mybucket", "/key", "https://mybucket.s3.example.com/key"},
		{"uppercase label", "s3.example.com", "MyBucket", "/key", "https://mybucket.s3.example.com/key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := urlbuilder.New(tt.canonicalDomain)
			url, ok := b.SubdomainURL(tt.label, tt.path)
			assert.True(t, ok)
			assert.Equal(t, tt.wantURL, url)
		})
	}
}

// BucketURL and friends use the canonical domain too, so the same
// normalization applies there.
func TestBuilderCanonicalDomainNormalized(t *testing.T) {
	b := urlbuilder.New("S3.Example.com.")
	r := httptest.NewRequest("GET", "http://ignored/", nil)
	assert.Equal(t, "https://s3.example.com/mybucket", b.BucketURL(r, "mybucket"))
}
