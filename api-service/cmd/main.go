package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud-native-platform/api-service/config"
	"cloud-native-platform/api-service/controllers"
	"cloud-native-platform/api-service/database/postgres"
	"cloud-native-platform/api-service/database/repository"
	"cloud-native-platform/api-service/gemini"
	"cloud-native-platform/api-service/usecases"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := config.LoadEnv(); err != nil {
		return err
	}
	ai, err := gemini.NewFromEnv()
	if err != nil {
		return err
	}
	pool, err := postgres.Connect(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	processor := usecases.NewAnalyzePDFUseCase(repository.NewAnalysisStore(pool), ai)
	handler, err := controllers.NewAnalysisHandler("uploads", processor)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr: "127.0.0.1:8080", Handler: handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      180 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Println("doc-forge: http://localhost:8080")
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		// Restore default signal handling so a second interrupt exits immediately.
		stop()
	}

	log.Println("shutting down HTTP server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		if closeErr := server.Close(); closeErr != nil {
			log.Printf("force close HTTP server: %v", closeErr)
		}
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	log.Println("HTTP server stopped")
	return nil
}
