package urlbuilder

import (
	"fmt"
	"net"
	"strings"
)

// SubdomainFromHost extracts the leftmost DNS label from host when host is a
// strict subdomain of canonicalDomain, i.e. "{label}.{canonicalDomain}".
// It returns ok=false when:
//   - canonicalDomain is empty (subdomain resolution disabled)
//   - host equals canonicalDomain exactly (bare domain, not a subdomain)
//   - host does not end in "."+canonicalDomain (this also covers bare IP
//     hosts and hosts that merely contain canonicalDomain as a substring
//     rather than as a dot-separated suffix, e.g. "canonicalDomain.evil.com")
//
// A port suffix on host, if present, is stripped before comparison. Matching
// is case-insensitive; the returned label is lowercased.
//
// This is a general host-based routing primitive - e.g. a multi-tenant app
// keying a tenant off a subdomain, or S3-style virtual-hosted bucket
// addressing ("{bucket}.s3.example.com"). If the result feeds a routing or
// access-control decision, only ever pass a value read directly from a
// request's Host field - never a client-controlled forwarded header, since
// that would let a client pick the resolved label independently of what was
// actually signed or authenticated.
func SubdomainFromHost(host, canonicalDomain string) (label string, ok bool) {
	if canonicalDomain == "" {
		return "", false
	}

	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	host = strings.ToLower(host)
	canonicalDomain = strings.ToLower(canonicalDomain)

	if host == canonicalDomain {
		return "", false
	}

	suffix := "." + canonicalDomain
	if !strings.HasSuffix(host, suffix) {
		return "", false
	}

	label = strings.TrimSuffix(host, suffix)
	if label == "" {
		return "", false
	}

	return label, true
}

// SubdomainURL builds an absolute URL using label as the leftmost subdomain
// of the Builder's canonical domain: https://{label}.{canonicalDomain}{path}.
// It returns ok=false if the Builder has no canonical domain configured or
// label is empty - subdomain-style URLs have no meaningful "detect from
// request" fallback the way BuildURL does, since the domain itself must be
// fixed for the label to resolve at all.
func (b *Builder) SubdomainURL(label, path string) (url string, ok bool) {
	if b.canonicalDomain == "" || label == "" {
		return "", false
	}
	return fmt.Sprintf("https://%s.%s%s", label, b.canonicalDomain, path), true
}
