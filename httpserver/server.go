package httpserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	golog "log"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/trivago/go-bootstrap/logging"
)

type Config struct {
	// Port defines the HTTP port the server will be listen to.
	// Defaults to 8080 or 8443 for TLS when left empty
	Port int

	// Health defines the handler for the /healthz endpoint.
	Health gin.HandlerFunc

	// Ready defines the handler for the /readyz endpoint.
	Ready gin.HandlerFunc

	// InitRoutes defines a function that will be called to configure routes on
	// this server. Use it to define the handler for your routes.
	InitRoutes func(router *gin.Engine)

	// PathTLSCert points to the TLS certificate file to use for HTTPS.
	// When left empty, the server will not use TLS.
	PathTLSCert string

	// PathTLSKey points to the TLS key file to use for HTTPS.
	// When left empty, the server will not use TLS.
	PathTLSKey string

	// CertCacheDuration defines how long a certificate will be cached in memory,
	// before it is reloaded from disk. Default duration is 7 days.
	CertCacheDuration time.Duration
}

// AlwaysOk is a gin handler that always returns a 200 OK.
func AlwaysOk(c *gin.Context) {
	c.Status(http.StatusOK)
	c.Writer.WriteHeaderNow()
}

// New creates a new HTTP server with the given health and ready handlers.
// Pass an initRoutes function to configure routes on this server.
func New(port int, health, ready gin.HandlerFunc, initRoutes func(router *gin.Engine)) *http.Server {
	return NewWithConfig(Config{
		Port:       port,
		Health:     health,
		Ready:      ready,
		InitRoutes: initRoutes,
	})
}

// NewWithConfig allows a more fine-grained configuration of the HTTP server.
// Use it to e.g. create a server with TLS enabled.
func NewWithConfig(config Config) *http.Server {
	router := gin.New()
	router.Use(newZeroLogLogger([]string{"/healthz", "/readyz"}), gin.Recovery())

	// Setup routes
	if config.Health == nil {
		router.GET("/healthz", AlwaysOk)
	} else {
		router.GET("/healthz", config.Health)
	}

	if config.Ready == nil {
		router.GET("/readyz", AlwaysOk)
	} else {
		router.GET("/readyz", config.Ready)
	}

	if config.InitRoutes != nil {
		config.InitRoutes(router)
	}

	// Setup port and TLS
	port := 8080
	if config.Port > 0 {
		port = config.Port
	} else if len(config.PathTLSCert) > 0 || len(config.PathTLSKey) > 0 {
		port = 8443
	}

	var tlsConfig *tls.Config
	if len(config.PathTLSCert) > 0 && len(config.PathTLSKey) > 0 {
		reloadDuration := time.Hour * 24 * 7 // 7 days
		if config.CertCacheDuration > 0 {
			reloadDuration = config.CertCacheDuration
		}

		// Create a certificate handler that is reloading the certificate from disk.
		// This is required to support certificate rotation.
		cert := newFileBasedCert(config.PathTLSCert, config.PathTLSKey, reloadDuration)
		tlsConfig = &tls.Config{
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return cert.GetCertificate()
			},
		}
	}

	return &http.Server{
		Addr:      fmt.Sprintf(":%d", port),
		Handler:   router,
		ErrorLog:  golog.New(logging.ErrorLogWriter{}, "", 0),
		TLSConfig: tlsConfig,
	}
}

// Listen starts the given HTTP server and blocks until a stop signal like SIGINT or SIGTERM is received.
// Use the signalHandler if you need to react on any of these signals.
func Listen(srv *http.Server, signalHandler func(os.Signal)) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGQUIT)

	// Launch server async, as ListenAndServeTLS is blocking.
	go func() {
		log.Info().Msg("Starting listener")

		// This call is blocking
		if err := srv.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				log.Warn().Msg("HTTP server was instructed to close")
			} else {
				log.Error().Err(err).Msg("Failed to start HTTP server")
			}
		}

		log.Info().Msg("Listener exited")
		signal.Stop(signalChan)
		close(signalChan)
	}()

	// React on external OS signals to trigger a shutdown.
	// If the channel was closed, the server did not start

	if sig, isOpen := <-signalChan; isOpen {
		log.Info().Msgf("Received signal: %s", sig.String())
		if signalHandler != nil {
			signalHandler(sig)
		}

		log.Info().Msg("Stopping HTTP server")

		// This call is blocking and unblocks the server go routine
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Error().Err(err).Msg("Graceful shutdown failed")
		}
	}
}
