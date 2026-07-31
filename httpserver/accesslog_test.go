package httpserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestShouldSkipAccessLog verifies exact, case-sensitive path matching for
// access-log exclusions.
func TestShouldSkipAccessLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		// name identifies the test case.
		name string
		// ignorePaths is the exclusion list under test.
		ignorePaths []string
		// path is the request path to check.
		path string
		// wantSkip is whether the path should be excluded from logging.
		wantSkip bool
	}{
		{
			name:        "exact match skips",
			ignorePaths: []string{"/metrics"},
			path:        "/metrics",
			wantSkip:    true,
		},
		{
			name:        "different case does not skip",
			ignorePaths: []string{"/metrics"},
			path:        "/Metrics",
			wantSkip:    false,
		},
		{
			name:        "prefix does not skip",
			ignorePaths: []string{"/metrics"},
			path:        "/metrics/foo",
			wantSkip:    false,
		},
		{
			name:        "empty list does not skip",
			ignorePaths: nil,
			path:        "/metrics",
			wantSkip:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantSkip, shouldSkipAccessLog(tt.ignorePaths, tt.path))
		})
	}
}

// TestResolveClientIP verifies proxy-header precedence and peer fallback for
// access-log client identity.
func TestResolveClientIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		// name identifies the test case.
		name string
		// peerAddr is the remote peer address, optionally with a port.
		peerAddr string
		// forwardedFor is the X-Forwarded-For header value.
		forwardedFor string
		// realIP is the X-Real-IP header value.
		realIP string
		// want is the expected client IP string.
		want string
	}{
		{
			name:         "single X-Forwarded-For",
			peerAddr:     "10.0.0.1:1234",
			forwardedFor: "203.0.113.10",
			realIP:       "198.51.100.20",
			want:         "203.0.113.10",
		},
		{
			name:         "comma chain returns leftmost trimmed",
			peerAddr:     "10.0.0.1:1234",
			forwardedFor: " 203.0.113.10 , 198.51.100.20 ",
			realIP:       "",
			want:         "203.0.113.10",
		},
		{
			name:         "empty X-Forwarded-For falls back to X-Real-IP",
			peerAddr:     "10.0.0.1:1234",
			forwardedFor: "   ",
			realIP:       "198.51.100.20",
			want:         "198.51.100.20",
		},
		{
			name:         "both empty falls back to peer host with port stripped",
			peerAddr:     "10.0.0.1:1234",
			forwardedFor: "",
			realIP:       "",
			want:         "10.0.0.1",
		},
		{
			name:         "peer value without port passed through",
			peerAddr:     "10.0.0.1",
			forwardedFor: "",
			realIP:       "",
			want:         "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, resolveClientIP(tt.peerAddr, tt.forwardedFor, tt.realIP))
		})
	}
}
