package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"identity-manager/internal/app"
)

const defaultShutdownTimeout = 10 * time.Second

func main() {
	ctx := context.Background()

	application, err := app.New(ctx)
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}
	defer application.Close()

	go func() {
		if err := application.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			application.Log.WithError(err).Fatal("server failed")
		}
	}()

	application.Log.WithField("port", application.Config.App.Port).Info("identity-manager started")

	stopCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-stopCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	if err := application.Server.Shutdown(shutdownCtx); err != nil {
		application.Log.WithError(err).Error("graceful shutdown failed")
	}
}
