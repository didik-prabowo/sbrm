package app

import (
	"context"
	"fmt"
	"time"

	"net/http"

	"github.com/didik-prabowo/sbrm/internal/delivery"
	"github.com/didik-prabowo/sbrm/internal/infra/database"
	"github.com/didik-prabowo/sbrm/internal/infra/http_server"
	"github.com/didik-prabowo/sbrm/internal/repository/postgres"
	"github.com/didik-prabowo/sbrm/internal/service"
	"github.com/didik-prabowo/sbrm/pkg/bcrypt"
	"github.com/didik-prabowo/sbrm/pkg/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type HTTPServerServiceConfig struct {
	HTTPConfig     http_server.HTTPServerConfig `envconfig:"HTTP"`
	DatabaseConfig database.DatabaseConfig      `envconfig:"DB"`
	JWTConfig      jwt.JWTConfig                `envconfig:"JWT"`
}

type httpServerService struct {
	database   *database.Database
	httpServer *http_server.HTTPServer
}

func NewHTTPService(cfg *HTTPServerServiceConfig) (*httpServerService, error) {
	db, err := database.NewDatabase(&cfg.DatabaseConfig)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	jwtManager := jwt.NewManager(&cfg.JWTConfig)

	sqlRepo := postgres.NewSQLRepository(db)
	userRepository := postgres.NewUserRepository(sqlRepo)
	roomRepository := postgres.NewRoomRepository(sqlRepo)
	bookingRepository := postgres.NewBookingRepository(sqlRepo)

	authService := service.NewAuthService(bcrypt.BcryptHasher{}, *jwtManager, userRepository)
	roomService := service.NewRoomService(roomRepository)
	bookingService := service.NewBookingService(sqlRepo, roomRepository, bookingRepository)

	authController := delivery.NewAuthDelivery(authService)
	roomController := delivery.NewRoomDelivery(roomService, jwtManager)
	bookingController := delivery.NewBookingDelivery(bookingService, jwtManager)

	httpServer, err := http_server.NewHTTPServer(newEcho(), &cfg.HTTPConfig)
	if err != nil {
		return nil, fmt.Errorf("init http server: %w", err)
	}

	service := &httpServerService{
		database:   db,
		httpServer: httpServer,
	}

	service.initializeMiddleware()
	service.registerDeliveries(authController, roomController, bookingController)

	return service, nil
}

func (s *httpServerService) Start(ctx context.Context) error {
	if err := s.httpServer.Start(ctx); err != nil {
		<-s.closeResources(ctx)
		return fmt.Errorf("start server: %w", err)
	}
	return nil
}

func (s *httpServerService) Stop(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	<-s.closeResources(ctx)
	return nil
}

func (s *httpServerService) initializeMiddleware() {
	e := s.httpServer.Instance()
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
}

func (s *httpServerService) registerDeliveries(controllers ...http_server.Controller) {
	instance := s.httpServer.Instance()
	instance.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	v1 := instance.Group("/api/v1")
	for _, c := range controllers {
		c.RegisterRoute(v1)
	}
}

func (s *httpServerService) closeResources(ctx context.Context) <-chan struct{} {
	return gracefulShutdown(ctx, 10*time.Second, map[string]operation{
		"database": func(ctx context.Context) error {
			return s.database.Close(ctx)
		},
	})
}

func newEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = delivery.HTTPErrorHandler
	return e
}
