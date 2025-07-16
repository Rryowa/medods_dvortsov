package memory

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/models"
	"github.com/rryowa/medods_dvortsov/internal/storage"
)

type InMemorySessionManager struct {
	mu       sync.RWMutex
	sessions map[string]models.RefreshSession
	log      *zap.SugaredLogger
}

func NewSessionRepository(log *zap.SugaredLogger) storage.SessionRepository {
	return &InMemorySessionManager{
		sessions: make(map[string]models.RefreshSession),
		log:      log,
	}
}

func (m *InMemorySessionManager) CreateSession(ctx context.Context, session models.RefreshSession, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions[session.ID] = session
	m.log.Debugw("Session created", "sessionID", session.ID, "userID", session.UserID, "ttl", ttl)

	return nil
}

func (m *InMemorySessionManager) GetCurrentSession(ctx context.Context, refreshToken string) (*models.RefreshSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.log.Debugw("Attempting to get session", "refreshToken", refreshToken)
	session, ok := m.sessions[refreshToken]
	if !ok {
		m.log.Debugw("Session not found", "refreshToken", refreshToken)
		return nil, storage.ErrSessionNotFound
	}

	m.log.Debugw("Session found", "sessionID", session.ID, "userID", session.UserID)
	return &session, nil
}

func (m *InMemorySessionManager) DeleteSession(ctx context.Context, refreshToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, refreshToken)

	return nil
}

func (m *InMemorySessionManager) DeleteAllUserSessions(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, session := range m.sessions {
		if session.UserID == userID {
			delete(m.sessions, id)
		}
	}

	return nil
}
