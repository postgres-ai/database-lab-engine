/*
2019 © Postgres.ai
*/

// Package client provides a client for Database Lab HTTP API.
package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/models"
)

const (
	verificationHeader = "Verification-Token"

	urlKey          = "url"
	requestDumpKey  = "request-dump"
	responseDumpKey = "response-dump"
)

// Client provides a Database Lab API client.
type Client struct {
	url               *url.URL
	verificationToken string
	client            *http.Client
	logger            logrus.FieldLogger
}

// Options describes options of a Database Lab API client.
type Options struct {
	Host              string
	VerificationToken string
	Insecure          bool
}

// NewClient constructs a new Client struct.
func NewClient(options Options, logger logrus.FieldLogger) (*Client, error) {
	u, err := url.Parse(options.Host)
	if err != nil {
		return nil, err
	}

	u.Path = strings.TrimRight(u.Path, "/")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: options.Insecure},
	}

	return &Client{
		url:               u,
		verificationToken: options.VerificationToken,
		client:            &http.Client{Transport: tr},
		logger:            logger,
	}, nil
}

// URL builds URL for a specific endpoint.
func (c *Client) URL(endpoint string) *url.URL {
	p := path.Join(c.url.Path, endpoint)

	u := *c.url
	u.Path = p

	return &u
}

// Do makes an HTTP request.
func (c *Client) Do(ctx context.Context, request *http.Request) (response *http.Response, err error) {
	// Log request and response.
	defer func() {
		if err != nil {
			b := strings.Builder{}
			b.WriteString(fmt.Sprintf("Database Lab request error: %s\n%s: %s\n",
				err.Error(), urlKey, request.URL.String()))

			if requestDump, err := httputil.DumpRequest(request, true); err == nil {
				b.WriteString(requestDumpKey)
				b.WriteString(": ")
				b.Write(requestDump)
			}

			if response != nil {
				if responseDump, err := httputil.DumpResponse(response, true); err == nil {
					b.WriteString(responseDumpKey)
					b.WriteString(": ")
					b.Write(responseDump)
				}
			}

			log.Dbg(b.String())
		}
	}()

	request.Header.Add(verificationHeader, c.verificationToken)
	request = request.WithContext(ctx)

	response, err = c.client.Do(request)
	if err != nil {
		return nil, err
	}

	// Extract error if the status code is not successful.
	if response.StatusCode >= http.StatusBadRequest {
		b, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return response, err
		}

		defer func() { _ = response.Body.Close() }()

		errModel := models.Error{}
		if err = json.Unmarshal(b, &errModel); err != nil {
			return response, errors.Wrapf(err, "failed to parse an error message: %s", (string(b)))
		}

		return response, errModel
	}

	return response, nil
}