package app

import (
	"context"
	"errors"
	"fmt"
	"log"

	"vault/internal/auth"
	"vault/internal/config"
	"vault/internal/db"
	"vault/internal/files"
	httpserver "vault/internal/http"
	"vault/internal/storage"
)

// Application wires together config, database connections, and HTTP server.
type Application struct {
	cfg    config.Config
	dbPool *db.Pool
	srv    *httpserver.Server
}

func NewApplication(ctx context.Context, cfg config.Config) (*Application, error) {
	pool, err := db.NewPool(ctx, cfg.SupabaseDBURL)
	if err != nil {
		return nil, err
	}

	if cfg.SupabaseURL == "" || cfg.SupabaseServiceRoleKey == "" {
		return nil, errors.New("supabase storage is not configured")
	}

	storageClient := storage.NewSupabaseClient(cfg.SupabaseURL, cfg.StorageBucket, cfg.SupabaseServiceRoleKey)
	fileSvc := files.NewService(pool, storageClient, cfg.MaxUploadBytes)

	oauth, err := auth.NewGoogleOAuth(cfg)
	if err != nil {
		return nil, fmt.Errorf("google oauth: %w", err)
	}

	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, cfg.SessionTTL)
	srv := httpserver.NewServer(cfg, pool, fileSvc, oauth, jwtMgr)

	return &Application{
		cfg:    cfg,
		dbPool: pool,
		srv:    srv,
	}, nil
}

func (a *Application) Start() error {
	log.Printf("connected to Supabase Postgres, starting HTTP server on :%s", a.cfg.Port)
	return a.srv.Start()
}

func (a *Application) Shutdown(ctx context.Context) {
	if a.dbPool != nil {
		a.dbPool.Close()
	}
}
