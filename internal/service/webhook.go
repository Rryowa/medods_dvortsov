package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

const (
	defaultHTTPStatusThreshold = 300
)

type WebhookService struct {
	client     *http.Client
	log        *zap.SugaredLogger
	webhookURL string
}

func NewWebhookService(log *zap.SugaredLogger, webhookURL string) *WebhookService {
	return &WebhookService{
		client:     &http.Client{},
		log:        log,
		webhookURL: webhookURL,
	}
}

func (s *WebhookService) NotifyIPChange(ctx context.Context, data map[string]interface{}) {
	go func() {
		if s.webhookURL == "" {
			return
		}

		payload, err := json.Marshal(data)
		if err != nil {
			s.log.Errorw("failed to marshal webhook payload", "error", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewBuffer(payload))
		if err != nil {
			s.log.Errorw("failed to create webhook request", "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			s.log.Errorw("failed to send webhook", "error", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= defaultHTTPStatusThreshold {
			s.log.Warnw("webhook returned non-2xx status", "status", resp.StatusCode)
		}
	}()
}
