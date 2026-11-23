package http_server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
)

type Controller interface {
	RegisterRoute(g *echo.Group)
}

type HTTPServerConfig struct {
	Host         string        `envconfig:"HOST" default:"0.0.0.0"`
	Port         int           `envconfig:"PORT" default:"8802"`
	ReadTimeout  time.Duration `envconfig:"READ_TIMEOUT" default:"5s"`
	WriteTimeout time.Duration `envconfig:"WRITE_TIMEOUT" default:"10s"`
}

func (config *HTTPServerConfig) validate() error {
	if config.Port < 1024 || config.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Port)
	}

	if config.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be greater than 0")
	}
	if config.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be greater than 0")
	}

	return nil
}

type HTTPServer struct {
	config  *HTTPServerConfig
	mu      sync.RWMutex
	server  *echo.Echo
	started uint32
}

func NewHTTPServer(ec *echo.Echo, config *HTTPServerConfig) (*HTTPServer, error) {
	httpServer := &HTTPServer{
		config: config,
		server: ec,
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return httpServer, nil
}

func (s *HTTPServer) Register(middlewares []echo.MiddlewareFunc, controllers []Controller) error {
	s.server.Use(middlewares...)

	for _, ctrl := range controllers {
		ctrl.RegisterRoute(s.server.Group(""))
	}

	return nil
}

func (s *HTTPServer) Start(ctx context.Context) error {
	s.mu.Lock()

	if started := atomic.LoadUint32(&s.started); started == 1 {
		s.mu.Unlock()
		return errors.New("server already started")
	}

	atomic.StoreUint32(&s.started, 1)

	s.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.server.Server.ReadTimeout = s.config.ReadTimeout
	s.server.Server.WriteTimeout = s.config.WriteTimeout

	if err := s.server.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if started := atomic.LoadUint32(&s.started); started == 0 {
		return errors.New("server not started")
	}

	atomic.StoreUint32(&s.started, 0)

	if ctx == nil {
		ctx = context.Background()
	}

	return s.server.Shutdown(ctx)
}

func (s *HTTPServer) Instance() *echo.Echo {
	return s.server
}
