package httpserver

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
}

// TestTLS verifies that a TLS-enabled net/http server serves /healthz over
// a local ephemeral listener.
func TestTLS(t *testing.T) {
	t.Parallel()

	srv, err := NewWithConfig(Config{
		PathTLSCert: "../hack/tls.cert",
		PathTLSKey:  "../hack/tls.key",
	}, http.NewServeMux())
	require.NoError(t, err)
	require.NotNil(t, srv.Server.TLSConfig)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		_ = srv.Server.ServeTLS(listener, "", "")
	}()
	t.Cleanup(func() {
		_ = srv.Shutdown(context.Background())
		_ = listener.Close()
	})

	addr := listener.Addr().String()
	client := &http.Client{Transport: &http.Transport{
		DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
			return net.Dial(network, addr)
		},
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}

	resp, err := client.Get("https://localhost/healthz")
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestNewTLSConfig verifies TLS configuration is applied without listening.
func TestNewTLSConfig(t *testing.T) {
	t.Parallel()

	srv, err := NewWithConfig(Config{
		PathTLSCert: "../hack/tls.cert",
		PathTLSKey:  "../hack/tls.key",
	}, http.NewServeMux())

	require.NoError(t, err)
	assert.Equal(t, ":8443", srv.Server.Addr)
	assert.NotNil(t, srv.Server.TLSConfig)
	assert.NotNil(t, srv.Server.TLSConfig.GetCertificate)
}

// TestNewFastHTTPTLSConfig verifies fasthttp TLS configuration.
func TestNewFastHTTPTLSConfig(t *testing.T) {
	t.Parallel()

	srv, err := NewFastHTTPWithConfig(Config{
		PathTLSCert: "../hack/tls.cert",
		PathTLSKey:  "../hack/tls.key",
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, ":8443", srv.addr)
	assert.True(t, srv.useTLS)
	assert.NotNil(t, srv.Server.TLSConfig)
}

// TestHTTPHandlerBehaviors covers probes, delegation, and recovery for
// net/http servers.
func TestHTTPHandlerBehaviors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		// name identifies the test case.
		name string
		// config is the server configuration under test.
		config Config
		// handler is the application handler.
		handler http.Handler
		// method is the HTTP method to send.
		method string
		// path is the request path to send.
		path string
		// wantStatus is the expected response status.
		wantStatus int
		// wantBody is an optional expected response body.
		wantBody string
	}{
		{
			name:       "default health",
			config:     Config{},
			method:     http.MethodGet,
			path:       "/healthz",
			wantStatus: http.StatusOK,
		},
		{
			name:       "default ready",
			config:     Config{},
			method:     http.MethodGet,
			path:       "/readyz",
			wantStatus: http.StatusOK,
		},
		{
			name: "failed health check",
			config: Config{
				Health: func(context.Context) error {
					return errors.New("unhealthy")
				},
			},
			method:     http.MethodGet,
			path:       "/healthz",
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name: "failed ready check",
			config: Config{
				Ready: func(context.Context) error {
					return errors.New("not ready")
				},
			},
			method:     http.MethodGet,
			path:       "/readyz",
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:   "delegates to handler",
			config: Config{},
			handler: http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusCreated)
				_, _ = writer.Write([]byte("ok"))
			}),
			method:     http.MethodGet,
			path:       "/api",
			wantStatus: http.StatusCreated,
			wantBody:   "ok",
		},
		{
			name:   "recovers from panic",
			config: Config{},
			handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				panic("boom")
			}),
			method:     http.MethodGet,
			path:       "/panic",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv, err := NewWithConfig(tt.config, tt.handler)
			require.NoError(t, err)

			listener := startHTTPServer(t, srv)
			defer func() {
				_ = listener.Close()
			}()

			request, err := http.NewRequest(tt.method, "http://"+listener.Addr().String()+tt.path, nil)
			require.NoError(t, err)

			response, err := http.DefaultClient.Do(request)
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

