package db

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func uuidPtrFromPG(value pgtype.UUID) (*uuid.UUID, error) {
	if !value.Valid {
		return nil, nil
	}
	uid, err := uuid.FromBytes(value.Bytes[:])
	if err != nil {
		return nil, err
	}
	return &uid, nil
}
