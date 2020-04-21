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
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
)

const (
	accessToken = "Access-Token"
)

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
		return errors.New("invalid config of Platform Client given: URL and AccessToken must not be empty")
	}

	return nil
}

type responseParser func(*http.Response) error

func newJSONParser(v interface{}) responseParser {
	return func(resp *http.Response) error {
		return json.NewDecoder(resp.Body).Decode(v)
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
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read response")
		}

		log.Dbg(fmt.Sprintf("Response: %v", string(body)))

		response.Body = ioutil.NopCloser(bytes.NewBuffer(body))
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

// TokenCheckRequest represents token checking request.
type TokenCheckRequest struct {
	Token string `json:"token"`
}

// TokenCheckResponse represents a response of a token checking request.
type TokenCheckResponse struct {
	APIResponse
	OrganizationID uint `json:"org_id"`
	Personal       bool `json:"is_personal"`
}

// CheckPlatformToken makes an HTTP request to check the Platform Access Token.
func (p *Client) CheckPlatformToken(ctx context.Context, request TokenCheckRequest) (TokenCheckResponse, error) {
	respData := TokenCheckResponse{}

	if err := p.doPost(ctx, "/rpc/dblab_token_check", request, &respData); err != nil {
		return respData, errors.Wrap(err, "failed to post request")
	}

	if respData.Code != "" || respData.Message != "" {
		log.Dbg(fmt.Sprintf("Unsuccessful response given. Request: %v", request))

		return respData, errors.Errorf("error: %v", respData)
	}

	return respData, nil
}

// URL builds URL for a specific endpoint.
func (p *Client) buildURL(urlPath string) *url.URL {
	fullPath := path.Join(p.url.Path, urlPath)

	u := *p.url
	u.Path = fullPath

	return &u
}
