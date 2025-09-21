package auth

import "context"

// contextKey prevents collisions with other context values.
type contextKey string

const userKey contextKey = "vault:user"

// Session contains the authenticated user identity embedded in requests.
type Session struct {
	UserID string
	Email  string
	Name   string
	Role   string
}

// WithSession stores the session on the request context.
func WithSession(ctx context.Context, s *Session) context.Context {
	if s == nil {
		return ctx
	}
	return context.WithValue(ctx, userKey, s)
}

// SessionFromContext retrieves the authenticated user from context, when available.
func SessionFromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(userKey).(*Session)
	return s, ok
}
