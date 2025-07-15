package service

import (
	"context"
	"sync"
	"time"

	"github.com/rryowa/medods_dvortsov/internal/models"
)

type InMemoryRefreshTokenManager struct {
	mu       sync.RWMutex
	sessions map[string]models.RefreshSession
}

func NewInMemoryRefreshTokenManager() *InMemoryRefreshTokenManager {
	return &InMemoryRefreshTokenManager{
		sessions: make(map[string]models.RefreshSession),
	}
}

func (m *InMemoryRefreshTokenManager) CreateSession(_ context.Context, s models.RefreshSession, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions[s.RefreshToken] = s
	return nil
}

func (m *InMemoryRefreshTokenManager) FindSession(_ context.Context, token string) (*models.RefreshSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[token]
	if !ok {
		return nil, nil
	}
	return &s, nil
}

func (m *InMemoryRefreshTokenManager) DeleteSession(_ context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, token)
	return nil
}

func (m *InMemoryRefreshTokenManager) DeleteAllUserSessions(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for k, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, k)
		}
	}
	return nil
}
