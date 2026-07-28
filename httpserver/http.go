package httpserver

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	golog "log"

	"github.com/rs/zerolog/log"
	"github.com/trivago/go-bootstrap/logging"
)

const (
	// healthPath is the liveness probe endpoint.
	healthPath = "/healthz"
	// readyPath is the readiness probe endpoint.
	readyPath = "/readyz"
)

// HTTPServer wraps net/http.Server and exposes the shared Server lifecycle.
type HTTPServer struct {
	// Server is the underlying net/http server. Callers may configure
	// additional fields before Listen.
	Server *http.Server
}

// defaultDisableAccessLogFor is the access-log exclusion list used by the
// convenience constructors.
var defaultDisableAccessLogFor = []string{healthPath, readyPath}

// New creates a net/http server with the given port, probe checks, and
// handler. Access logs for /healthz and /readyz are disabled by default.
func New(port int, health, ready Check, handler http.Handler) (*HTTPServer, error) {
	return NewWithConfig(Config{
		Port:                port,
		Health:              health,
		Ready:               ready,
		DisableAccessLogFor: slices.Clone(defaultDisableAccessLogFor),
	}, handler)
}

// NewWithConfig creates a net/http server with fine-grained configuration.
// The handler may be any net/http-compatible router such as Gin or the
// standard ServeMux.
func NewWithConfig(config Config, handler http.Handler) (*HTTPServer, error) {
	syncLogThresholds()

	tlsConfig, err := buildTLSConfig(config)
	if err != nil {
		return nil, err
	}

	wrapped := wrapHTTPHandler(config, handler)

	return &HTTPServer{
		Server: &http.Server{
			Addr:      resolveAddr(config),
			Handler:   wrapped,
			ErrorLog:  golog.New(logging.ErrorLogWriter{}, "", 0),
			TLSConfig: tlsConfig,
		},
	}, nil
}

// ListenAndServe starts the server and blocks until it stops. Expected
// shutdown results are normalized to a nil error.
func (s *HTTPServer) ListenAndServe() error {
	var err error
	if s.Server.TLSConfig != nil {
		err = s.Server.ListenAndServeTLS("", "")
	} else {
		err = s.Server.ListenAndServe()
	}

	if err == nil || errors.Is(err, http.ErrServerClosed) {
		if errors.Is(err, http.ErrServerClosed) {
			log.Warn().Msg("HTTP server was instructed to close")
		}
		return nil
	}
	return err
}

// Shutdown gracefully stops the underlying net/http server.
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}

// wrapHTTPHandler installs recovery, access logging, and probe routes around
// the application handler.
func wrapHTTPHandler(config Config, handler http.Handler) http.Handler {
	if handler == nil {
		handler = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	}

	withProbes := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			switch request.URL.Path {
			case healthPath:
				writer.WriteHeader(probeStatus(request.Context(), config.Health))
				return
			case readyPath:
				writer.WriteHeader(probeStatus(request.Context(), config.Ready))
				return
			}
		}
		handler.ServeHTTP(writer, request)
	})

	withRecovery := recoverHTTP(withProbes)
	return accessLogHTTP(config.DisableAccessLogFor, withRecovery)
}

// statusRecorder captures the response status while preserving Unwrap for
// optional interfaces on the underlying ResponseWriter.
type statusRecorder struct {
	http.ResponseWriter
	// status is the HTTP status written to the response.
	status int
	// written reports whether WriteHeader has already been called.
	written bool
}

// WriteHeader records the status code and forwards it.
func (recorder *statusRecorder) WriteHeader(statusCode int) {
	if recorder.written {
		return
	}
	recorder.status = statusCode
	recorder.written = true
	recorder.ResponseWriter.WriteHeader(statusCode)
}

// Write ensures a default status is recorded before writing the body.
func (recorder *statusRecorder) Write(body []byte) (int, error) {
	if !recorder.written {
		recorder.WriteHeader(http.StatusOK)
	}
	return recorder.ResponseWriter.Write(body)
}

// Unwrap exposes the underlying ResponseWriter for http.ResponseController.
func (recorder *statusRecorder) Unwrap() http.ResponseWriter {
	return recorder.ResponseWriter
}

// accessLogHTTP logs each request after the next handler returns.
func accessLogHTTP(ignorePaths []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{
			ResponseWriter: writer,
			status:         http.StatusOK,
		}

		next.ServeHTTP(recorder, request)

		writeAccessLog(
			ignorePaths,
			request.URL.Path,
			recorder.status,
			request.Method,
			request.RemoteAddr,
			time.Since(started),
			"",
		)
	})
}

// recoverHTTP converts panics into HTTP 500 responses.
func recoverHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error().Interface("panic", recovered).Msg("Recovered from panic")
				http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(writer, request)
	})
}
