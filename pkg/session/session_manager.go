package session

import (
	"context"

	"github.com/google/uuid"
)

// SessionManager is a struct that can manage MCP sessions.
type SessionManager struct {
}

type Session struct {
	// SessionID is the ID of the session.
	SessionID uuid.UUID
	// SessionState is the state of the session.
	SessionState SessionState
}

type SessionState interface {
	// String returns the string representation of the session state.
	// This would be used to serialize session state to store it in remote storage if configured.
	String() string
}

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

func NewSession() *Session {
	return &Session{
		SessionID:    uuid.New(),
		SessionState: nil,
	}
}
