package logging

import (
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ErrorLogWriter is an adapter between the standard library's log package and zerolog.
// Everything will be logged as an error.
type ErrorLogWriter struct{}

// DebugLogWriter is an adapter between the standard library's log package and zerolog.
// Everything will be logged as debug.
type DebugLogWriter struct{}

// NullWriter is an adapter between the standard library's log package and zerolog.
// All messages will be discarded.
type NullWriter struct{}

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.LevelFieldName = "severity"
}

// Write of DebugWriter log the output as error
func (w ErrorLogWriter) Write(p []byte) (n int, err error) {
	log.Error().Msg(string(p))
	return len(p), nil
}

// Write of DebugWriter log the output as debug
func (w DebugLogWriter) Write(p []byte) (n int, err error) {
	log.Debug().Msg(string(p))
	return len(p), nil
}

// Write of NullWriter drops the output
func (w NullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// SetLogLevel defines the zerolog level based on commonly used loglevel strings.
func SetLogLevel(logLevel string) {
	switch strings.ToLower(logLevel) {
	default:
		fallthrough
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn", "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error", "critical":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}
}
