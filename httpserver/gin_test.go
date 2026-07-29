package httpserver

import (
	"io"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewGinBehaviors covers InitRoutes delegation, default probes, custom
// Gin probe status codes, and a nil InitRoutes callback.
func TestNewGinBehaviors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		// name identifies the test case.
		name string
		// config is the Gin server configuration under test.
		config GinConfig
		// path is the request path to send.
		path string
		// wantStatus is the expected response status.
		wantStatus int
		// wantBody is an optional expected response body.
		wantBody string
	}{
		{
			name:       "default health",
			config:     GinConfig{},
			path:       "/healthz",
			wantStatus: http.StatusOK,
		},
		{
			name:       "default ready",
			config:     GinConfig{},
			path:       "/readyz",
			wantStatus: http.StatusOK,
		},
		{
			name: "custom health status",
			config: GinConfig{
				Health: func(ctx *gin.Context) {
					ctx.Status(http.StatusGone)
					ctx.Writer.WriteHeaderNow()
				},
			},
			path:       "/healthz",
			wantStatus: http.StatusGone,
		},
		{
			name: "init routes delegation",
			config: GinConfig{
				InitRoutes: func(router *gin.Engine) {
					router.GET("/hello", func(ctx *gin.Context) {
						ctx.String(http.StatusOK, "world")
					})
				},
			},
			path:       "/hello",
			wantStatus: http.StatusOK,
			wantBody:   "world",
		},
		{
			name: "nil init routes keeps probes",
			config: GinConfig{
				Health: AlwaysOk,
				Ready:  AlwaysOk,
			},
			path:       "/healthz",
			wantStatus: http.StatusOK,
		},
		{
			name: "explicit always ok ready",
			config: GinConfig{
				Ready: AlwaysOk,
			},
			path:       "/readyz",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv, err := NewGinWithConfig(tt.config)
			require.NoError(t, err)

			listener := startHTTPServer(t, srv)
			defer func() {
				_ = listener.Close()
			}()

			response, err := http.Get("http://" + listener.Addr().String() + tt.path)
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

// TestNewGinConvenience verifies the port-based constructor wires probes and
// disables access logs for health endpoints by default.
func TestNewGinConvenience(t *testing.T) {
	t.Parallel()

	srv, err := NewGin(0, AlwaysOk, AlwaysOk, func(router *gin.Engine) {
		router.GET("/hello", func(ctx *gin.Context) {
			ctx.String(http.StatusOK, "world")
		})
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"/healthz", "/readyz"}, defaultDisableAccessLogFor)

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
			name:       "health",
			path:       "/healthz",
			wantStatus: http.StatusOK,
		},
		{
			name:       "ready",
			path:       "/readyz",
			wantStatus: http.StatusOK,
		},
		{
			name:       "route",
			path:       "/hello",
			wantStatus: http.StatusOK,
			wantBody:   "world",
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

// TestGinAsHTTPHandler verifies that a caller-built Gin engine still works
// with NewWithConfig and Check-based probes.
func TestGinAsHTTPHandler(t *testing.T) {
	t.Parallel()

	router := gin.New()
	router.GET("/hello", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "world")
	})

	srv, err := NewWithConfig(Config{
		Health:              CheckOK,
		Ready:               CheckOK,
		DisableAccessLogFor: []string{"/healthz", "/readyz"},
	}, router)
	require.NoError(t, err)

	listener := startHTTPServer(t, srv)
	baseURL := "http://" + listener.Addr().String()

	response, err := http.Get(baseURL + "/hello")
	require.NoError(t, err)
	defer func() {
		_ = response.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, response.StatusCode)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, "world", string(body))
}

// TestNewConvenience verifies the net/http convenience constructor.
func TestNewConvenience(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte("ok"))
	})

	srv, err := New(0, CheckOK, CheckOK, mux)
	require.NoError(t, err)

	listener := startHTTPServer(t, srv)
	response, err := http.Get("http://" + listener.Addr().String() + "/hello")
	require.NoError(t, err)
	defer func() {
		_ = response.Body.Close()
	}()

	assert.Equal(t, http.StatusCreated, response.StatusCode)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, "ok", string(body))
}
