package httpserver

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// fileBasedCert is a certificate handler that is reloading the certificate from
// disk if certCacheDuration has passed.
type fileBasedCert struct {
	mutex             *sync.Mutex
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
		certFile:          certFile,
		keyFile:           keyFile,
		lastRefresh:       time.Now(),
		certCacheDuration: certCacheDuration,
		mutex:             &sync.Mutex{},
	}
}

// GetCertificate returns a certificate from the cache, or loads it from disk if
// it is not cached yet or certCacheDuration has passed.
func (c *fileBasedCert) GetCertificate() (*tls.Certificate, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	now := time.Now()

	reload := func() (*tls.Certificate, error) {
		c.cert = nil // make sure we don't return the old certificate.
		cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)

		switch {
		case err != nil:
			return nil, err
		case cert.Leaf == nil:
			return nil, fmt.Errorf("certificate leaf is nil")
		case now.After(cert.Leaf.NotAfter):
			// This is a warning on purpose, as we don't want to fail the
			// server startup if the certificate is expired. We will just keep
			// using the expired certificate, which will be result in an error
			// for the client.
			log.Warn().Msg("reloaded TLS certificate has already expired.")

			// When certCacheDuration is set to a value higher than one minute,
			// we will retry within the next minute. This is to make sure that
			// we don't keep using an expired certificate for too long.
			if c.certCacheDuration > time.Minute {
				now = now.Add(time.Minute - c.certCacheDuration)
			}
		}

		c.cert = &cert
		c.lastRefresh = now
		return &cert, nil
	}

	switch {
	// Load the certificate from disk if we don't have one cached yet.
	case c.cert == nil:
		log.Info().Msg("No TLS certificate cached, loading.")
		return reload()

	// Reload the certificate if it has expired.
	case now.After(c.cert.Leaf.NotAfter):
		log.Warn().Msg("TLS certificate has expired, reloading.")
		return reload()

	// Check for a new certificate in regular intervals.
	// We only change the loaded certificate if the signature has changed.
	case now.Sub(c.lastRefresh) > c.certCacheDuration:
		log.Info().Msg("TLS certificate cache duration has passed.")

		// Check if the certificate file has been changed since the last
		// refresh. This is a simple check that only compares the modification
		// time of the file.
		if fileInfo, err := os.Stat(c.certFile); err == nil && fileInfo.ModTime().After(c.lastRefresh) {
			log.Warn().Msg("TLS certificate file has been modifed since last refresh.")
			return reload()
		}

		// Check if the certificate signature has changed since the last
		// refresh. This is a more expensive check that compares the signature
		// of the certificate.
		cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
		if err != nil || cert.Leaf == nil {
			log.Error().Err(err).Msg("Failed to load TLS certificate for comparison, keeping cached certificate.")
			return c.cert, nil
		}

		if !bytes.Equal(cert.Leaf.Signature, c.cert.Leaf.Signature) {
			log.Warn().Msg("Detected certificate signature change, reloading.")
			return reload()
		}
	}

	return c.cert, nil
}
