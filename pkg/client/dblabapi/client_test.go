package dblabapi

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTripFunc represents a mock type.
type roundTripFunc func(req *http.Request) *http.Response

// RoundTrip is a mock function.
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewTestClient returns a mock of *http.Client.
func NewTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestNewClient(t *testing.T) {
	// The test case also checks if the client can be work with a no-ideal URL.
	c, err := NewClient(Options{
		Host:              "https://example.com//",
		VerificationToken: "testVerify",
	})
	require.NoError(t, err)

	assert.IsType(t, &Client{}, c)
	assert.Equal(t, "https://example.com", c.url.String())
	assert.Equal(t, "testVerify", c.verificationToken)
	assert.IsType(t, &http.Client{}, c.client)
}

func TestClientURL(t *testing.T) {
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "testVerify",
	})
	require.NoError(t, err)

	assert.Equal(t, "https://example.com/test-url", c.URL("test-url").String())
}
