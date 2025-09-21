package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID         uuid.UUID
	Email      string
	Name       *string
	Role       string
	QuotaBytes int64
	CreatedAt  time.Time
}

const upsertUserSQL = `
insert into users (email, name)
values ($1, nullif($2, ''))
on conflict (email)
    do update set name = excluded.name
returning id, email, name, role, quota_bytes, created_at;
`

const getUserByIDSQL = `
select id, email, name, role, quota_bytes, created_at
from users
where id = $1;
`

func (p *Pool) UpsertUser(ctx context.Context, email, name string) (User, error) {
	var user User
	if p == nil {
		return user, errors.New("nil db pool")
	}

	row := p.QueryRow(ctx, upsertUserSQL, email, name)
	if err := row.Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.QuotaBytes, &user.CreatedAt); err != nil {
		return user, fmt.Errorf("upsert user: %w", err)
	}
	return user, nil
}

func (p *Pool) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var user User
	if p == nil {
		return user, errors.New("nil db pool")
	}

	row := p.QueryRow(ctx, getUserByIDSQL, id)
	if err := row.Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.QuotaBytes, &user.CreatedAt); err != nil {
		return user, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}
