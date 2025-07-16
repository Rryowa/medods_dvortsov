package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/rryowa/medods_dvortsov/internal/models"
)

type InMemoryAPIKeyManager struct {
	mu      sync.RWMutex
	apiKeys map[string]models.APIKey
}

func NewAPIKeyRepository() *InMemoryAPIKeyManager {
	// TODO: Database
	apiKeys := make(map[string]models.APIKey)
	apiKeys["test_api_key"] = models.APIKey{
		Key:      "test_api_key",
		ClientID: "test_client_id",
	}
	return &InMemoryAPIKeyManager{
		apiKeys: apiKeys,
	}
}

func (m *InMemoryAPIKeyManager) GetAPIKey(ctx context.Context, apiKey string) (*models.APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, ok := m.apiKeys[apiKey]
	if !ok {
		return nil, fmt.Errorf("api key not found")
	}

	return &key, nil
}
