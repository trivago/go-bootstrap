package httpserver

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	jww "github.com/spf13/jwalterweatherman" // See https://github.com/spf13/viper/issues/1152
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
func shouldSkipAccessLog(ignorePaths []string, path string) bool {
	for _, ignored := range ignorePaths {
		if strings.EqualFold(path, ignored) {
			return true
		}
	}
	return false
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
