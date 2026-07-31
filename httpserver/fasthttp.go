package httpserver

import (
	"context"
	"slices"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

// FastHTTPServer wraps fasthttp.Server and exposes the shared Server
// lifecycle.
type FastHTTPServer struct {
	// Server is the underlying fasthttp server. Callers may configure
	// additional fields before Listen.
	Server *fasthttp.Server

	// addr is the listen address used by ListenAndServe.
	addr string

	// useTLS reports whether TLS should be enabled.
	useTLS bool
}

// NewFastHTTP creates a fasthttp server with the given port, probe checks,
// and handler. Access logs for /healthz and /readyz are disabled by default.
func NewFastHTTP(
	port int,
	health, ready Check,
	handler fasthttp.RequestHandler,
) (*FastHTTPServer, error) {
	return NewFastHTTPWithConfig(Config{
		Port:                port,
		Health:              health,
		Ready:               ready,
		DisableAccessLogFor: slices.Clone(defaultDisableAccessLogFor),
	}, handler)
}

// NewFastHTTPWithConfig creates a native fasthttp server with fine-grained
// configuration, shared probes, access logging, recovery, and optional TLS.
func NewFastHTTPWithConfig(
	config Config,
	handler fasthttp.RequestHandler,
) (*FastHTTPServer, error) {
	syncLogThresholds()

	tlsConfig, err := buildTLSConfig(config)
	if err != nil {
		return nil, err
	}

	wrapped := wrapFastHTTPHandler(config, handler)

	return &FastHTTPServer{
		Server: &fasthttp.Server{
			Handler:   wrapped,
			TLSConfig: tlsConfig,
			Logger:    &fastHTTPLogger{},
		},
		addr:   resolveAddr(config),
		useTLS: tlsConfig != nil,
	}, nil
}

// ListenAndServe starts the server and blocks until it stops. Expected
// shutdown results are normalized to a nil error.
func (s *FastHTTPServer) ListenAndServe() error {
	var err error
	if s.useTLS {
		err = s.Server.ListenAndServeTLS(s.addr, "", "")
	} else {
		err = s.Server.ListenAndServe(s.addr)
	}

	if err == nil {
		log.Warn().Msg("HTTP server was instructed to close")
	}
	return err
}

// Shutdown gracefully stops the underlying fasthttp server.
func (s *FastHTTPServer) Shutdown(ctx context.Context) error {
	return s.Server.ShutdownWithContext(ctx)
}

// wrapFastHTTPHandler installs recovery, access logging, and probe routes
// around the application handler.
func wrapFastHTTPHandler(config Config, handler fasthttp.RequestHandler) fasthttp.RequestHandler {
	if handler == nil {
		handler = func(ctx *fasthttp.RequestCtx) {}
	}

	withProbes := func(ctx *fasthttp.RequestCtx) {
		if ctx.IsGet() {
			switch string(ctx.Path()) {
			case healthPath:
				ctx.SetStatusCode(probeStatus(ctx, config.Health))
				return
			case readyPath:
				ctx.SetStatusCode(probeStatus(ctx, config.Ready))
				return
			}
		}
		handler(ctx)
	}

	withRecovery := recoverFastHTTP(withProbes)
	return accessLogFastHTTP(config.DisableAccessLogFor, withRecovery)
}

// accessLogFastHTTP logs each request after the next handler returns.
func accessLogFastHTTP(ignorePaths []string, next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		started := time.Now()
		next(ctx)
		writeAccessLog(
			ignorePaths,
			string(ctx.Path()),
			ctx.Response.StatusCode(),
			string(ctx.Method()),
			resolveClientIP(
				ctx.RemoteIP().String(),
				string(ctx.Request.Header.Peek(forwardedForHeader)),
				string(ctx.Request.Header.Peek(realIPHeader)),
			),
			time.Since(started),
			"",
		)
	}
}

// recoverFastHTTP converts panics into HTTP 500 responses.
func recoverFastHTTP(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error().Interface("panic", recovered).Msg("Recovered from panic")
				ctx.Error(
					fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
					fasthttp.StatusInternalServerError,
				)
			}
		}()
		next(ctx)
	}
}

// fastHTTPLogger adapts fasthttp server logs to zerolog.
type fastHTTPLogger struct{}

// Printf writes a fasthttp log line as an error event.
func (fastHTTPLogger) Printf(format string, args ...any) {
	log.Error().Msgf(format, args...)
}
