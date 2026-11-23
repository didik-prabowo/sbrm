package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/didik-prabowo/sbrm/app"
	cfg "github.com/didik-prabowo/sbrm/config"
	"github.com/kelseyhightower/envconfig"
)

func initializeConfig() error {
	path := os.Getenv("CONFIG")
	if path == "" {
		path = ".env"
	}

	m, err := cfg.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && os.Getenv("CONFIG") == "" {
		return nil
	}
	if err != nil {
		return err
	}

	return cfg.Setenv(m)
}

func main() {
	mode := flag.String("mode", "http", "run mode (only http for now)")
	flag.Parse()

	if err := initializeConfig(); err != nil {
		fmt.Printf("ERROR: error initializing config, err=%v\n", err)
		os.Exit(1)
	}

	if err := serve(*mode); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
}

func serve(mode string) error {
	var (
		service app.Service
		err     error
	)

	switch mode {
	case "http":
		var cfg app.HTTPServerServiceConfig
		if err = envconfig.Process("SBRM", &cfg); err != nil {
			return fmt.Errorf("process http config: %w", err)
		}

		service, err = app.NewHTTPService(&cfg)

	default:
		return fmt.Errorf("unknown mode %q, only http is supported", mode)
	}

	if err != nil {
		return fmt.Errorf("create %s service: %w", mode, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() { errChan <- service.Start(ctx) }()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(interrupt)

	select {
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("service error: %w", err)
		}
		return nil
	case sig := <-interrupt:
		fmt.Printf("Received signal: %v\n", sig)
		cancel()
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer stopCancel()

	return service.Stop(stopCtx)
}
