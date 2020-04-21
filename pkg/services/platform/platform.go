/*
2019 © Postgres.ai
*/

// Package platform provides a Platform service.
package platform

import (
	"context"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/client/platform"
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
	cfg            Config
	client         *platform.Client
	organizationID uint
}

// NewService creates a new platform service.
func NewService(cfg Config) *Service {
	return &Service{
		cfg: cfg,
	}
}

// Init initialize a Platform service instance.
func (s *Service) Init(ctx context.Context) error {
	if !s.IsPersonalTokenEnabled() {
		return nil
	}

	client, err := platform.NewClient(platform.ClientConfig{
		URL:         s.cfg.URL,
		AccessToken: s.cfg.AccessToken,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create a new Platform Client")
	}

	s.client = client

	platformToken, err := client.CheckPlatformToken(ctx, platform.TokenCheckRequest{Token: s.cfg.AccessToken})
	if err != nil {
		return err
	}

	if platformToken.OrganizationID == 0 {
		return errors.New("invalid organization ID associated with the given Platform Access Token")
	}

	s.organizationID = platformToken.OrganizationID

	return nil
}

// IsAllowedToken checks if the Platform Personal Token is allowed.
func (s *Service) IsAllowedToken(ctx context.Context, personalToken string) bool {
	if !s.IsPersonalTokenEnabled() {
		return true
	}

	platformToken, err := s.client.CheckPlatformToken(ctx, platform.TokenCheckRequest{Token: personalToken})
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
