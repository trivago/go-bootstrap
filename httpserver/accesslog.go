package httpserver

import (
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	jww "github.com/spf13/jwalterweatherman" // See https://github.com/spf13/viper/issues/1152
)

const (
	// forwardedForHeader is the HTTP header used by proxies to convey the
	// original client address chain.
	forwardedForHeader = "X-Forwarded-For"
	// realIPHeader is the HTTP header used by some proxies to convey the
	// original client address.
	realIPHeader = "X-Real-IP"
)

// logThresholdsOnce ensures jwalterweatherman is aligned once per process.
var logThresholdsOnce sync.Once

// syncLogThresholds aligns jwalterweatherman verbosity with zerolog.
func syncLogThresholds() {
	logThresholdsOnce.Do(func() {
		switch zerolog.GlobalLevel() {
		default:
			fallthrough
		case zerolog.DebugLevel:
			jww.SetLogThreshold(jww.LevelDebug)
			jww.SetStdoutThreshold(jww.LevelDebug)
		case zerolog.InfoLevel:
			jww.SetLogThreshold(jww.LevelInfo)
			jww.SetStdoutThreshold(jww.LevelInfo)
		case zerolog.WarnLevel:
			jww.SetLogThreshold(jww.LevelWarn)
			jww.SetStdoutThreshold(jww.LevelWarn)
		case zerolog.ErrorLevel:
			jww.SetLogThreshold(jww.LevelError)
			jww.SetStdoutThreshold(jww.LevelError)
		}
	})
}

// shouldSkipAccessLog reports whether path is excluded from access logging.
// The match is exact and case-sensitive.
func shouldSkipAccessLog(ignorePaths []string, path string) bool {
	return slices.Contains(ignorePaths, path)
}

// resolveClientIP returns the client IP for an access log entry. Proxy
// headers take precedence over the peer address, mirroring
// gin.Context.ClientIP with Gin's default trusted proxies. Both headers are
// client controlled, so a deployment without a header-stripping proxy can
// see spoofed values.
func resolveClientIP(peerAddr, forwardedFor, realIP string) string {
	if trimmed := strings.TrimSpace(forwardedFor); len(trimmed) > 0 {
		leftmost := strings.TrimSpace(strings.Split(trimmed, ",")[0])
		if len(leftmost) > 0 {
			return leftmost
		}
	}

	if trimmed := strings.TrimSpace(realIP); len(trimmed) > 0 {
		return trimmed
	}

	host, _, err := net.SplitHostPort(peerAddr)
	if err != nil {
		return peerAddr
	}
	return host
}

// writeAccessLog emits a structured access log entry for one request.
func writeAccessLog(
	ignorePaths []string,
	path string,
	status int,
	method string,
	clientIP string,
	latency time.Duration,
	errMsg string,
) {
	if shouldSkipAccessLog(ignorePaths, path) {
		return
	}

	var event *zerolog.Event
	switch {
	case len(errMsg) > 0:
		event = log.Warn().Err(fmt.Errorf("%s", errMsg))
	case status >= 500:
		event = log.Warn()
	default:
		event = log.Info()
	}

	event.Str("latency", latency.String()).
		Int("status", status).
		Str("clientip", clientIP).
		Str("method", method).
		Str("path", path).
		Send()
}
