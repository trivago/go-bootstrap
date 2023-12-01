package httpserver

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTLS(t *testing.T) {
	srv, err := NewWithConfig(Config{
		Port:        8443,
		PathTLSCert: "../hack/tls.cert",
		PathTLSKey:  "../hack/tls.key",
	})

	assert.NoError(t, err)
	assert.NotNil(t, srv)

	go Listen(srv, nil)

	// On OSX a warning pops up where the user needs to allow network access by the
	// unittest. To give the user some time, we wait for 5 seconds.
	time.Sleep(5 * time.Second)

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	_, err = http.Get("https://localhost:8443/healthz")
	assert.NoError(t, err)
}
