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
	accessToken = "Access-Token"
)

// ConfigValidationError represents a config validation error.
type ConfigValidationError error

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
	accessToken string
	client      *http.Client
}

// ClientConfig describes configuration parameters of Postgres.ai Platform client.
type ClientConfig struct {
	URL         string
	AccessToken string
}

// NewClient creates a new Platform API client.
func NewClient(platformCfg ClientConfig) (*Client, error) {
	if err := validateConfig(platformCfg); err != nil {
		return nil, err
	}

	u, err := url.Parse(platformCfg.URL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse the platform host")
	}

	u.Path = strings.TrimRight(u.Path, "/")

	p := Client{
		url:         u,
		accessToken: platformCfg.AccessToken,
		client: &http.Client{
			Transport: &http.Transport{},
		},
	}

	return &p, nil
}

func validateConfig(config ClientConfig) error {
	if config.URL == "" || config.AccessToken == "" {
		return ConfigValidationError(errors.New("invalid config of Platform Client given: URL and AccessToken must not be empty"))
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
	request.Header.Add(accessToken, p.accessToken)
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
