// Command api is the entrypoint for the Barbershop Management Platform HTTP API.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gcollin65/barbershop/internal/config"
	apihttp "github.com/gcollin65/barbershop/internal/http"
	"github.com/gcollin65/barbershop/internal/logging"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// No logger yet; fail loudly on bad configuration.
		panic(err)
	}

	logger, err := logging.New(cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      apihttp.NewRouter(logger),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", zap.Int("port", cfg.Port), zap.String("env", cfg.Env))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		logger.Fatal("server error", zap.Error(err))
	case sig := <-stop:
		logger.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
		return
	}
	logger.Info("server stopped")
}
