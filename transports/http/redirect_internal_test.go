package httptransport

import (
	"net/http"
	"net/url"
	"testing"
)

// defaultCheckRedirect treats hostnames case-insensitively so legitimate
// same-host redirects with mixed-case spelling (Example.COM vs example.com)
// aren't refused. The match still includes the port: 8080 != 8081.
func TestDefaultCheckRedirectCaseInsensitive(t *testing.T) {
	cases := []struct {
		name, from, to string
		wantErr        bool
	}{
		{"same host, same case", "https://example.com/a", "https://example.com/b", false},
		{"same host, different case", "https://Example.COM/a", "https://example.com/b", false},
		{"same host with port, different case", "https://Example.COM:8080/a", "https://example.com:8080/b", false},
		{"different host", "https://a.example.com/", "https://b.example.com/", true},
		{"same host, different port", "https://example.com:8080/", "https://example.com:8081/", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			via := []*http.Request{{URL: mustParseURL(t, tc.from)}}
			next := &http.Request{URL: mustParseURL(t, tc.to)}
			err := defaultCheckRedirect(next, via)
			if tc.wantErr && err == nil {
				t.Errorf("expected cross-host refusal, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected refusal: %v", err)
			}
		})
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return u
}
