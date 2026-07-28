package httpserver

import (
	golog "log"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/trivago/go-bootstrap/v2/logging"
)

// ginModeOnce ensures Gin console and mode settings run once per process.
var ginModeOnce sync.Once

// GinConfig provides configuration options for a Gin-backed HTTP server.
type GinConfig struct {
	// Port defines the HTTP port the server will listen on.
	// Defaults to 8080, or 8443 for TLS when left empty.
	Port int

	// Health defines the handler for the /healthz endpoint.
	// When nil, AlwaysOk is used.
	Health gin.HandlerFunc

	// Ready defines the handler for the /readyz endpoint.
	// When nil, AlwaysOk is used.
	Ready gin.HandlerFunc

	// DisableAccessLogFor defines a list of paths for which the access log
	// will not be written. The path must be a full match to be disabled.
	DisableAccessLogFor []string

	// InitRoutes defines a function that will be called to configure routes
	// on this server. Use it to define the handler for your routes.
	InitRoutes func(router *gin.Engine)

	// PathTLSCert points to the TLS certificate file to use for HTTPS.
	// When left empty, the server will not use TLS.
	PathTLSCert string

	// PathTLSKey points to the TLS key file to use for HTTPS.
	// When left empty, the server will not use TLS.
	PathTLSKey string

	// CertCacheDuration defines how long a certificate will be cached in
	// memory before it is reloaded from disk. Default duration is 7 days.
	CertCacheDuration time.Duration
}

// AlwaysOk is a Gin handler that always returns HTTP 200 OK.
func AlwaysOk(ctx *gin.Context) {
	ctx.Status(http.StatusOK)
	ctx.Writer.WriteHeaderNow()
}

// NewGin creates a Gin-backed HTTP server with the given port, probe
// handlers, and route initializer. Access logs for /healthz and /readyz are
// disabled by default.
func NewGin(
	port int,
	health, ready gin.HandlerFunc,
	initRoutes func(router *gin.Engine),
) (*HTTPServer, error) {
	return NewGinWithConfig(GinConfig{
		Port:                port,
		Health:              health,
		Ready:               ready,
		InitRoutes:          initRoutes,
		DisableAccessLogFor: slices.Clone(defaultDisableAccessLogFor),
	})
}

// NewGinWithConfig creates a Gin-backed HTTP server with fine-grained
// configuration. The package creates the Gin engine and calls InitRoutes to
// register application routes.
func NewGinWithConfig(config GinConfig) (*HTTPServer, error) {
	syncLogThresholds()
	configureGinMode()

	tlsConfig, err := buildTLSConfig(config.asConfig())
	if err != nil {
		return nil, err
	}

	router := gin.New()
	router.Use(newGinZeroLogLogger(config.DisableAccessLogFor), gin.Recovery())

	health := config.Health
	if health == nil {
		health = AlwaysOk
	}
	ready := config.Ready
	if ready == nil {
		ready = AlwaysOk
	}

	router.GET(healthPath, health)
	router.GET(readyPath, ready)

	if config.InitRoutes != nil {
		config.InitRoutes(router)
	}

	return &HTTPServer{
		Server: &http.Server{
			Addr:      resolveAddr(config.asConfig()),
			Handler:   router,
			ErrorLog:  golog.New(logging.ErrorLogWriter{}, "", 0),
			TLSConfig: tlsConfig,
		},
	}, nil
}

// asConfig maps Gin-specific settings onto the shared Config used for port
// and TLS helpers.
func (config GinConfig) asConfig() Config {
	return Config{
		Port:                config.Port,
		DisableAccessLogFor: config.DisableAccessLogFor,
		PathTLSCert:         config.PathTLSCert,
		PathTLSKey:          config.PathTLSKey,
		CertCacheDuration:   config.CertCacheDuration,
	}
}

// configureGinMode disables console color and selects Gin release mode when
// the global zerolog level is above debug.
func configureGinMode() {
	ginModeOnce.Do(func() {
		gin.DisableConsoleColor()
		gin.DefaultWriter = logging.DebugLogWriter{}

		if zerolog.GlobalLevel() > zerolog.DebugLevel {
			gin.SetMode(gin.ReleaseMode)
		}
	})
}

// newGinZeroLogLogger returns Gin middleware that writes structured access
// logs through zerolog.
func newGinZeroLogLogger(ignorePaths []string) gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Output: logging.NullWriter{}, // output is done through the formatter
		Formatter: func(params gin.LogFormatterParams) string {
			writeAccessLog(
				ignorePaths,
				params.Path,
				params.StatusCode,
				params.Method,
				params.ClientIP,
				params.Latency,
				params.ErrorMessage,
			)
			return ""
		},
	})
}