// TestFastHTTPHandlerBehaviors covers probes, delegation, and recovery for
// fasthttp servers.
func TestFastHTTPHandlerBehaviors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		// name identifies the test case.
		name string
		// config is the server configuration under test.
		config Config
		// handler is the application handler.
		handler fasthttp.RequestHandler
		// method is the HTTP method to send.
		method string
		// path is the request path to send.
		path string
		// wantStatus is the expected response status.
		wantStatus int
		// wantBody is an optional expected response body.
		wantBody string
	}{
		{
			name:       "default health",
			config:     Config{},
			method:     fasthttp.MethodGet,
			path:       "/healthz",
			wantStatus: fasthttp.StatusOK,
		},
		{
			name:       "default ready",
			config:     Config{},
			method:     fasthttp.MethodGet,
			path:       "/readyz",
			wantStatus: fasthttp.StatusOK,
		},
		{
			name: "failed health check",
			config: Config{
				Health: func(context.Context) error {
					return errors.New("unhealthy")
				},
			},
			method:     fasthttp.MethodGet,
			path:       "/healthz",
			wantStatus: fasthttp.StatusServiceUnavailable,
		},
		{
			name: "failed ready check",
			config: Config{
				Ready: func(context.Context) error {
					return errors.New("not ready")
				},
			},
			method:     fasthttp.MethodGet,
			path:       "/readyz",
			wantStatus: fasthttp.StatusServiceUnavailable,
		},
		{
			name:   "delegates to handler",
			config: Config{},
			handler: func(ctx *fasthttp.RequestCtx) {
				ctx.SetStatusCode(fasthttp.StatusCreated)
				ctx.SetBodyString("ok")
			},
			method:     fasthttp.MethodGet,
			path:       "/api",
			wantStatus: fasthttp.StatusCreated,
			wantBody:   "ok",
		},
		{
			name:   "recovers from panic",
			config: Config{},
			handler: func(ctx *fasthttp.RequestCtx) {
				panic("boom")
			},
			method:     fasthttp.MethodGet,
			path:       "/panic",
			wantStatus: fasthttp.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv, err := NewFastHTTPWithConfig(tt.config, tt.handler)
			require.NoError(t, err)

			client := startFastHTTPServer(t, srv)

			status, body, err := client.Get(nil, "http://fasthttp"+tt.path)
			if tt.method != fasthttp.MethodGet {
				t.Fatalf("unsupported method %s", tt.method)
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, status)
			if len(tt.wantBody) > 0 {
				assert.Equal(t, tt.wantBody, string(body))
			}
		})
	}
}

// TestHTTPServerLifecycleNormalizesClosedError verifies Shutdown results are
// treated as success by ListenAndServe.
func TestHTTPServerLifecycleNormalizesClosedError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	srv, err := NewWithConfig(Config{}, http.NewServeMux())
	require.NoError(t, err)
	srv.Server.Addr = addr

	done := make(chan error, 1)
	go func() {
		done <- srv.ListenAndServe()
	}()

	require.Eventually(t, func() bool {
		conn, dialErr := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if dialErr != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, srv.Shutdown(context.Background()))
	assert.NoError(t, <-done)
}

// TestFastHTTPServerLifecycle verifies Shutdown stops ListenAndServe.
func TestFastHTTPServerLifecycle(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	srv, err := NewFastHTTPWithConfig(Config{}, nil)
	require.NoError(t, err)
	srv.addr = addr

	done := make(chan error, 1)
	go func() {
		done <- srv.ListenAndServe()
	}()

	require.Eventually(t, func() bool {
		conn, dialErr := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if dialErr != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, srv.Shutdown(context.Background()))
	assert.NoError(t, <-done)
}

// TestResolvePortDefaults verifies default port selection.
func TestResolvePortDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		// name identifies the test case.
		name string
		// config is the input configuration.
		config Config
		// want is the expected port.
		want int
	}{
		{name: "plain default", config: Config{}, want: 8080},
		{name: "explicit port", config: Config{Port: 9090}, want: 9090},
		{name: "tls default", config: Config{PathTLSCert: "c", PathTLSKey: "k"}, want: 8443},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, resolvePort(tt.config))
		})
	}
}

// startHTTPServer starts srv on a local listener and returns it.
func startHTTPServer(t *testing.T, srv *HTTPServer) net.Listener {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		_ = srv.Server.Serve(listener)
	}()

	t.Cleanup(func() {
		_ = srv.Shutdown(context.Background())
	})

	return listener
}

// startFastHTTPServer starts srv on an in-memory listener and returns a
// client dialing that listener.
func startFastHTTPServer(t *testing.T, srv *FastHTTPServer) *fasthttp.Client {
	t.Helper()

	listener := fasthttputil.NewInmemoryListener()
	go func() {
		_ = srv.Server.Serve(listener)
	}()

	t.Cleanup(func() {
		_ = srv.Shutdown(context.Background())
		_ = listener.Close()
	})

	return &fasthttp.Client{
		Dial: func(string) (net.Conn, error) {
			return listener.Dial()
		},
	}
}
