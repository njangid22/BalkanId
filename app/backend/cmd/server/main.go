package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"vault/internal/app"
	"vault/internal/config"
)

func main() {
	_ = godotenv.Overload("../.env")
	if _, err := os.Stat(".env"); err == nil {
		_ = godotenv.Overload(".env")
	}

	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application, err := app.NewApplication(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to construct application: %v", err)
	}

	go func() {
		if err := application.Start(); err != nil {
			log.Fatalf("server exited with error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("shutting down")
	application.Shutdown(context.Background())
}
