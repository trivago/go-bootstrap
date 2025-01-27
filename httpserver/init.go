package httpserver

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	jww "github.com/spf13/jwalterweatherman" // See https://github.com/spf13/viper/issues/1152

	"github.com/trivago/go-bootstrap/logging"
)

func init() {
	gin.DisableConsoleColor()
	gin.DefaultWriter = logging.DebugLogWriter{}
}

func newZeroLogLogger(ignorePaths []string) gin.HandlerFunc {
	// Set log level and verbosity

	switch zerolog.GlobalLevel() {
	default:
		fallthrough
	case zerolog.DebugLevel:
		jww.SetLogThreshold(jww.LevelDebug)
		jww.SetStdoutThreshold(jww.LevelDebug)
	case zerolog.InfoLevel:
		jww.SetLogThreshold(jww.LevelInfo)
		jww.SetStdoutThreshold(jww.LevelInfo)
		gin.SetMode(gin.ReleaseMode)
	case zerolog.WarnLevel:
		jww.SetLogThreshold(jww.LevelWarn)
		jww.SetStdoutThreshold(jww.LevelWarn)
		gin.SetMode(gin.ReleaseMode)
	case zerolog.ErrorLevel:
		jww.SetLogThreshold(jww.LevelError)
		jww.SetStdoutThreshold(jww.LevelError)
		gin.SetMode(gin.ReleaseMode)
	}

	// Create a new, zerlog based gin logger
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Output: logging.NullWriter{}, // output is done through the formatter
		Formatter: func(e gin.LogFormatterParams) string {
			var event *zerolog.Event

			for _, p := range ignorePaths {
				if strings.EqualFold(e.Path, p) {
					return ""
				}
			}

			switch {
			case len(e.ErrorMessage) > 0:
				event = log.Warn().Err(fmt.Errorf("%s", e.ErrorMessage))
			case e.StatusCode >= 500:
				event = log.Warn()
			default:
				event = log.Info()
			}

			event.Str("latency", e.Latency.String()).
				Int("status", e.StatusCode).
				Str("clientip", e.ClientIP).
				Str("method", e.Method).
				Str("path", e.Path).Send()

			return ""
		},
	})
}
