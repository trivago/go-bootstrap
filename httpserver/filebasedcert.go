package httpserver

import (
	"crypto/tls"
	"time"

	"github.com/rs/zerolog/log"
)

// fileBasedCert is a certificate handler that is reloading the certificate from
// disk if certCacheDuration has passed.
type fileBasedCert struct {
	certFile          string
	keyFile           string
	lastRefresh       time.Time
	certCacheDuration time.Duration
	cert              *tls.Certificate
}

// newFileBasedCert creates a new certificate handler that is reloading the
// certificate from disk if certCacheDuration has passed.
func newFileBasedCert(certFile, keyFile string, certCacheDuration time.Duration) *fileBasedCert {
	return &fileBasedCert{
		certFile:    certFile,
		keyFile:     keyFile,
		lastRefresh: time.Now(),
	}
}

// GetCertificate returns a certificate from the cache, or loads it from disk if
// it is not cached yet or certCacheDuration has passed.
func (c *fileBasedCert) GetCertificate() (*tls.Certificate, error) {
	// Make sure we force a refresh when the certificate has expired
	if c.cert != nil && time.Now().After(c.cert.Leaf.NotAfter) {
		log.Warn().Msg("TLS certificate has expired, reloading.")
		c.cert = nil
	}

	// Load the certificate from disk if it is not cached yet or certCacheDuration
	// has passed.
	if c.cert == nil || time.Since(c.lastRefresh) > c.certCacheDuration {
		if c.cert != nil {
			log.Info().Msg("TLS cache duration has expired, reloading certificate from disk.")
		}
		cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
		if err != nil {
			return nil, err
		}

		c.cert = &cert
		c.lastRefresh = time.Now()
	}

	return c.cert, nil
}
