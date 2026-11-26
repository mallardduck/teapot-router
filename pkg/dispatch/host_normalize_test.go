package dispatch

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// HostHasSubdomain inherits the Host normalization rules from
// urlbuilder.SubdomainFromHost, so a trailing dot on either side must not
// change the result.
func TestHostHasSubdomainTrailingDot(t *testing.T) {
	tests := []struct {
		name            string
		host            string
		canonicalDomain string
		want            bool
	}{
		{"trailing dot on host", "mybucket.s3.example.com.", "s3.example.com", true},
		{"trailing dot on canonical domain", "mybucket.s3.example.com", "s3.example.com.", true},
		{"bare domain with trailing dot", "s3.example.com.", "s3.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://placeholder/", nil)
			req.Host = tt.host
			assert.Equal(t, tt.want, HostHasSubdomain(tt.canonicalDomain).Matches(req))
		})
	}
}
