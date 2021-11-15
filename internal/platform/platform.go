/*
2019 © Postgres.ai
*/

// Package platform provides a Platform service.
package platform

import (
	"context"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/client/platform"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
)

// PersonalTokenVerifier declares an interface of a struct for Platform Personal Token verification.
type PersonalTokenVerifier interface {
	IsAllowedToken(ctx context.Context, token string) bool
	IsPersonalTokenEnabled() bool
}

// Config provides configuration for the Platform service.
type Config struct {
	URL                 string `yaml:"url"`
	AccessToken         string `yaml:"accessToken"`
	EnablePersonalToken bool   `yaml:"enablePersonalTokens"`
}

// Service defines a Platform service.
type Service struct {
	Client         *platform.Client
	cfg            Config
	organizationID uint
}

// New creates a new platform service.
func New(ctx context.Context, cfg Config) (*Service, error) {
	s := &Service{cfg: cfg}

	client, err := platform.NewClient(platform.ClientConfig{
		URL:         s.cfg.URL,
		AccessToken: s.cfg.AccessToken,
	})
	if err != nil {
		if _, ok := err.(platform.ConfigValidationError); ok {
			log.Warn(err)
			return s, nil
		}

		return nil, errors.Wrap(err, "failed to create a new Platform Client")
	}

	s.Client = client

	if !s.IsPersonalTokenEnabled() {
		return s, nil
	}

	platformToken, err := client.CheckPlatformToken(ctx, platform.TokenCheckRequest{Token: s.cfg.AccessToken})
	if err != nil {
		return nil, err
	}

	if platformToken.OrganizationID == 0 {
		return nil, errors.New("invalid organization ID associated with the given Platform Access Token")
	}

	s.organizationID = platformToken.OrganizationID

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

	return s.isAllowedOrganization(platformToken.OrganizationID)
}

// IsPersonalTokenEnabled checks if the Platform Personal Token is enabled.
func (s *Service) IsPersonalTokenEnabled() bool {
	return s.cfg.EnablePersonalToken
}

// isAllowedOrganization checks if organization is associated to the current Platform service.
func (s *Service) isAllowedOrganization(organizationID uint) bool {
	return organizationID != 0 && organizationID == s.organizationID
}
