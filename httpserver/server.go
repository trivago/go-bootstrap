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

	"github.com/rs/zerolog/log"
)

// Check reports probe health. A nil Check or a nil error means the probe
// succeeds. Any non-nil error marks the probe as failed.
type Check func(ctx context.Context) error

// Config provides all available configuration options for the HTTP server.
type Config struct {
	// Port defines the HTTP port the server will listen on.
	// Defaults to 8080, or 8443 for TLS when left empty.
	Port int

	// Health defines the check for the /healthz endpoint.
	// When nil, the endpoint always returns 200 OK.
	Health Check

	// Ready defines the check for the /readyz endpoint.
	// When nil, the endpoint always returns 200 OK.
	Ready Check

	// DisableAccessLogFor defines a list of paths for which the access log
	// will not be written. The path must be a full match to be disabled.
	DisableAccessLogFor []string

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

// Server is the shared lifecycle for net/http and fasthttp servers.
type Server interface {
	// ListenAndServe starts the server and blocks until it stops.
	ListenAndServe() error

	// Shutdown gracefully stops the server.
	Shutdown(ctx context.Context) error
}

// CheckOK is a Check that always reports success.
func CheckOK(_ context.Context) error {
	return nil
}

// Listen starts the given server and blocks until a stop signal like SIGINT
// or SIGTERM is received. Use signalHandler if you need to react on any of
// these signals.
func Listen(srv Server, signalHandler func(os.Signal)) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Launch server async, as ListenAndServe is blocking.
	go func() {
		log.Info().Msg("Starting listener")

		if err := srv.ListenAndServe(); err != nil {
			log.Error().Err(err).Msg("Failed to start HTTP server")
		}

		log.Info().Msg("Listener exited")
		signal.Stop(signalChan)
		close(signalChan)
	}()

	// React on external OS signals to trigger a shutdown.
	// If the channel was closed, the server did not start.
	if sig, isOpen := <-signalChan; isOpen {
		log.Info().Msgf("Received signal: %s", sig.String())
		if signalHandler != nil {
			signalHandler(sig)
		}

		log.Info().Msg("Stopping HTTP server")

		// This call is blocking and unblocks the server go routine.
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Error().Err(err).Msg("Graceful shutdown failed")
		}
	}
}

// resolvePort returns the listen port from config, applying TLS defaults.
func resolvePort(config Config) int {
	if config.Port > 0 {
		return config.Port
	}
	if len(config.PathTLSCert) > 0 || len(config.PathTLSKey) > 0 {
		return 8443
	}
	return 8080
}

// resolveAddr returns the listen address for the configured port.
func resolveAddr(config Config) string {
	return fmt.Sprintf(":%d", resolvePort(config))
}

// buildTLSConfig creates a TLS config with rotating certificates when both
// certificate paths are set. Returns nil when TLS is not configured.
func buildTLSConfig(config Config) (*tls.Config, error) {
	if len(config.PathTLSCert) == 0 || len(config.PathTLSKey) == 0 {
		return nil, nil
	}

	reloadDuration := time.Hour * 24 * 7 // 7 days
	if config.CertCacheDuration > 0 {
		reloadDuration = config.CertCacheDuration
	}

	log.Debug().Msgf(
		"Using TLS certificate %s and key %s",
		config.PathTLSCert,
		config.PathTLSKey,
	)

	cert := newFileBasedCert(config.PathTLSCert, config.PathTLSKey, reloadDuration)
	if _, err := cert.GetCertificate(); err != nil {
		return nil, err
	}

	return &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			loaded, err := cert.GetCertificate()
			if err != nil {
				return nil, err
			}
			if err := hello.SupportsCertificate(loaded); err != nil {
				// This error will be hidden by go's standard library, so we
				// log it here.
				log.Error().Err(err).Msg("Certificate does not match client requirements")
				return nil, err
			}
			return loaded, nil
		},
	}, nil
}

// probeStatus maps a Check result to an HTTP status code.
func probeStatus(ctx context.Context, check Check) int {
	if check == nil {
		return http.StatusOK
	}
	if err := check(ctx); err != nil {
		return http.StatusServiceUnavailable
	}
	return http.StatusOK
}
