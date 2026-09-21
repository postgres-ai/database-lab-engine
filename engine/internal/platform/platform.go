/*
2019 © Postgres.ai
*/

// Package platform provides a Platform service.
package platform

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/platform"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// PersonalTokenVerifier declares an interface of a struct for Platform Personal Token verification.
type PersonalTokenVerifier interface {
	IsAllowedToken(ctx context.Context, token string) bool
	IsPersonalTokenEnabled() bool
	AuthenticatePersonalToken(ctx context.Context, token string) (UserIdentity, bool)
}

// UserIdentity holds the identity behind a verified personal token.
type UserIdentity struct {
	Email string
}

// Config provides configuration for the Platform service.
type Config struct {
	URL                 string `yaml:"url"`
	OrgKey              string `yaml:"orgKey"`
	ProjectName         string `yaml:"projectName"`
	AccessToken         string `yaml:"accessToken"`
	EnablePersonalToken bool   `yaml:"enablePersonalTokens"`
	EnableTelemetry     bool   `yaml:"enableTelemetry"`
	BindClonesToUser    bool   `yaml:"bindClonesToUser"`
}

// Service defines a Platform service. Reload replaces the config, the verified token and the
// API client while request handlers read them, so mu guards all three.
type Service struct {
	mu     sync.RWMutex
	client *platform.Client
	cfg    Config
	token  Token
}

// Token defines verified Platform Token.
type Token struct {
	OrganizationID uint
}

// New creates a new platform service.
func New(ctx context.Context, cfg Config, instanceID string) (*Service, error) {
	s := &Service{cfg: cfg}

	client, err := platform.NewClient(platform.ClientConfig{
		URL:         s.cfg.URL,
		OrgKey:      s.cfg.OrgKey,
		ProjectName: s.cfg.ProjectName,
		AccessToken: s.cfg.AccessToken,
		InstanceID:  instanceID,
	})
	if err != nil {
		var cvWarning *platform.ConfigValidationWarning
		if errors.As(err, &cvWarning) {
			log.Warn(err)

			s.client = client

			return s, nil
		}

		return nil, fmt.Errorf("failed to create new Platform Client: %w", err)
	}

	s.client = client

	if s.cfg.AccessToken != "" {
		platformToken, err := client.CheckPlatformToken(ctx, platform.TokenCheckRequest{Token: s.cfg.AccessToken})
		if err != nil {
			return nil, err
		}

		if platformToken.OrganizationID == 0 {
			return nil, errors.New("invalid organization ID associated with the given Platform Access Token")
		}

		s.token = Token{
			OrganizationID: platformToken.OrganizationID,
		}
	}

	return s, nil
}

// Reload reloads service configuration. It copies the fields one by one rather than replacing
// the whole struct, because the struct carries a mutex and must not be copied.
func (s *Service) Reload(newService *Service) {
	cfg, token, client := newService.snapshot()

	s.mu.Lock()
	s.cfg = cfg
	s.token = token
	s.client = client
	s.mu.Unlock()
}

func (s *Service) snapshot() (Config, Token, *platform.Client) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg, s.token, s.client
}

// Client returns the Platform API client the service is configured with.
func (s *Service) Client() *platform.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.client
}

// IsAllowedToken checks if the Platform Personal Token is allowed.
func (s *Service) IsAllowedToken(ctx context.Context, personalToken string) bool {
	_, ok := s.AuthenticatePersonalToken(ctx, personalToken)

	return ok
}

// AuthenticatePersonalToken verifies a personal token and returns the user identity behind it.
func (s *Service) AuthenticatePersonalToken(ctx context.Context, personalToken string) (UserIdentity, bool) {
	if !s.IsPersonalTokenEnabled() {
		return UserIdentity{}, false
	}

	platformToken, err := s.Client().CheckPlatformToken(ctx, platform.TokenCheckRequest{Token: personalToken})
	if err != nil {
		return UserIdentity{}, false
	}

	if !platformToken.Personal {
		log.Dbg("Non-personal token given")

		return UserIdentity{}, false
	}

	if !s.isAllowedOrganization(platformToken.OrganizationID) {
		return UserIdentity{}, false
	}

	return UserIdentity{Email: platformToken.Email}, true
}

// BindClonesToUser reports whether each clone must be labeled with a trusted dblab_user
// value derived from the authenticated user identity; the clone's Postgres username is
// left unchanged.
func (s *Service) BindClonesToUser() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg.BindClonesToUser
}

// IsPersonalTokenEnabled checks if the Platform Personal Token is enabled.
func (s *Service) IsPersonalTokenEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg.EnablePersonalToken
}

// isAllowedOrganization checks if organization is associated to the current Platform service.
func (s *Service) isAllowedOrganization(organizationID uint) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return organizationID != 0 && organizationID == s.token.OrganizationID
}

// IsTelemetryEnabled checks if the Platform Telemetry is enabled.
func (s *Service) IsTelemetryEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg.EnableTelemetry
}

// OriginURL reports the origin Platform hostname.
func (s *Service) OriginURL() string {
	s.mu.RLock()
	configuredURL := s.cfg.URL
	s.mu.RUnlock()

	parsedURL, err := url.Parse(configuredURL)
	if err != nil {
		log.Dbg("Cannot parse Platform URL")
	}

	platformURL := url.URL{Scheme: parsedURL.Scheme, Host: parsedURL.Host}

	return platformURL.String()
}

// AccessToken returns Platform AccessToken.
func (s *Service) AccessToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg.AccessToken
}

// Token returns verified Platform Token.
func (s *Service) Token() Token {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.token
}

// OrgKey returns the organization key of the instance.
func (s *Service) OrgKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg.OrgKey
}
