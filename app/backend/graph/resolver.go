package graph

import (
	"vault/internal/db"
	"vault/internal/files"
)

// Resolver wires application dependencies into GraphQL resolvers.
type Resolver struct {
	DB      *db.Pool
	FileSvc *files.Service
}

func NewResolver(pool *db.Pool, fileSvc *files.Service) *Resolver {
	return &Resolver{DB: pool, FileSvc: fileSvc}
}
