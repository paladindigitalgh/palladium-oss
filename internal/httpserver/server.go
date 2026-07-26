// Package httpserver wraps net/http.Server with graceful shutdown driven by
// context cancellation.
package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Config holds the settings needed to run a Server.
type Config struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// Server runs an http.Handler and shuts it down gracefully when its Run
// context is cancelled.
type Server struct {
	inner           *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

// New builds a Server ready to Run.
func New(cfg Config, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		inner: &http.Server{
			Addr:         cfg.Addr,
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		logger:          logger,
		shutdownTimeout: cfg.ShutdownTimeout,
	}
}

// Run starts the server and blocks until either it fails or ctx is
// cancelled, in which case it attempts a graceful shutdown bounded by the
// configured shutdown timeout.
func (s *Server) Run(ctx context.Context) error {
	serveErr := make(chan error, 1)

	go func() {
		s.logger.Info("http server listening", "addr", s.inner.Addr)
		if err := s.inner.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		return s.shutdown()
	}
}

func (s *Server) shutdown() error {
	s.logger.Info("http server shutting down", "timeout", s.shutdownTimeout)

	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if err := s.inner.Shutdown(ctx); err != nil {
		s.logger.Error("http server graceful shutdown failed", "error", err)
		return err
	}

	s.logger.Info("http server stopped")
	return nil
}
