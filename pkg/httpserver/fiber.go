// Package httpserver implements HTTP server.
package httpserver

import (
	"context"
	"time"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
)

const (
	_defaultAddr            = ":80"
	_defaultReadTimeout     = 5 * time.Second
	_defaultWriteTimeout    = 5 * time.Second
	_defaultShutdownTimeout = 3 * time.Second
)

// Server -.
type Server struct {
	App    *fiber.App
	notify chan error

	address         string
	prefork         bool
	readTimeout     time.Duration
	writeTimeout    time.Duration
	shutdownTimeout time.Duration
}

func (s *Server) ListenAndServe() error {
	return s.App.Listen(s.address, fiber.ListenConfig{EnablePrefork: s.prefork})
}

func (s *Server) ShutdownContext(ctx context.Context) error { return s.App.ShutdownWithContext(ctx) }

// New -.
func New(opts ...Option) *Server {
	s := &Server{
		App:             nil,
		notify:          make(chan error, 1),
		address:         _defaultAddr,
		readTimeout:     _defaultReadTimeout,
		writeTimeout:    _defaultWriteTimeout,
		shutdownTimeout: _defaultShutdownTimeout,
	}

	// Custom options
	for _, opt := range opts {
		opt(s)
	}

	app := fiber.New(fiber.Config{
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
		BodyLimit:    64 * 1024,
		JSONDecoder:  json.Unmarshal,
		JSONEncoder:  json.Marshal,
	})

	s.App = app

	return s
}

// Start -.
func (s *Server) Start() {
	go func() {
		s.notify <- s.ListenAndServe()
		close(s.notify)
	}()
}

// Notify -.
func (s *Server) Notify() <-chan error {
	return s.notify
}

// Shutdown -.
func (s *Server) Shutdown() error {
	return s.App.ShutdownWithTimeout(s.shutdownTimeout)
}
