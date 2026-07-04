package urlbuilder_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/urlbuilder"
)

func TestSubdomainFromHost(t *testing.T) {
	tests := []struct {
		name            string
		host            string
		canonicalDomain string
		wantLabel       string
		wantOK          bool
	}{
		{
			name:            "bare canonical domain is not a subdomain",
			host:            "s3.example.com",
			canonicalDomain: "s3.example.com",
			wantLabel:       "",
			wantOK:          false,
		},
		{
			name:            "simple subdomain resolves label",
			host:            "mybucket.s3.example.com",
			canonicalDomain: "s3.example.com",
			wantLabel:       "mybucket",
			wantOK:          true,
		},
		{
			name:            "IP host is not a subdomain",
			host:            "192.168.1.10",
			canonicalDomain: "s3.example.com",
			wantLabel:       "",
			wantOK:          false,
		},
		{
			name:            "port suffix is stripped before matching",
			host:            "mybucket.s3.example.com:9000",
			canonicalDomain: "s3.example.com",
			wantLabel:       "mybucket",
			wantOK:          true,
		},
		{
			name:            "IPv6 host with port is not a subdomain",
			host:            "[::1]:9000",
			canonicalDomain: "s3.example.com",
			wantLabel:       "",
			wantOK:          false,
		},
		{
			name:            "dotted label still resolves",
			host:            "my.dotted.bucket.s3.example.com",
			canonicalDomain: "s3.example.com",
			wantLabel:       "my.dotted.bucket",
			wantOK:          true,
		},
		{
			name:            "empty canonical domain disables resolution",
			host:            "mybucket.s3.example.com",
			canonicalDomain: "",
			wantLabel:       "",
			wantOK:          false,
		},
		{
			name:            "mixed-case host normalizes to lowercase label",
			host:            "MyBucket.S3.Example.Com",
			canonicalDomain: "s3.example.com",
			wantLabel:       "mybucket",
			wantOK:          true,
		},
		{
			name:            "unrelated host is not a subdomain",
			host:            "example.org",
			canonicalDomain: "s3.example.com",
			wantLabel:       "",
			wantOK:          false,
		},
		{
			name:            "host that merely contains canonical domain as substring, not suffix",
			host:            "s3.example.com.evil.com",
			canonicalDomain: "s3.example.com",
			wantLabel:       "",
			wantOK:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label, ok := urlbuilder.SubdomainFromHost(tt.host, tt.canonicalDomain)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantLabel, label)
		})
	}
}

func TestSubdomainURL(t *testing.T) {
	tests := []struct {
		name            string
		canonicalDomain string
		label           string
		path            string
		wantURL         string
		wantOK          bool
	}{
		{
			name:            "label and root path",
			canonicalDomain: "s3.example.com",
			label:           "mybucket",
			path:            "",
			wantURL:         "https://mybucket.s3.example.com",
			wantOK:          true,
		},
		{
			name:            "label with sub-path",
			canonicalDomain: "s3.example.com",
			label:           "mybucket",
			path:            "/path/to/object.txt",
			wantURL:         "https://mybucket.s3.example.com/path/to/object.txt",
			wantOK:          true,
		},
		{
			name:            "no canonical domain configured",
			canonicalDomain: "",
			label:           "mybucket",
			path:            "",
			wantURL:         "",
			wantOK:          false,
		},
		{
			name:            "empty label",
			canonicalDomain: "s3.example.com",
			label:           "",
			path:            "/key",
			wantURL:         "",
			wantOK:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := urlbuilder.New(tt.canonicalDomain)
			url, ok := b.SubdomainURL(tt.label, tt.path)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantURL, url)
		})
	}
}
