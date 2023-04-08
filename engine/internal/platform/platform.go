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
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/platform"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// PersonalTokenVerifier declares an interface of a struct for Platform Personal Token verification.
type PersonalTokenVerifier interface {
	IsAllowedToken(ctx context.Context, token string) bool
	IsPersonalTokenEnabled() bool
}

// Config provides configuration for the Platform service.
type Config struct {
	URL                 string `yaml:"url"`
	OrgKey              string `yaml:"orgKey"`
	ProjectName         string `yaml:"projectName"`
	AccessToken         string `yaml:"accessToken"`
	EnablePersonalToken bool   `yaml:"enablePersonalTokens"`
	EnableTelemetry     bool   `yaml:"enableTelemetry"`
}

// Service defines a Platform service.
type Service struct {
	Client *platform.Client
	cfg    Config
	token  Token
}

// Token defines verified Platform Token.
type Token struct {
	OrganizationID uint
	TokenType      string
	ValidUntil     *time.Time
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

			s.Client = client

			return s, nil
		}

		return nil, fmt.Errorf("failed to create new Platform Client: %w", err)
	}

	s.Client = client

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
			TokenType:      platformToken.TokenType,
			ValidUntil:     platformToken.ValidUntil,
		}
	}

	return s, nil
}

// Reload reloads service configuration.
func (s *Service) Reload(newService *Service) {
	*s = *newService
}

// IsAllowedToken checks if the Platform Personal Token is allowed.
func (s *Service) IsAllowedToken(ctx context.Context, personalToken string) bool {
	if !s.IsPersonalTokenEnabled() {
		return true
	}

	platformToken, err := s.Client.CheckPlatformToken(ctx, platform.TokenCheckRequest{Token: personalToken})
	if err != nil {
		return false
	}

	if platformToken.TokenType != platform.PersonalType {
		log.Dbg(fmt.Sprintf("Non-personal token given: %s", platformToken.TokenType))

		return false
	}

	return s.isAllowedOrganization(platformToken.OrganizationID)
}

// IsPersonalTokenEnabled checks if the Platform Personal Token is enabled.
func (s *Service) IsPersonalTokenEnabled() bool {
	return s.cfg.EnablePersonalToken
}

// isAllowedOrganization checks if organization is associated to the current Platform service.
func (s *Service) isAllowedOrganization(organizationID uint) bool {
	return organizationID != 0 && organizationID == s.token.OrganizationID
}

// IsTelemetryEnabled checks if the Platform Telemetry is enabled.
func (s *Service) IsTelemetryEnabled() bool {
	return s.cfg.EnableTelemetry
}

// OriginURL reports the origin Platform hostname.
func (s *Service) OriginURL() string {
	parsedURL, err := url.Parse(s.cfg.URL)
	if err != nil {
		log.Dbg("Cannot parse Platform URL")
	}

	platformURL := url.URL{Scheme: parsedURL.Scheme, Host: parsedURL.Host}

	return platformURL.String()
}

// AccessToken returns Platform AccessToken.
func (s *Service) AccessToken() string {
	return s.cfg.AccessToken
}

// Token returns verified Platform Token.
func (s *Service) Token() Token {
	return s.token
}

// OrgKey returns the organization key of the instance.
func (s *Service) OrgKey() string {
	return s.cfg.OrgKey
}
