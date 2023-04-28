/*
2019 © Postgres.ai
*/

// Package platform provides the Platform API client.
package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	accessToken   = "Access-Token"
	orgKey        = "Org-Key"
	instanceIDKey = "Selfassigned-Instance-ID"
)

// ConfigValidationWarning represents a config validation warning.
type ConfigValidationWarning struct {
	Message string
}

// NewConfigValidationWarning creates a new ConfigValidationWarning.
func NewConfigValidationWarning(message string) *ConfigValidationWarning {
	return &ConfigValidationWarning{Message: message}
}

// Error returns the warning message.
func (c *ConfigValidationWarning) Error() string {
	return c.Message
}

// APIResponse represents common fields of an API response.
type APIResponse struct {
	Hint    string `json:"hint"`
	Details string `json:"details"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Client provides the Platform API client.
type Client struct {
	url         *url.URL
	orgKey      string
	projectName string
	accessToken string
	instanceID  string
	client      *http.Client
}

// ClientConfig describes configuration parameters of Postgres.ai Platform client.
type ClientConfig struct {
	URL         string
	OrgKey      string
	ProjectName string
	AccessToken string
	InstanceID  string
}

// NewClient creates a new Platform API client.
func NewClient(platformCfg ClientConfig) (*Client, error) {
	p := Client{
		orgKey:      platformCfg.OrgKey,
		projectName: platformCfg.ProjectName,
		accessToken: platformCfg.AccessToken,
		instanceID:  platformCfg.InstanceID,
		client: &http.Client{
			Transport: &http.Transport{},
		},
	}

	if err := validateConfig(platformCfg); err != nil {
		return &p, err
	}

	u, err := url.Parse(platformCfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse platform URL: %w", err)
	}

	u.Path = strings.TrimRight(u.Path, "/")
	p.url = u

	return &p, checkConfigTokens(platformCfg)
}

func validateConfig(config ClientConfig) error {
	if config.URL == "" {
		if config.OrgKey != "" || config.AccessToken != "" {
			return errors.New("platform.url in config must not be empty")
		}

		return NewConfigValidationWarning("Platform URL is empty")
	}

	return nil
}

func checkConfigTokens(config ClientConfig) error {
	if config.AccessToken == "" && config.OrgKey == "" {
		return NewConfigValidationWarning("Both accessToken and orgKey are empty; at least one must be specified")
	}

	return nil
}

type responseParser func(*http.Response) error

func newJSONParser(v interface{}) responseParser {
	return func(resp *http.Response) error {
		return json.NewDecoder(resp.Body).Decode(v)
	}
}

func newUploadParser(v interface{}) responseParser {
	return func(resp *http.Response) error {
		if resp.StatusCode != http.StatusOK {
			return json.NewDecoder(resp.Body).Decode(v)
		}

		return nil
	}
}

func (p *Client) doRequest(ctx context.Context, request *http.Request, parser responseParser) error {
	if p.accessToken != "" {
		request.Header.Add(accessToken, p.accessToken)
	}

	if p.orgKey != "" {
		request.Header.Add(orgKey, p.orgKey)
	}

	request.Header.Add(instanceIDKey, p.instanceID)

	request = request.WithContext(ctx)

	response, err := p.client.Do(request)
	if err != nil {
		return errors.Wrap(err, "failed to make API request")
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read response")
		}

		log.Dbg(fmt.Sprintf("Response: %v", string(body)))

		response.Body = io.NopCloser(bytes.NewBuffer(body))
		if err := parser(response); err != nil {
			return errors.Wrap(err, "failed to parse response")
		}

		return errors.Errorf("unsuccessful status given: %d", response.StatusCode)
	}

	return parser(response)
}

func (p *Client) doPost(ctx context.Context, path string, data interface{}, response interface{}) error {
	reqData, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request")
	}

	postURL := p.buildURL(path).String()

	r, err := http.NewRequest(http.MethodPost, postURL, bytes.NewBuffer(reqData))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	if err := p.doRequest(ctx, r, newJSONParser(&response)); err != nil {
		return errors.Wrap(err, "failed to perform request")
	}

	return nil
}

func (p *Client) doUpload(ctx context.Context, path string, reqData []byte, headers map[string]string, respData responseParser) error {
	postURL := p.buildURL(path).String()

	r, err := http.NewRequest(http.MethodPost, postURL, bytes.NewBuffer(reqData))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	for key, value := range headers {
		r.Header.Add(key, value)
	}

	if err := p.doRequest(ctx, r, respData); err != nil {
		return errors.Wrap(err, "failed to perform request")
	}

	return nil
}

// URL builds URL for a specific endpoint.
func (p *Client) buildURL(urlPath string) *url.URL {
	fullPath := path.Join(p.url.Path, urlPath)

	u := *p.url
	u.Path = fullPath

	return &u
}
