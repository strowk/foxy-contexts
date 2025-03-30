package session

import (
	"context"

	"github.com/google/uuid"
)

// SessionManager is a struct that can manage MCP sessions.
type SessionManager struct {
	sessions map[uuid.UUID]*Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[uuid.UUID]*Session),
	}
}

func (sm *SessionManager) GetSessionData(ctx context.Context) SessionData {
	session, ok := getSessionFromContext(ctx)
	if !ok {
		return nil
	}
	return session.SessionData
}

func (sm *SessionManager) SetSessionData(ctx context.Context, sessionData SessionData) {
	session, ok := getSessionFromContext(ctx)
	if !ok {
		return
	}
	session.SessionData = sessionData
	sm.saveSession(session)
}

func (sm *SessionManager) GetSessionFromContext(ctx context.Context) (*Session, bool) {
	session, ok := getSessionFromContext(ctx)
	// double check if was not removed from the session manager
	if ok {
		_, ok = sm.sessions[session.SessionID]
	}
	if !ok {
		return nil, false
	}

	return session, ok
}

func (sm *SessionManager) FindSessionById(sessionId uuid.UUID) (*Session, bool) {
	session, ok := sm.sessions[sessionId]
	return session, ok
}

func (sm *SessionManager) ResolveSessionOrCreateNew(ctx context.Context, sessionId uuid.UUID) (context.Context, *Session, error) {
	session, ok := sm.FindSessionById(sessionId)
	if ok {
		ctx = WithSession(ctx, session)
		return ctx, session, nil
	}

	return sm.CreateNewSession(ctx)
}

func (sm *SessionManager) DeleteSession(sessionId uuid.UUID) {
	delete(sm.sessions, sessionId)
}

func (sm *SessionManager) CreateNewSession(ctx context.Context) (context.Context, *Session, error) {
	session := NewSession()
	ctx = WithSession(ctx, session)
	sm.saveSession(session)
	return ctx, session, nil
}

func (sm *SessionManager) saveSession(session *Session) {
	sm.sessions[session.SessionID] = session
}

type Session struct {
	// SessionID is the ID of the session.
	SessionID uuid.UUID
	// SessionData is the state of the session.
	SessionData SessionData
}

type SessionData interface {
	// String returns the string representation of the session state.
	// This would be used to serialize session state to store it in remote storage if configured.
	String() string
}
