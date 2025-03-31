package session

import (
	"context"

	"github.com/google/uuid"
)

type SessionContextKey string

const (
	sessionKey SessionContextKey = "sessionKey"
)

func getSessionFromContext(ctx context.Context) (*Session, bool) {
	session, ok := ctx.Value(sessionKey).(*Session)
	return session, ok
}

func WithSession(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, sessionKey, session)
}

func WithNewSession(ctx context.Context) context.Context {
	return WithSession(ctx, NewSession())
}

func NewSession() *Session {
	return &Session{
		SessionID:   uuid.New(),
		SessionData: nil,
	}
}
