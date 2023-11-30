package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	golog "log"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/trivago/go-bootstrap/logging"
)

// AlwaysOk is a gin handler that always returns a 200 OK.
func AlwaysOk(c *gin.Context) {
	c.Status(http.StatusOK)
	c.Writer.WriteHeaderNow()
}

// New creates a new HTTP server with the given health and ready handlers.
// Pass an initRoutes function to configure routes on this server.
func New(port int, health, ready gin.HandlerFunc, initRoutes func(router *gin.Engine)) *http.Server {
	router := gin.New()
	router.Use(newZeroLogLogger([]string{"/healthz", "/readyz"}), gin.Recovery())

	router.GET("/healthz", health)
	router.GET("/readyz", ready)

	if initRoutes != nil {
		initRoutes(router)
	}

	return &http.Server{
		Addr:     fmt.Sprintf(":%d", port),
		Handler:  router,
		ErrorLog: golog.New(logging.ErrorLogWriter{}, "", 0),
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
