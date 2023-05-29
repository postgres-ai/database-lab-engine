// Package webhooks configures the webhooks that will be called by the DBLab Engine when an event occurs.
package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	// DLEWebhookTokenHeader defines the HTTP header name to send secret with the webhook request.
	DLEWebhookTokenHeader = "DLE-Webhook-Token"
)

// Config defines webhooks configuration.
type Config struct {
	Hooks []Hook `yaml:"hooks"`
}

// Hook defines structure of the webhook configuration.
type Hook struct {
	URL     string   `yaml:"url"`
	Secret  string   `yaml:"secret"`
	Trigger []string `yaml:"trigger"`
}

// Service listens events and performs webhooks requests.
type Service struct {
	client        *http.Client
	hooksRegistry map[string][]Hook
	eventCh       <-chan EventTyper
}

// NewService creates a new Webhook Service.
func NewService(cfg *Config, eventCh <-chan EventTyper) *Service {
	whs := &Service{
		client: &http.Client{
			Transport: &http.Transport{},
		},
		hooksRegistry: make(map[string][]Hook),
		eventCh:       eventCh,
	}

	whs.Reload(cfg)

	return whs
}

// Reload reloads Webhook Service configuration.
func (s *Service) Reload(cfg *Config) {
	s.hooksRegistry = make(map[string][]Hook)

	for _, hook := range cfg.Hooks {
		if err := validateURL(hook.URL); err != nil {
			log.Msg("Skip webhook processing:", err)
			continue
		}

		for _, event := range hook.Trigger {
			s.hooksRegistry[event] = append(s.hooksRegistry[event], hook)
		}
	}

	log.Dbg("Registered webhooks", s.hooksRegistry)
}

func validateURL(hookURL string) error {
	parsedURL, err := url.ParseRequestURI(hookURL)
	if err != nil {
		return fmt.Errorf("URL %q is invalid: %w", hookURL, err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("no scheme found in %q", hookURL)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("no host found in %q", hookURL)
	}

	return nil
}

// Run starts webhook listener.
func (s *Service) Run(ctx context.Context) {
	for whEvent := range s.eventCh {
		hooks, ok := s.hooksRegistry[whEvent.GetType()]
		if !ok {
			log.Dbg("Skipped unknown hook: ", whEvent.GetType())

			continue
		}

		log.Dbg("Trigger event:", whEvent)

		for _, hook := range hooks {
			go s.triggerWebhook(ctx, hook, whEvent)
		}
	}
}

func (s *Service) triggerWebhook(ctx context.Context, hook Hook, whEvent EventTyper) {
	log.Msg("Webhook request: ", hook.URL)

	resp, err := s.makeRequest(ctx, hook, whEvent)

	if err != nil {
		log.Err("Webhook error:", err)
		return
	}

	log.Dbg("Webhook status code: ", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Err("Webhook error:", err)
		return
	}

	log.Dbg("Webhook response: ", string(body))
}

func (s *Service) makeRequest(ctx context.Context, hook Hook, whEvent EventTyper) (*http.Response, error) {
	payload, err := json.Marshal(whEvent)
	if err != nil {
		return nil, err
	}

	log.Dbg("Webhook payload: ", string(payload))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	if hook.Secret != "" {
		req.Header.Add(DLEWebhookTokenHeader, hook.Secret)
	}

	req.Header.Set("Content-Type", "application/json")

	return s.client.Do(req)
}
