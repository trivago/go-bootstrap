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
