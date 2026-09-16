// Package webhooks configures the webhooks that will be called by the DBLab Engine when an event occurs.
package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/goroutine"
)

const (
	// DLEWebhookTokenHeader defines the HTTP header name to send secret with the webhook request.
	DLEWebhookTokenHeader = "DBLab-Webhook-Token"

	requestTimeout        = 10 * time.Second
	dialTimeout           = 5 * time.Second
	tlsHandshakeTimeout   = 5 * time.Second
	idleConnTimeout       = 90 * time.Second
	expectContinueTimeout = time.Second
	maxIdleConns          = 10
	maxResponseBodySize   = 64 << 10
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
	mu            sync.RWMutex
	hooksRegistry map[string][]Hook
	eventCh       <-chan EventTyper
}

// NewService creates a new Webhook Service.
func NewService(cfg *Config, eventCh <-chan EventTyper) *Service {
	whs := &Service{
		client:        newClient(),
		hooksRegistry: make(map[string][]Hook),
		eventCh:       eventCh,
	}

	whs.Reload(cfg)

	return whs
}

func newClient() *http.Client {
	return &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: dialTimeout}).DialContext,
			TLSHandshakeTimeout:   tlsHandshakeTimeout,
			ResponseHeaderTimeout: requestTimeout,
			ExpectContinueTimeout: expectContinueTimeout,
			IdleConnTimeout:       idleConnTimeout,
			MaxIdleConns:          maxIdleConns,
		},
	}
}

// Reload reloads Webhook Service configuration.
func (s *Service) Reload(cfg *Config) {
	registry := make(map[string][]Hook)

	for _, hook := range cfg.Hooks {
		if err := validateURL(hook.URL); err != nil {
			log.Msg("Skip webhook processing:", err)
			continue
		}

		for _, event := range hook.Trigger {
			registry[event] = append(registry[event], hook)
		}
	}

	s.mu.Lock()
	s.hooksRegistry = registry
	s.mu.Unlock()

	log.Dbg("Registered webhooks", registry)
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
		hooks := s.hooksFor(whEvent.GetType())
		if len(hooks) == 0 {
			log.Dbg("Skipped unknown hook: ", whEvent.GetType())

			continue
		}

		log.Dbg("Trigger event:", whEvent)

		for _, hook := range hooks {
			goroutine.Go("webhook "+hook.URL, func() { s.triggerWebhook(ctx, hook, whEvent) })
		}
	}
}

// hooksFor returns a copy of the hooks registered for the event type, so a concurrent Reload
// cannot alter the slice while it is being dispatched.
func (s *Service) hooksFor(eventType string) []Hook {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Clone(s.hooksRegistry[eventType])
}

func (s *Service) triggerWebhook(ctx context.Context, hook Hook, whEvent EventTyper) {
	log.Msg("Webhook request: ", hook.URL)

	resp, err := s.makeRequest(ctx, hook, whEvent)
	if err != nil {
		log.Err("webhook error:", err)
		return
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if err != nil {
		log.Err("webhook error:", err)
		return
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Warn(fmt.Sprintf("webhook %s responded with status %d", hook.URL, resp.StatusCode))
		log.Dbg("Webhook response: ", string(body))

		return
	}

	log.Dbg("Webhook status code: ", resp.StatusCode)
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
