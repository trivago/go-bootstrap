package httpserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGinFramework verifies that a Gin engine works as the application
// handler behind New, including shared probe endpoints.
func TestGinFramework(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/hello", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "world")
	})
	router.GET("/fail", func(ctx *gin.Context) {
		ctx.Status(http.StatusBadRequest)
	})

	srv, err := New(Config{
		Health: AlwaysOk,
		Ready: func(context.Context) error {
			return errors.New("not ready")
		},
		DisableAccessLogFor: []string{"/healthz", "/readyz"},
	}, router)
	require.NoError(t, err)

	listener := startHTTPServer(t, srv)
	baseURL := "http://" + listener.Addr().String()

	tests := []struct {
		// name identifies the test case.
		name string
		// path is the request path to send.
		path string
		// wantStatus is the expected response status.
		wantStatus int
		// wantBody is an optional expected response body.
		wantBody string
	}{
		{
			name:       "gin route",
			path:       "/hello",
			wantStatus: http.StatusOK,
			wantBody:   "world",
		},
		{
			name:       "gin status",
			path:       "/fail",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "health probe",
			path:       "/healthz",
			wantStatus: http.StatusOK,
		},
		{
			name:       "ready probe failure",
			path:       "/readyz",
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response, err := http.Get(baseURL + tt.path)
			require.NoError(t, err)
			defer func() {
				_ = response.Body.Close()
			}()

			assert.Equal(t, tt.wantStatus, response.StatusCode)
			if len(tt.wantBody) > 0 {
				body, err := io.ReadAll(response.Body)
				require.NoError(t, err)
				assert.Equal(t, tt.wantBody, string(body))
			}
		})
	}
}
